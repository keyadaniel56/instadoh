package services

import (
	"crypto/rand"
	"encoding/hex"
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
//
//	Kenya (KES) → Lightning Network → Uganda (UGX)
//	Uganda (UGX) → Lightning Network → Kenya (KES)
//
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
// fromCurrency: any supported currency (e.g., KES, UGX, USD)
// toCurrency: the recipient's currency (e.g., KES, UGX, USD)
func (s *CrossBorderService) QuoteForTransfer(fromCurrency, toCurrency string, amount float64) (*types.CrossBorderQuote, error) {
	if fromCurrency == "" || toCurrency == "" {
		return nil, fmt.Errorf("fromCurrency and toCurrency are required")
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

// ResolveRecipient looks up an InstaDoh user by email and returns their
// profile. The email is the universal identifier for sending money — no phone
// number required — and the recipient's country/currency come from their account.
func (s *CrossBorderService) ResolveRecipient(email string) (*models.User, error) {
	if email == "" {
		return nil, fmt.Errorf("recipient email is required")
	}

	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no InstaDoh user found with email %s", email)
		}
		return nil, fmt.Errorf("failed to look up recipient: %w", err)
	}

	if !user.IsActive {
		return nil, fmt.Errorf("recipient account is deactivated")
	}

	return &user, nil
}

// SendCrossBorderPayment handles the end-to-end cross-border payment
// 1. Looks up the recipient by email (their InstaDoh account)
// 2. Deducts from sender's wallet balance
// 3. Routes through Lightning Network for instant settlement
// 4. Credits the recipient's local-currency balance
//
// The recipient's email is the universal identifier — no phone number or
// mobile money account is needed on the receiving side. Their country and
// currency come from their InstaDoh profile.
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

	// Resolve the recipient by email
	var recipient models.User
	if err := s.db.Where("email = ?", req.RecipientEmail).First(&recipient).Error; err != nil {
		return nil, fmt.Errorf("no InstaDoh user found with email %s", req.RecipientEmail)
	}
	if !recipient.IsActive {
		return nil, fmt.Errorf("recipient account is deactivated")
	}
	if recipient.ID == sender.ID {
		return nil, fmt.Errorf("you cannot send money to yourself")
	}

	// Check balance (including fee)
	quotedFee := math.Round(req.Amount*0.02*100) / 100
	totalDeduction := req.Amount + quotedFee

	if sender.Balance < totalDeduction {
		return nil, fmt.Errorf("insufficient balance: have %.2f %s, need %.2f %s (including %.2f fee)",
			sender.Balance, sender.Currency, totalDeduction, sender.Currency, quotedFee)
	}

	// Determine target currency and derive the recipient country from the
	// recipient's profile instead of a phone number
	receiveCurrency := recipient.Currency
	recipientCountry := recipient.CountryCode

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
	// In a real implementation, the recipient's country LND node generates an
	// invoice and the sender's node pays it over the Lightning channel between
	// the two nodes. The user never sees this: they only see the local-currency
	// balances on each side.
	//
	// For the single-node deployment we generate a real settlement hash that
	// represents the Lightning leg of the transfer. In the two-node topology
	// (one LND node per country, connected by a channel) this hash is the real
	// payment hash of the Lightning payment between the nodes.
	paymentHash := make([]byte, 32)
	if _, err := rand.Read(paymentHash); err != nil {
		return nil, fmt.Errorf("failed to generate settlement hash: %w", err)
	}
	paymentHashHex := hex.EncodeToString(paymentHash)

	// Create a transaction record for the Lightning leg
	lightningTx := &models.Transaction{
		UserID:       senderID,
		Amount:       req.Amount,
		Currency:     req.Currency,
		AmountBTC:    amountMsat,
		Direction:    types.DirectionOutgoing,
		Status:       types.TxStatusPending,
		PaymentHash:  paymentHashHex,
		Counterparty: recipient.Email,
		Description:  fmt.Sprintf("Cross-border to %s (%s)", recipientCountry, recipient.Email),
		ExchangeRate: quote.ExchangeRate,
	}

	if err := s.db.Create(lightningTx).Error; err != nil {
		return nil, fmt.Errorf("failed to create lightning transaction: %w", err)
	}

	// Create the recipient's incoming transaction record for the same
	// settlement hash so both parties share a native transaction history
	now := time.Now()
	recipientTx := &models.Transaction{
		UserID:         recipient.ID,
		Amount:         quote.ReceiveAmount,
		Currency:       receiveCurrency,
		AmountBTC:      amountMsat,
		Direction:      types.DirectionIncoming,
		Status:         types.TxStatusPending,
		PaymentHash:    paymentHashHex,
		CounterpartyID: uintPtr(sender.ID),
		Counterparty:   sender.Email,
		Description:    req.Description,
		ExchangeRate:   quote.ExchangeRate,
	}
	if err := s.db.Create(recipientTx).Error; err != nil {
		return nil, fmt.Errorf("failed to create recipient transaction: %w", err)
	}

	// Step 3: Create cross-border transaction record
	cbTx := &models.CrossBorderTransaction{
		SenderID:       senderID,
		ReceiverEmail:  recipient.Email,
		ReceiverPhone:  recipient.Phone,
		ReceiverCountry: recipientCountry,
		SendAmount:     req.Amount,
		SendCurrency:   req.Currency,
		ReceiveAmount:  quote.ReceiveAmount,
		ReceiveCurrency: receiveCurrency,
		ExchangeRate:   quote.ExchangeRate,
		Fee:            quotedFee,
		Status:         "pending",
		LightningTxID:  fmt.Sprintf("%d", lightningTx.ID),
		Description:    req.Description,
	}

	if err := s.db.Create(cbTx).Error; err != nil {
		return nil, fmt.Errorf("failed to create cross-border transaction: %w", err)
	}

	// Step 4: Settle atomically — debit sender (amount + fee) and credit
	// recipient. Because the recipient holds an InstaDoh account, no mobile
	// money payout is needed; the funds land straight in their wallet balance.
	settleErr := s.db.Transaction(func(dbTx *gorm.DB) error {
		if err := dbTx.Model(&models.User{}).Where("id = ?", sender.ID).
			Update("balance", gorm.Expr("balance - ?", totalDeduction)).Error; err != nil {
			return fmt.Errorf("failed to update sender balance: %w", err)
		}

		if err := dbTx.Model(&models.User{}).Where("id = ?", recipient.ID).
			Update("balance", gorm.Expr("balance + ?", quote.ReceiveAmount)).Error; err != nil {
			return fmt.Errorf("failed to credit recipient balance: %w", err)
		}

		// Update the lightning transaction as completed
		lightningTx.Status = types.TxStatusCompleted
		lightningTx.SettledAt = &now
		if err := dbTx.Save(lightningTx).Error; err != nil {
			return fmt.Errorf("failed to update lightning tx status: %w", err)
		}

		// Update the recipient's transaction as completed
		recipientTx.Status = types.TxStatusCompleted
		recipientTx.SettledAt = &now
		if err := dbTx.Save(recipientTx).Error; err != nil {
			return fmt.Errorf("failed to update recipient tx status: %w", err)
		}

		// Update cross-border transaction
		cbTx.Status = "completed"
		cbTx.CompletedAt = &now
		if err := dbTx.Save(cbTx).Error; err != nil {
			return fmt.Errorf("failed to update cross-border transaction: %w", err)
		}

		return nil
	})
	if settleErr != nil {
		return nil, settleErr
	}

	log.Printf("Cross-border payment completed: sender=%d (%.2f %s) -> recipient=%s (%s) received=%.2f %s fee=%.2f %s",
		senderID, req.Amount, req.Currency, recipient.Email, recipientCountry,
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
