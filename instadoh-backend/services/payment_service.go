package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"instadoh-backend/config"
	"instadoh-backend/models"
	"instadoh-backend/types"

	"gorm.io/gorm"
)

// PaymentService handles all payment processing business logic
type PaymentService struct {
	db              *gorm.DB
	lnd             *LNDService
	exchangeService *ExchangeService
	lnAddress       *LightningAddressService
	cfg             *config.Config
}

// NewPaymentService creates a new payment service
func NewPaymentService(db *gorm.DB, lnd *LNDService, exchangeService *ExchangeService, lnAddress *LightningAddressService, cfg *config.Config) *PaymentService {
	return &PaymentService{
		db:              db,
		lnd:             lnd,
		exchangeService: exchangeService,
		lnAddress:       lnAddress,
		cfg:             cfg,
	}
}

// CreateInvoice creates a Lightning invoice for receiving money
// The user specifies amount in their local currency, system handles BTC conversion
func (s *PaymentService) CreateInvoice(userID uint, req *types.CreateInvoiceRequest) (*types.InvoiceResponse, error) {
	user, err := s.getUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Get BTC/USD price
	btcPrice, err := s.exchangeService.GetBTCUSDPrice()
	if err != nil {
		return nil, fmt.Errorf("failed to get BTC price: %w", err)
	}

	// Get exchange rate from user's currency to USD
	var exchangeRateToUSD float64
	if req.Currency == "USD" {
		exchangeRateToUSD = 1.0
	} else {
		rate, err := s.exchangeService.GetRate(req.Currency)
		if err != nil {
			return nil, fmt.Errorf("failed to get rate for %s: %w", req.Currency, err)
		}
		exchangeRateToUSD = rate
	}

	// Convert to USD first, then to BTC
	// exchangeRateToUSD is "how many units of this currency per 1 USD" (e.g., KES=150)
	// So to convert KES to USD: amount / rate = USD
	var usdAmount float64
	if req.Currency == "USD" {
		usdAmount = req.Amount
	} else {
		usdAmount = req.Amount / exchangeRateToUSD
	}

	// Convert USD to millisatoshis
	btcPortion := usdAmount / btcPrice
	amountMsat := int64(btcPortion * 1e11)

	if amountMsat <= 0 {
		return nil, fmt.Errorf("amount too small after conversion")
	}

	// Create the Lightning invoice
	lnInvoice, err := s.lnd.CreateInvoice(amountMsat, req.Description, req.ExpirySeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to create lightning invoice: %w", err)
	}

	paymentHash := hex.EncodeToString(lnInvoice.RHash)
	paymentRequest := lnInvoice.PaymentRequest
	expiry := time.Now().Add(time.Duration(req.ExpirySeconds) * time.Second)
	if req.ExpirySeconds <= 0 {
		expiry = time.Now().Add(1 * time.Hour)
	}

	// Record the transaction in the database
	tx := &models.Transaction{
		UserID:         user.ID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		AmountBTC:      amountMsat,
		Direction:      types.DirectionIncoming,
		Status:         types.TxStatusPending,
		PaymentHash:    paymentHash,
		PaymentRequest: paymentRequest,
		Description:    req.Description,
		ExchangeRate:   exchangeRateToUSD,
	}

	if err := s.db.Create(tx).Error; err != nil {
		return nil, fmt.Errorf("failed to save transaction: %w", err)
	}

	return &types.InvoiceResponse{
		ID:             tx.ID,
		PaymentRequest: paymentRequest,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Description:    req.Description,
		Status:         string(tx.Status),
		ExpiresAt:      expiry,
		CreatedAt:      tx.CreatedAt,
	}, nil
}

// SendPayment sends a Lightning payment on behalf of a user
// The user pays in their local currency; the system converts and settles over
// Lightning instantly. The user never deals with BTC directly.
func (s *PaymentService) SendPayment(userID uint, req *types.SendPaymentRequest) (*types.TransactionResponse, error) {
	user, err := s.getUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Decode the payment request to validate it and get the amount
	decoded, err := s.lnd.DecodePaymentRequest(req.Invoice)
	if err != nil {
		return nil, fmt.Errorf("invalid payment request: %w", err)
	}

	// Get BTC/USD price for conversion
	btcPrice, err := s.exchangeService.GetBTCUSDPrice()
	if err != nil {
		return nil, fmt.Errorf("failed to get BTC price: %w", err)
	}

	// Get the rate for the user's currency relative to USD
	var exchangeRateToUSD float64
	if req.Currency == "USD" {
		exchangeRateToUSD = 1.0
	} else {
		rate, err := s.exchangeService.GetRate(req.Currency)
		if err != nil {
			return nil, fmt.Errorf("failed to get rate for %s: %w", req.Currency, err)
		}
		exchangeRateToUSD = rate
	}

	// Determine the amount to pay in millisatoshis.
	// If the invoice has a fixed amount, respect it; otherwise use the amount
	// requested by the user in their local currency.
	var amountMsat int64
	var amountInCurrency float64
	if decoded.AmountMsat > 0 {
		amountMsat = decoded.AmountMsat
		// Convert the invoice's msat value back to the user's local currency.
		usdValue := float64(amountMsat) / 1e11 * btcPrice
		if req.Currency == "USD" {
			amountInCurrency = usdValue
		} else {
			amountInCurrency = usdValue * exchangeRateToUSD
		}
	} else {
		// Zero-amount invoice: pay whatever the user specified in local currency.
		var usdAmount float64
		if req.Currency == "USD" {
			usdAmount = req.Amount
		} else {
			usdAmount = req.Amount / exchangeRateToUSD
		}
		btcPortion := usdAmount / btcPrice
		amountMsat = int64(btcPortion * 1e11)
		amountInCurrency = req.Amount
	}

	if amountMsat <= 0 {
		return nil, fmt.Errorf("amount too small after conversion")
	}

	// For fixed-amount invoices, respect the invoice amount in the user's
	// currency rather than the (possibly incompatible) requested amount.
	if decoded.AmountMsat > 0 {
		req.Amount = math.Round(amountInCurrency*100) / 100
	}

	// Check user balance
	if user.Balance < req.Amount {
		return nil, fmt.Errorf("insufficient balance: have %.2f, need %.2f", user.Balance, req.Amount)
	}

	// Send the payment through Lightning Network
	resp, err := s.lnd.PayInvoice(req.Invoice, amountMsat)
	if err != nil {
		return nil, fmt.Errorf("payment failed: %w", err)
	}

	paymentHash := hex.EncodeToString(resp.PaymentHash)
	preimage := hex.EncodeToString(resp.Preimage)

	// Start a database transaction
	txErr := s.db.Transaction(func(dbTx *gorm.DB) error {
		// Deduct from user balance
		if err := dbTx.Model(&user).Update("balance", gorm.Expr("balance - ?", req.Amount)).Error; err != nil {
			return fmt.Errorf("failed to update balance: %w", err)
		}

		// Record the transaction
		tx := &models.Transaction{
			UserID:         user.ID,
			Amount:         req.Amount,
			Currency:       req.Currency,
			AmountBTC:      amountMsat,
			Direction:      types.DirectionOutgoing,
			Status:         types.TxStatusCompleted,
			PaymentHash:    paymentHash,
			PaymentRequest: req.Invoice,
			Preimage:       preimage,
			Counterparty:   decoded.Destination,
			Description:    decoded.Description,
			FeeMsat:        resp.FeeMsat,
			ExchangeRate:   exchangeRateToUSD,
			SettledAt:      timePtr(time.Now()),
		}

		if err := dbTx.Create(tx).Error; err != nil {
			return fmt.Errorf("failed to save transaction: %w", err)
		}

		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	// Fetch the created transaction
	var savedTx models.Transaction
	s.db.Where("payment_hash = ?", paymentHash).First(&savedTx)

	respTx := savedTx.ToResponse()
	return &respTx, nil
}

// GetTransaction returns details of a specific transaction
func (s *PaymentService) GetTransaction(userID uint, txID uint) (*types.TransactionResponse, error) {
	var tx models.Transaction
	if err := s.db.Where("id = ? AND user_id = ?", txID, userID).First(&tx).Error; err != nil {
		return nil, fmt.Errorf("transaction not found: %w", err)
	}

	// If status is pending, check with LND for updates
	if tx.Status == types.TxStatusPending && s.lnd.IsConnected() {
		lnInvoice, err := s.lnd.GetInvoiceStatus(tx.PaymentHash)
		if err == nil {
			if IsInvoiceSettled(lnInvoice) && tx.Direction == types.DirectionIncoming {
				tx.Status = types.TxStatusCompleted
				now := time.Now()
				tx.SettledAt = &now
				tx.Preimage = hex.EncodeToString(lnInvoice.RPreimage)
				s.db.Save(&tx)

				// Credit the user's balance
				s.db.Model(&models.User{}).Where("id = ?", tx.UserID).
					Update("balance", gorm.Expr("balance + ?", tx.Amount))
			}
		}
	}

	resp := tx.ToResponse()
	return &resp, nil
}

// ListTransactions returns all transactions for a user
func (s *PaymentService) ListTransactions(userID uint, page, limit int) ([]types.TransactionResponse, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	var total int64
	s.db.Model(&models.Transaction{}).Where("user_id = ?", userID).Count(&total)

	var txs []models.Transaction
	if err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&txs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	responses := make([]types.TransactionResponse, len(txs))
	for i, tx := range txs {
		responses[i] = tx.ToResponse()
	}

	return responses, total, nil
}

// GetBalance returns the user's current balance
func (s *PaymentService) GetBalance(userID uint) (*types.BalanceResponse, error) {
	user, err := s.getUserByID(userID)
	if err != nil {
		return nil, err
	}

	return &types.BalanceResponse{
		Currency: user.Currency,
		Balance:  user.Balance,
	}, nil
}

// ResolveRecipient looks up an InstaDoh user by email and returns the info a
// sender needs to preview/confirm a payment to that user. The recipient's email
// is the universal identifier for sending money — no phone number required.
func (s *PaymentService) ResolveRecipient(email string) (*models.User, error) {
	if email == "" {
		return nil, fmt.Errorf("recipient email is required")
	}

	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no InstaDoh user found with email %s", email)
		}
		return nil, fmt.Errorf("failed to look up recipient: %w", err)
	}

	if !user.IsActive {
		return nil, fmt.Errorf("recipient account is deactivated")
	}

	return &user, nil
}

// SendByEmail sends money to another InstaDoh user identified by their email
// address. The email is the universal, country-agnostic identifier — the
// sender never needs a phone number, and the recipient's local currency and
// country come from their account profile.
func (s *PaymentService) SendByEmail(senderID uint, req *types.SendByEmailRequest) (*models.Transaction, error) {
	recipient, err := s.ResolveRecipient(req.RecipientEmail)
	if err != nil {
		return nil, err
	}
	if recipient.ID == senderID {
		return nil, fmt.Errorf("you cannot send money to yourself")
	}
	return s.internalTransfer(senderID, recipient.Email, req.Amount, req.Currency, req.Description)
}

// internalTransfer atomically debits one InstaDoh user and credits another, in
// the recipient's local currency. Used for email-to-email sends and internal
// Lightning Address pays. Reuses the underlying exchange rates for conversion.
func (s *PaymentService) internalTransfer(senderID uint, recipientEmail string, amount float64, currency, description string) (*models.Transaction, error) {
	sender, err := s.getUserByID(senderID)
	if err != nil {
		return nil, fmt.Errorf("sender not found: %w", err)
	}

	recipient, err := s.ResolveRecipient(recipientEmail)
	if err != nil {
		return nil, err
	}

	if sender.Email == recipient.Email {
		return nil, fmt.Errorf("you cannot send money to yourself")
	}

	// Validate sender's currency matches request
	if sender.Currency != currency {
		return nil, fmt.Errorf("you have %s balance, not %s", sender.Currency, currency)
	}

	// Check balance (including 2% fee)
	fee := math.Round(amount*0.02*100) / 100
	totalDeduction := amount + fee
	if sender.Balance < totalDeduction {
		return nil, fmt.Errorf("insufficient balance: have %.2f %s, need %.2f %s (including %.2f fee)",
			sender.Balance, sender.Currency, totalDeduction, sender.Currency, fee)
	}

	// Convert the send amount into the recipient's currency.
	var receiveAmount float64
	if sender.Currency == recipient.Currency {
		receiveAmount = amount
	} else {
		usdValue, err := s.exchangeService.Convert(amount, sender.Currency, "USD")
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s to USD: %w", sender.Currency, err)
		}
		receiveAmount, err = s.exchangeService.Convert(usdValue, "USD", recipient.Currency)
		if err != nil {
			return nil, fmt.Errorf("failed to convert USD to %s: %w", recipient.Currency, err)
		}
	}
	receiveAmount = math.Round(receiveAmount*100) / 100

	amountMsat, err := s.fiatToMsat(amount, sender.Currency)
	if err != nil {
		return nil, err
	}

	// Generate a settlement hash representing the Lightning leg.
	paymentHash := make([]byte, 32)
	if _, err := rand.Read(paymentHash); err != nil {
		return nil, fmt.Errorf("failed to generate settlement hash: %w", err)
	}
	paymentHashHex := hex.EncodeToString(paymentHash)
	now := time.Now()

	// Atomically debit the sender and credit the recipient.
	var sentTx *models.Transaction
	txErr := s.db.Transaction(func(dbTx *gorm.DB) error {
		// Sender's outgoing transaction
		sent := &models.Transaction{
			UserID:       sender.ID,
			Amount:       amount,
			Currency:     sender.Currency,
			AmountBTC:    amountMsat,
			Direction:    types.DirectionOutgoing,
			Status:       types.TxStatusCompleted,
			PaymentHash:  paymentHashHex,
			Counterparty: recipient.Email,
			Description:  description,
			FeeMsat:      int64(float64(amountMsat) * 0.02),
			SettledAt:    &now,
		}
		if err := dbTx.Create(sent).Error; err != nil {
			return fmt.Errorf("failed to record outgoing transaction: %w", err)
		}

		// Recipient's incoming transaction
		received := &models.Transaction{
			UserID:         recipient.ID,
			Amount:         receiveAmount,
			Currency:       recipient.Currency,
			AmountBTC:      amountMsat,
			Direction:      types.DirectionIncoming,
			Status:         types.TxStatusCompleted,
			PaymentHash:    paymentHashHex,
			CounterpartyID: uintPtr(sender.ID),
			Counterparty:   sender.Email,
			Description:    description,
			SettledAt:      &now,
		}
		if err := dbTx.Create(received).Error; err != nil {
			return fmt.Errorf("failed to record incoming transaction: %w", err)
		}

		// Debit sender (amount + fee)
		if err := dbTx.Model(&models.User{}).Where("id = ?", sender.ID).
			Update("balance", gorm.Expr("balance - ?", totalDeduction)).Error; err != nil {
			return fmt.Errorf("failed to debit sender balance: %w", err)
		}

		// Credit recipient
		if err := dbTx.Model(&models.User{}).Where("id = ?", recipient.ID).
			Update("balance", gorm.Expr("balance + ?", receiveAmount)).Error; err != nil {
			return fmt.Errorf("failed to credit recipient balance: %w", err)
		}

		sentTx = sent
		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	log.Printf("Internal payment completed: sender=%d (%s) -> recipient=%s (%s) amount=%.2f %s received=%.2f %s fee=%.2f %s",
		sender.ID, sender.Email, recipient.Email, recipient.Currency,
		amount, sender.Currency, receiveAmount, recipient.Currency, fee, sender.Currency)

	return sentTx, nil
}

// WithdrawToLightning pays out a user's local-currency balance to any Lightning
// destination in the world: a BOLT11 invoice or a Lightning Address
// (alice@example.com). This is the global cash-out path that replaces the need
// for a mobile money provider — anyone with a Lightning wallet is reachable.
func (s *PaymentService) WithdrawToLightning(userID uint, req *types.WithdrawRequest) (*models.Transaction, error) {
	user, err := s.getUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if user.Currency != req.Currency {
		return nil, fmt.Errorf("you have %s balance, not %s", user.Currency, req.Currency)
	}

	dest := strings.TrimSpace(req.Destination)
	if dest == "" {
		return nil, fmt.Errorf("destination is required")
	}

	// Internal InstaDoh Lightning Address -> direct account credit, no network
	// payment needed (a node cannot pay its own invoices).
	if IsLightningAddress(dest) && s.lnAddress.IsInternalAddress(dest) {
		if req.Amount <= 0 {
			return nil, fmt.Errorf("amount is required when withdrawing to an InstaDoh address")
		}
		var recipient models.User
		if err := s.db.Where("LOWER(ln_address) = LOWER(?)", dest).First(&recipient).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("no InstaDoh user found for lightning address %s", dest)
			}
			return nil, fmt.Errorf("failed to look up recipient: %w", err)
		}
		return s.internalTransfer(userID, recipient.Email, req.Amount, req.Currency, req.Description)
	}

	// Resolve the destination to a concrete BOLT11 invoice before paying.
	bolt11 := ""
	var decoded *DecodedPaymentRequest
	if IsBolt11Invoice(dest) {
		bolt11 = dest
		decoded, err = s.lnd.DecodePaymentRequest(bolt11)
		if err != nil {
			return nil, fmt.Errorf("invalid payment request: %w", err)
		}
	} else if IsLightningAddress(dest) {
		// Need a msat amount before we can fetch the address's invoice.
		amountMsat, convertErr := s.fiatToMsat(req.Amount, req.Currency)
		if convertErr != nil {
			return nil, convertErr
		}
		bolt11, err = s.lnAddress.FetchLnurlInvoice(dest, amountMsat, req.Description)
		if err != nil {
			return nil, err
		}
		decoded, err = s.lnd.DecodePaymentRequest(bolt11)
		if err != nil {
			return nil, fmt.Errorf("invalid invoice from %s: %w", dest, err)
		}
	} else {
		return nil, fmt.Errorf("destination must be a BOLT11 invoice or a Lightning Address")
	}

	// Determine the amount to pay. Fixed-amount invoices win; otherwise use the
	// user-specified amount in their local currency.
	var amountMsat int64
	var amountInCurrency float64
	if decoded.AmountMsat > 0 {
		amountMsat = decoded.AmountMsat
		amountInCurrency, err = s.msatToFiat(amountMsat, req.Currency)
		if err != nil {
			return nil, err
		}
		amountInCurrency = math.Round(amountInCurrency*100) / 100
	} else {
		if req.Amount <= 0 {
			return nil, fmt.Errorf("amount is required for this invoice")
		}
		amountMsat, err = s.fiatToMsat(req.Amount, req.Currency)
		if err != nil {
			return nil, err
		}
		amountInCurrency = req.Amount
	}

	if amountMsat <= 0 {
		return nil, fmt.Errorf("amount too small after conversion")
	}

	// Check balance for the resolved amount.
	if user.Balance < amountInCurrency {
		return nil, fmt.Errorf("insufficient balance: have %.2f %s, need %.2f %s",
			user.Balance, user.Currency, amountInCurrency, req.Currency)
	}

	// Execute the Lightning payment.
	resp, err := s.lnd.PayInvoice(bolt11, amountMsat)
	if err != nil {
		return nil, fmt.Errorf("payment failed: %w", err)
	}

	paymentHash := hex.EncodeToString(resp.PaymentHash)
	preimage := hex.EncodeToString(resp.Preimage)
	now := time.Now()

	var savedTx *models.Transaction
	txErr := s.db.Transaction(func(dbTx *gorm.DB) error {
		if err := dbTx.Model(&models.User{}).Where("id = ?", user.ID).
			Update("balance", gorm.Expr("balance - ?", amountInCurrency)).Error; err != nil {
			return fmt.Errorf("failed to update balance: %w", err)
		}

		tx := &models.Transaction{
			UserID:         user.ID,
			Amount:         amountInCurrency,
			Currency:       req.Currency,
			AmountBTC:      amountMsat,
			Direction:      types.DirectionOutgoing,
			Status:         types.TxStatusCompleted,
			PaymentHash:    paymentHash,
			PaymentRequest: bolt11,
			Preimage:       preimage,
			Counterparty:   decoded.Destination,
			Description:    req.Description,
			FeeMsat:        resp.FeeMsat,
			SettledAt:      &now,
		}
		if err := dbTx.Create(tx).Error; err != nil {
			return fmt.Errorf("failed to save transaction: %w", err)
		}
		savedTx = tx
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	log.Printf("Lightning withdrawal completed: user=%d dest=%s amount=%.2f %s msat=%d",
		user.ID, dest, amountInCurrency, req.Currency, amountMsat)

	return savedTx, nil
}

// fiatToMsat converts an amount in the given currency into millisatoshis.
func (s *PaymentService) fiatToMsat(amount float64, currency string) (int64, error) {
	btcPrice, err := s.exchangeService.GetBTCUSDPrice()
	if err != nil {
		return 0, fmt.Errorf("failed to get BTC price: %w", err)
	}
	var usdAmount float64
	if currency == "USD" {
		usdAmount = amount
	} else {
		rate, err := s.exchangeService.GetRate(currency)
		if err != nil {
			return 0, fmt.Errorf("failed to get rate for %s: %w", currency, err)
		}
		usdAmount = amount / rate
	}
	msat := int64((usdAmount / btcPrice) * 1e11)
	if msat <= 0 {
		return 0, fmt.Errorf("amount too small after conversion")
	}
	return msat, nil
}

// msatToFiat converts millisatoshis back into the given currency.
func (s *PaymentService) msatToFiat(msat int64, currency string) (float64, error) {
	btcPrice, err := s.exchangeService.GetBTCUSDPrice()
	if err != nil {
		return 0, fmt.Errorf("failed to get BTC price: %w", err)
	}
	usdValue := float64(msat) / 1e11 * btcPrice
	if currency == "USD" {
		return usdValue, nil
	}
	rate, err := s.exchangeService.GetRate(currency)
	if err != nil {
		return 0, fmt.Errorf("failed to get rate for %s: %w", currency, err)
	}
	return usdValue * rate, nil
}

// CreateLNURLPInvoice creates a Lightning invoice for an inbound payment
// initiated via a Lightning Address (lnurl-pay). The amount is in millisatoshis
// exactly as requested by the payer; the user's local-currency amount that will
// be credited on settlement is derived from the current exchange rate.
func (s *PaymentService) CreateLNURLPInvoice(userID uint, amountMsat int64, comment string) (*types.InvoiceResponse, error) {
	user, err := s.getUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	fiatAmount, err := s.msatToFiat(amountMsat, user.Currency)
	if err != nil {
		return nil, err
	}
	fiatAmount = math.Round(fiatAmount*100) / 100
	if fiatAmount <= 0 {
		return nil, fmt.Errorf("amount too small to credit in %s", user.Currency)
	}

	lnInvoice, err := s.lnd.CreateInvoice(amountMsat, comment, 3600)
	if err != nil {
		return nil, fmt.Errorf("failed to create lightning invoice: %w", err)
	}

	paymentHash := hex.EncodeToString(lnInvoice.RHash)
	tx := &models.Transaction{
		UserID:         user.ID,
		Amount:         fiatAmount,
		Currency:       user.Currency,
		AmountBTC:      amountMsat,
		Direction:      types.DirectionIncoming,
		Status:         types.TxStatusPending,
		PaymentHash:    paymentHash,
		PaymentRequest: lnInvoice.PaymentRequest,
		Description:    comment,
	}

	if err := s.db.Create(tx).Error; err != nil {
		return nil, fmt.Errorf("failed to save transaction: %w", err)
	}

	return &types.InvoiceResponse{
		ID:             tx.ID,
		PaymentRequest: lnInvoice.PaymentRequest,
		Amount:         fiatAmount,
		Currency:       user.Currency,
		Description:    comment,
		Status:         string(tx.Status),
		ExpiresAt:      time.Now().Add(1 * time.Hour),
		CreatedAt:      tx.CreatedAt,
	}, nil
}

// HandleInvoiceSettled processes a webhook or subscription notification that an
// invoice was settled. It credits the recipient's local-currency balance the
// instant settlement is detected. Idempotent: already-completed transactions are
// ignored so subscribers and webhooks cannot double-credit.
func (s *PaymentService) HandleInvoiceSettled(paymentHash string, preimage string, settledAmt int64) error {
	if paymentHash == "" {
		return fmt.Errorf("payment hash is empty")
	}

	var tx models.Transaction
	if err := s.db.Where("payment_hash = ?", paymentHash).First(&tx).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// The shared LND node may settle invoices that we did not create
			// (e.g. wallet-to-wallet). Nothing to credit.
			log.Printf("Settled invoice %s is not an InstaDoh transaction, ignoring", paymentHash)
			return nil
		}
		return fmt.Errorf("transaction fetch failed for payment hash %s: %w", paymentHash, err)
	}

	// Already settled — avoid double-crediting.
	if tx.Status == types.TxStatusCompleted && tx.Direction == types.DirectionIncoming {
		return nil
	}

	now := time.Now()
	tx.Status = types.TxStatusCompleted
	tx.SettledAt = &now
	if preimage != "" {
		tx.Preimage = preimage
	}

	if err := s.db.Save(&tx).Error; err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	// Only credit for incoming transactions.
	if tx.Direction != types.DirectionIncoming {
		log.Printf("Settled invoice %s is direction=%s, no credit applied", paymentHash, tx.Direction)
		return nil
	}

	// Credit the user's balance (amount in their local currency)
	user, err := s.getUserByID(tx.UserID)
	if err != nil {
		return err
	}

	user.Balance += tx.Amount
	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to credit user balance: %w", err)
	}

	log.Printf("Invoice settled: payment_hash=%s, user=%d, credited=%.2f %s",
		paymentHash, tx.UserID, tx.Amount, tx.Currency)

	return nil
}

// GetStats returns payment statistics for a user
func (s *PaymentService) GetStats(userID uint) (map[string]interface{}, error) {
	var stats struct {
		TotalReceived float64
		TotalSent     float64
		TxCount       int64
		PendingCount  int64
	}

	s.db.Model(&models.Transaction{}).
		Select("COALESCE(SUM(CASE WHEN direction = 'incoming' AND status = 'completed' THEN amount ELSE 0 END), 0) as total_received, COALESCE(SUM(CASE WHEN direction = 'outgoing' AND status = 'completed' THEN amount ELSE 0 END), 0) as total_sent, COUNT(*) as tx_count, COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) as pending_count").
		Where("user_id = ?", userID).
		Scan(&stats)

	return map[string]interface{}{
		"total_received":    stats.TotalReceived,
		"total_sent":        stats.TotalSent,
		"transaction_count": stats.TxCount,
		"pending_count":     stats.PendingCount,
	}, nil
}

func (s *PaymentService) getUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func uintPtr(v uint) *uint {
	return &v
}
