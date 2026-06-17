package services

import (
	"encoding/hex"
	"fmt"
	"log"
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
	cfg             *config.Config
}

// NewPaymentService creates a new payment service
func NewPaymentService(db *gorm.DB, lnd *LNDService, exchangeService *ExchangeService, cfg *config.Config) *PaymentService {
	return &PaymentService{
		db:              db,
		lnd:             lnd,
		exchangeService: exchangeService,
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
	lnInvoice, err := s.lnd.CreateInvoice(user, amountMsat, req.Description, req.ExpirySeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to create lightning invoice: %w", err)
	}

	paymentHash := hex.EncodeToString(lnInvoice.RHash)
	expiry := time.Now().Add(time.Duration(req.ExpirySeconds) * time.Second)
	if req.ExpirySeconds <= 0 {
		expiry = time.Now().Add(1 * time.Hour)
	}

	// Record the transaction in the database
	// Note: lnrpc v0.0.2 AddInvoiceResponse only has RHash, no PaymentRequest string
	tx := &models.Transaction{
		UserID:         user.ID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		AmountBTC:      amountMsat,
		Direction:      types.DirectionIncoming,
		Status:         types.TxStatusPending,
		PaymentHash:    paymentHash,
		PaymentRequest: "", // Not available in lnrpc v0.0.2
		Description:    req.Description,
		ExchangeRate:   exchangeRateToUSD,
	}

	if err := s.db.Create(tx).Error; err != nil {
		return nil, fmt.Errorf("failed to save transaction: %w", err)
	}

	return &types.InvoiceResponse{
		ID:             tx.ID,
		PaymentRequest: "", // Not available in lnrpc v0.0.2
		Amount:         req.Amount,
		Currency:       req.Currency,
		Description:    req.Description,
		Status:         string(tx.Status),
		ExpiresAt:      expiry,
		CreatedAt:      tx.CreatedAt,
	}, nil
}

// SendPayment sends a Lightning payment on behalf of a user
func (s *PaymentService) SendPayment(userID uint, req *types.SendPaymentRequest) (*types.TransactionResponse, error) {
	user, err := s.getUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Check user balance
	if user.Balance < req.Amount {
		return nil, fmt.Errorf("insufficient balance: have %.2f, need %.2f", user.Balance, req.Amount)
	}

	// Get BTC/USD price for conversion
	btcPrice, err := s.exchangeService.GetBTCUSDPrice()
	if err != nil {
		return nil, fmt.Errorf("failed to get BTC price: %w", err)
	}

	// Convert fiat amount to millisatoshis
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

	// exchangeRateToUSD is "how many units of this currency per 1 USD" (e.g., KES=150)
	// So to convert KES to USD: amount / rate = USD
	var usdAmount float64
	if req.Currency == "USD" {
		usdAmount = req.Amount
	} else {
		usdAmount = req.Amount / exchangeRateToUSD
	}
	btcPortion := usdAmount / btcPrice
	amountMsat := int64(btcPortion * 1e11)

	// Decode the payment request to validate it and get the amount
	decoded, err := s.lnd.DecodePaymentRequest(req.Invoice)
	if err != nil {
		return nil, fmt.Errorf("invalid payment request: %w", err)
	}

	// Send the payment through Lightning Network
	resp, err := s.lnd.PayInvoice(req.Invoice, amountMsat)
	if err != nil {
		return nil, fmt.Errorf("payment failed: %w", err)
	}

	paymentHash := hex.EncodeToString(resp.PaymentHash)

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
			Preimage:       "",
			Counterparty:   "",
			Description:    decoded.Description,
			FeeMsat:        0, // Fee info not available in lnrpc v0.0.2
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
			if lnInvoice.Settled {
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

// HandleInvoiceSettled processes a webhook or notification that an invoice was settled
func (s *PaymentService) HandleInvoiceSettled(paymentHash string, preimage string, settledAmt int64) error {
	var tx models.Transaction
	if err := s.db.Where("payment_hash = ?", paymentHash).First(&tx).Error; err != nil {
		return fmt.Errorf("transaction not found for payment hash %s: %w", paymentHash, err)
	}

	// Update transaction status
	now := time.Now()
	tx.Status = types.TxStatusCompleted
	tx.Preimage = preimage
	tx.SettledAt = &now

	if err := s.db.Save(&tx).Error; err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
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

	log.Printf("Invoice settled: payment_hash=%s, user=%d, amount=%.2f %s",
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
		"total_received": stats.TotalReceived,
		"total_sent":     stats.TotalSent,
		"transaction_count": stats.TxCount,
		"pending_count":  stats.PendingCount,
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

