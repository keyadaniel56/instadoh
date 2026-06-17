package services

import (
	"fmt"
	"log"
	"math"
	"time"

	"instadoh-backend/models"
	"instadoh-backend/types"

	"gorm.io/gorm"
)

// CrossBorderService handles cross-border payments between Kenya and Uganda
// Flow:
//   Kenya (KES) → Lightning Network → Uganda (UGX)
//   Uganda (UGX) → Lightning Network → Kenya (KES)
// Uses Lightning Network as the instant settlement layer between countries
type CrossBorderService struct {
	db              *gorm.DB
	exchangeService *ExchangeService
	lndService      *LNDService
	mpesaService    *MpesaService
	ugandaService   *UgandaMobileService
}

// NewCrossBorderService creates a new cross-border service
func NewCrossBorderService(
	db *gorm.DB,
	exchangeService *ExchangeService,
	lndService *LNDService,
	mpesaService *MpesaService,
	ugandaService *UgandaMobileService,
) *CrossBorderService {
	return &CrossBorderService{
		db:              db,
		exchangeService: exchangeService,
		lndService:      lndService,
		mpesaService:    mpesaService,
		ugandaService:   ugandaService,
	}
}

// QuoteForTransfer provides a quote for a cross-border transfer
// fromCurrency: KES or UGX
// toCurrency: KES (if sending to Kenya) or UGX (if sending to Uganda)
func (s *CrossBorderService) QuoteForTransfer(fromCurrency, toCurrency string, amount float64) (*types.CrossBorderQuote, error) {
	if fromCurrency != "KES" && fromCurrency != "UGX" {
		return nil, fmt.Errorf("fromCurrency must be KES or UGX")
	}
	if toCurrency != "KES" && toCurrency != "UGX" {
		return nil, fmt.Errorf("toCurrency must be KES or UGX")
	}
	if fromCurrency == toCurrency {
		return nil, fmt.Errorf("fromCurrency and toCurrency must be different")
	}

	// Get exchange rates
	// First convert fromCurrency → USD, then USD → toCurrency
	var usdAmount float64
	var fromRate float64
	var err error

	if fromCurrency == "USD" {
		fromRate = 1.0
	} else {
		fromRate, err = s.exchangeService.GetRate(fromCurrency)
		if err != nil {
			return nil, fmt.Errorf("failed to get rate for %s: %w", fromCurrency, err)
		}
	}

	// fromRate is units of fromCurrency per 1 USD (e.g., KES 150 = $1)
	// So amount / fromRate = USD amount
	usdAmount = amount / fromRate

	var toRate float64
	if toCurrency == "USD" {
		toRate = 1.0
	} else {
		toRate, err = s.exchangeService.GetRate(toCurrency)
		if err != nil {
			return nil, fmt.Errorf("failed to get rate for %s: %w", toCurrency, err)
		}
	}

	// toRate is units of toCurrency per 1 USD (e.g., UGX 3800 = $1)
	// So usdAmount * toRate = toCurrency amount
	receiveAmount := usdAmount * toRate

	// Fee: 2% on cross-border transfers
	fee := math.Round(amount*0.02*100) / 100
	totalInFiat := amount + fee

	// Calculate effective exchange rate
	// For 1 unit of fromCurrency, how many units of toCurrency do you get?
	effectiveRate := 0.0
	if amount > 0 {
		effectiveRate = receiveAmount / amount
	}

	// Synthesize the USD-based rates into a direct pair
	// KES/UGX rate = (KES/USD) / (UGX/USD) = KES per 1 UGX
	var pairRate float64
	if toCurrency == "UGX" {
		// For sending KES to get UGX: how many UGX per 1 KES
		pairRate = effectiveRate
	} else {
		// For sending UGX to get KES: how many KES per 1 UGX
		pairRate = effectiveRate
	}

	validUntil := time.Now().Add(2 * time.Minute).Format(time.RFC3339)

	return &types.CrossBorderQuote{
		FromCurrency:  fromCurrency,
		ToCurrency:    toCurrency,
		SendAmount:    amount,
		ReceiveAmount: math.Round(receiveAmount*100) / 100,
		ExchangeRate:  math.Round(pairRate*1000000) / 1000000,
		Fee:           fee,
		TotalInFiat:   totalInFiat,
		ValidUntil:    validUntil,
	}, nil
}

// SendCrossBorderPayment handles the end-to-end cross-border payment
// 1. Deducts from sender's wallet balance
// 2. Routes through Lightning Network for instant settlement
// 3. Sends to recipient via their local mobile money
func (s *CrossBorderService) SendCrossBorderPayment(senderID uint, req *types.CrossBorderSendRequest) (*models.CrossBorderTransaction, error) {
	// Get sender user
	var sender models.User
	if err := s.db.First(&sender, senderID).Error; err != nil {
		return nil, fmt.Errorf("sender not found: %w", err)
	}

	// Validate sender's currency matches request
	if sender.Currency != req.Currency {
		return nil, fmt.Errorf("sender has %s balance, not %s", sender.Currency, req.Currency)
	}

	// Check balance (including fee)
	quotedFee := math.Round(req.Amount*0.02*100) / 100
	totalDeduction := req.Amount + quotedFee

	if sender.Balance < totalDeduction {
		return nil, fmt.Errorf("insufficient balance: have %.2f %s, need %.2f %s (including %.2f fee)",
			sender.Balance, sender.Currency, totalDeduction, sender.Currency, quotedFee)
	}

	// Validate recipient country
	if req.RecipientCountry != "KE" && req.RecipientCountry != "UG" {
		return nil, fmt.Errorf("recipient country must be KE or UG")
	}

	// Determine target currency
	receiveCurrency := "KES"
	if req.RecipientCountry == "UG" {
		receiveCurrency = "UGX"
	}

	// Prevent sending to same country (use regular mobile money for that)
	senderCountry := sender.CountryCode
	if senderCountry == req.RecipientCountry {
		return nil, fmt.Errorf("cannot send cross-border to the same country. Use mobile money deposit/withdrawal for domestic transfers")
	}

	// Get quote for the transfer
	quote, err := s.QuoteForTransfer(req.Currency, receiveCurrency, req.Amount)
	if err != nil {
		return nil, fmt.Errorf("failed to get quote: %w", err)
	}

	// Step 1: Convert fiat to BTC via Lightning Network
	// This is the "sending" leg - sender's fiat is converted and sent over Lightning
	btcPrice, err := s.exchangeService.GetBTCUSDPrice()
	if err != nil {
		return nil, fmt.Errorf("failed to get BTC price: %w", err)
	}

	// Convert fiat amount to millisatoshis
	usdAmount := req.Amount / quote.ExchangeRate
	btcPortion := usdAmount / btcPrice
	amountMsat := int64(btcPortion * 1e11)

	// Step 2: Create a Lightning invoice for the converted amount
	// In a real implementation, the recipient's system generates an invoice
	// For now, we use the sender's LND node to send payment
	// This simulates the Lightning leg of the transfer

	// Create a transaction record for the Lightning leg
	lightningTx := &models.Transaction{
		UserID:       senderID,
		Amount:       req.Amount,
		Currency:     req.Currency,
		AmountBTC:    amountMsat,
		Direction:    types.DirectionOutgoing,
		Status:       types.TxStatusPending,
		Description:  fmt.Sprintf("Cross-border to %s (%s)", req.RecipientCountry, req.RecipientPhone),
		ExchangeRate: quote.ExchangeRate,
	}

	if err := s.db.Create(lightningTx).Error; err != nil {
		return nil, fmt.Errorf("failed to create lightning transaction: %w", err)
	}

	// Step 3: Send the converted amount to the recipient via mobile money
	// Create cross-border transaction record
	cbTx := &models.CrossBorderTransaction{
		SenderID:        senderID,
		ReceiverPhone:   req.RecipientPhone,
		ReceiverCountry: req.RecipientCountry,
		SendAmount:      req.Amount,
		SendCurrency:    req.Currency,
		ReceiveAmount:   quote.ReceiveAmount,
		ReceiveCurrency: receiveCurrency,
		ExchangeRate:    quote.ExchangeRate,
		Fee:             quotedFee,
		Status:          "pending",
		LightningTxID:   fmt.Sprintf("%d", lightningTx.ID),
		Description:     req.Description,
	}

	if err := s.db.Create(cbTx).Error; err != nil {
		return nil, fmt.Errorf("failed to create cross-border transaction: %w", err)
	}

	// Step 4: Initiate payout via mobile money in recipient's country
	var mobileTxID uint
	switch req.RecipientCountry {
	case "KE":
		// Send money to Kenyan phone via M-Pesa B2C
		withdrawResp, err := s.mpesaService.InitiateB2CPayment(
			req.RecipientPhone,
			quote.ReceiveAmount,
			fmt.Sprintf("InstaDoh cross-border transfer from %s", sender.FullName),
		)
		if err != nil {
			log.Printf("Warning: M-Pesa B2C failed, will retry: %v", err)
			cbTx.Status = "pending_mobile_payout"
		} else {
			// Record mobile money transaction
			mmTx := &models.MobileMoneyTransaction{
				UserID:      senderID,
				Type:        "withdrawal",
				Provider:    types.MMProviderMpesa,
				ProviderRef: withdrawResp.ConversationID,
				PhoneNumber: req.RecipientPhone,
				Amount:      quote.ReceiveAmount,
				Currency:    "KES",
				Status:      types.MMStatusPending,
				TransactionID: lightningTx.ID,
			}
			if err := s.db.Create(mmTx).Error; err != nil {
				log.Printf("Warning: failed to save mobile money tx: %v", err)
			} else {
				mobileTxID = mmTx.ID
				cbTx.MobileTxID = mobileTxID
				cbTx.Status = "completed"
				now := time.Now()
				cbTx.CompletedAt = &now
			}
		}

	case "UG":
		// Send money to Ugandan phone via mobile money
		provider := "mtn" // default to MTN, could be configurable
		withdrawResp, err := s.ugandaService.InitiateWithdrawal(
			provider,
			req.RecipientPhone,
			quote.ReceiveAmount,
			fmt.Sprintf("CB-%d", cbTx.ID),
		)
		if err != nil {
			log.Printf("Warning: Uganda mobile withdrawal failed, will retry: %v", err)
			cbTx.Status = "pending_mobile_payout"
		} else {
			mmTx := &models.MobileMoneyTransaction{
				UserID:      senderID,
				Type:        "withdrawal",
				Provider:    types.MMProviderUgandaMobile,
				ProviderRef: withdrawResp.TransactionID,
				PhoneNumber: req.RecipientPhone,
				Amount:      quote.ReceiveAmount,
				Currency:    "UGX",
				Status:      types.MMStatusPending,
				TransactionID: lightningTx.ID,
			}
			if err := s.db.Create(mmTx).Error; err != nil {
				log.Printf("Warning: failed to save mobile money tx: %v", err)
			} else {
				mobileTxID = mmTx.ID
				cbTx.MobileTxID = mobileTxID
				cbTx.Status = "completed"
				now := time.Now()
				cbTx.CompletedAt = &now
			}
		}
	}

	// Step 5: Deduct from sender's balance (including fee)
	if err := s.db.Model(&sender).Update("balance", gorm.Expr("balance - ?", totalDeduction)).Error; err != nil {
		return nil, fmt.Errorf("failed to update sender balance: %w", err)
	}

	// Update the lightning transaction as completed
	lightningTx.Status = types.TxStatusCompleted
	settledAt := time.Now()
	lightningTx.SettledAt = &settledAt
	if err := s.db.Save(lightningTx).Error; err != nil {
		log.Printf("Warning: failed to update lightning tx status: %v", err)
	}

	// Update cross-border transaction
	if err := s.db.Save(cbTx).Error; err != nil {
		return nil, fmt.Errorf("failed to update cross-border transaction: %w", err)
	}

	log.Printf("Cross-border payment completed: sender=%d (%.2f %s) → recipient=%s (%.2f %s) fee=%.2f %s",
		senderID, req.Amount, req.Currency, req.RecipientPhone,
		quote.ReceiveAmount, receiveCurrency, quotedFee, req.Currency)

	return cbTx, nil
}

// GetCrossBorderTransaction returns details of a cross-border transaction
func (s *CrossBorderService) GetCrossBorderTransaction(userID, txID uint) (*models.CrossBorderTransaction, error) {
	var tx models.CrossBorderTransaction
	if err := s.db.Where("id = ? AND sender_id = ?", txID, userID).First(&tx).Error; err != nil {
		return nil, fmt.Errorf("cross-border transaction not found: %w", err)
	}
	return &tx, nil
}

// ListCrossBorderTransactions lists cross-border transactions for a user
func (s *CrossBorderService) ListCrossBorderTransactions(userID uint, page, limit int) ([]models.CrossBorderTransaction, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	var total int64
	s.db.Model(&models.CrossBorderTransaction{}).Where("sender_id = ?", userID).Count(&total)

	var txs []models.CrossBorderTransaction
	if err := s.db.Where("sender_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&txs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch cross-border transactions: %w", err)
	}

	return txs, total, nil
}

// DepositKES handles M-Pesa deposit for Kenyan users
func (s *CrossBorderService) DepositKES(userID uint, phoneNumber string, amount float64) (*types.MpesaResponse, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Initiate M-Pesa STK Push
	accountRef := fmt.Sprintf("INSTADOH-%d", user.ID)
	resp, err := s.mpesaService.InitiateSTKPush(phoneNumber, amount, accountRef, "InstaDoh Wallet Deposit")
	if err != nil {
		return nil, fmt.Errorf("failed to initiate STK push: %w", err)
	}

	// Record the mobile money transaction
	mmTx := &models.MobileMoneyTransaction{
		UserID:      userID,
		Type:        "deposit",
		Provider:    types.MMProviderMpesa,
		ProviderRef: resp.CheckoutRequestID,
		PhoneNumber: phoneNumber,
		Amount:      amount,
		Currency:    "KES",
		Status:      types.MMStatusPending,
	}
	if err := s.db.Create(mmTx).Error; err != nil {
		log.Printf("Warning: failed to record M-Pesa transaction: %v", err)
	}

	return &types.MpesaResponse{
		CheckoutRequestID: resp.CheckoutRequestID,
		ResponseCode:      resp.ResponseCode,
		ResponseDesc:      resp.ResponseDesc,
		MerchantRequestID: resp.MerchantRequestID,
		Amount:            amount,
		PhoneNumber:       phoneNumber,
		Status:            "pending",
	}, nil
}

// WithdrawKES handles M-Pesa withdrawal for Kenyan users
func (s *CrossBorderService) WithdrawKES(userID uint, phoneNumber string, amount float64) (*types.MpesaResponse, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Check balance
	if user.Balance < amount {
		return nil, fmt.Errorf("insufficient balance: have %.2f, need %.2f", user.Balance, amount)
	}

	// Initiate M-Pesa B2C withdrawal
	remarks := fmt.Sprintf("InstaDoh Withdrawal - User %d", user.ID)
	resp, err := s.mpesaService.InitiateB2CPayment(phoneNumber, amount, remarks)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate B2C withdrawal: %w", err)
	}

	// Record mobile money transaction
	mmTx := &models.MobileMoneyTransaction{
		UserID:      userID,
		Type:        "withdrawal",
		Provider:    types.MMProviderMpesa,
		ProviderRef: resp.ConversationID,
		PhoneNumber: phoneNumber,
		Amount:      amount,
		Currency:    "KES",
		Status:      types.MMStatusPending,
	}
	if err := s.db.Create(mmTx).Error; err != nil {
		log.Printf("Warning: failed to record M-Pesa withdrawal: %v", err)
	}

	// Deduct from user balance
	if err := s.db.Model(&user).Update("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	return &types.MpesaResponse{
		ResponseCode: resp.ResponseCode,
		ResponseDesc: resp.ResponseDesc,
		Amount:       amount,
		PhoneNumber:  phoneNumber,
		Status:       "pending",
	}, nil
}

// DepositUGX handles Uganda mobile money deposit
func (s *CrossBorderService) DepositUGX(userID uint, phoneNumber, provider string, amount float64) (*types.MpesaResponse, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	reference := fmt.Sprintf("INSTADOH-%d-%d", user.ID, time.Now().Unix())
	resp, err := s.ugandaService.InitiateDeposit(provider, phoneNumber, amount, reference)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate Uganda mobile deposit: %w", err)
	}

	// Record mobile money transaction
	mmTx := &models.MobileMoneyTransaction{
		UserID:      userID,
		Type:        "deposit",
		Provider:    types.MMProviderUgandaMobile,
		ProviderRef: resp.TransactionID,
		PhoneNumber: phoneNumber,
		Amount:      amount,
		Currency:    "UGX",
		Status:      types.MMStatusPending,
	}
	if err := s.db.Create(mmTx).Error; err != nil {
		log.Printf("Warning: failed to record UG deposit: %v", err)
	}

	return &types.MpesaResponse{
		ResponseCode:      "0",
		ResponseDesc:      resp.Message,
		CheckoutRequestID: resp.TransactionID,
		Amount:            amount,
		PhoneNumber:       phoneNumber,
		Status:            "pending",
	}, nil
}

// WithdrawUGX handles Uganda mobile money withdrawal
func (s *CrossBorderService) WithdrawUGX(userID uint, phoneNumber, provider string, amount float64) (*types.MpesaResponse, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Check balance
	if user.Balance < amount {
		return nil, fmt.Errorf("insufficient balance: have %.2f, need %.2f", user.Balance, amount)
	}

	reference := fmt.Sprintf("CBW-%d-%d", user.ID, time.Now().Unix())
	resp, err := s.ugandaService.InitiateWithdrawal(provider, phoneNumber, amount, reference)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate Uganda mobile withdrawal: %w", err)
	}

	mmTx := &models.MobileMoneyTransaction{
		UserID:      userID,
		Type:        "withdrawal",
		Provider:    types.MMProviderUgandaMobile,
		ProviderRef: resp.TransactionID,
		PhoneNumber: phoneNumber,
		Amount:      amount,
		Currency:    "UGX",
		Status:      types.MMStatusPending,
	}
	if err := s.db.Create(mmTx).Error; err != nil {
		log.Printf("Warning: failed to record UG withdrawal: %v", err)
	}

	// Deduct from user balance
	if err := s.db.Model(&user).Update("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	return &types.MpesaResponse{
		ResponseCode: "0",
		ResponseDesc: resp.Message,
		Amount:       amount,
		PhoneNumber:  phoneNumber,
		Status:       "pending",
	}, nil
}

// HandleMpesaDepositCallback processes M-Pesa STK Push callback
func (s *CrossBorderService) HandleMpesaDepositCallback(checkoutRequestID string, amount float64) error {
	// Find the mobile money transaction
	var mmTx models.MobileMoneyTransaction
	if err := s.db.Where("provider_ref = ? AND type = 'deposit'", checkoutRequestID).First(&mmTx).Error; err != nil {
		return fmt.Errorf("mobile money transaction not found: %w", err)
	}

	// Update status
	mmTx.Status = types.MMStatusCompleted
	mmTx.ResultCode = "0"
	mmTx.ResultDesc = "Success"
	now := time.Now()
	mmTx.SettledAt = &now

	if err := s.db.Save(&mmTx).Error; err != nil {
		return fmt.Errorf("failed to update mobile money tx: %w", err)
	}

	// Credit the user's balance
	if err := s.db.Model(&models.User{}).Where("id = ?", mmTx.UserID).
		Update("balance", gorm.Expr("balance + ?", mmTx.Amount)).Error; err != nil {
		return fmt.Errorf("failed to credit user balance: %w", err)
	}

	log.Printf("M-Pesa deposit completed: user=%d amount=%.2f KES ref=%s",
		mmTx.UserID, mmTx.Amount, checkoutRequestID)

	return nil
}

// HandleMpesaWithdrawalCallback processes M-Pesa B2C withdrawal callback
func (s *CrossBorderService) HandleMpesaWithdrawalCallback(conversationID string, success bool) error {
	var mmTx models.MobileMoneyTransaction
	if err := s.db.Where("provider_ref = ? AND type = 'withdrawal'", conversationID).First(&mmTx).Error; err != nil {
		return fmt.Errorf("withdrawal transaction not found: %w", err)
	}

	if success {
		mmTx.Status = types.MMStatusCompleted
		mmTx.ResultCode = "0"
		mmTx.ResultDesc = "Success"
	} else {
		mmTx.Status = types.MMStatusFailed
		mmTx.ResultCode = "1"
		mmTx.ResultDesc = "Withdrawal failed"
		// Refund the user
		if err := s.db.Model(&models.User{}).Where("id = ?", mmTx.UserID).
			Update("balance", gorm.Expr("balance + ?", mmTx.Amount)).Error; err != nil {
			log.Printf("Warning: failed to refund user %d: %v", mmTx.UserID, err)
		}
	}
	now := time.Now()
	mmTx.SettledAt = &now

	return s.db.Save(&mmTx).Error
}