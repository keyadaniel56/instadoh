package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"instadoh-backend/middleware"
	"instadoh-backend/services"
	"instadoh-backend/types"

	"github.com/gin-gonic/gin"
)

// PaymentHandler handles payment-related endpoints
type PaymentHandler struct {
	paymentService *services.PaymentService
	exchangeService *services.ExchangeService
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(paymentService *services.PaymentService, exchangeService *services.ExchangeService) *PaymentHandler {
	return &PaymentHandler{
		paymentService:  paymentService,
		exchangeService: exchangeService,
	}
}

// CreateInvoice creates a new Lightning invoice for receiving money
// POST /api/v1/payments/invoices
func (h *PaymentHandler) CreateInvoice(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	var req types.CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	invoice, err := h.paymentService.CreateInvoice(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, invoice)
}

// SendPayment sends a Lightning payment
// POST /api/v1/payments/send
func (h *PaymentHandler) SendPayment(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	var req types.SendPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	tx, err := h.paymentService.SendPayment(userID, &req)
	if err != nil {
		code := http.StatusBadRequest
		if errMsg := err.Error(); len(errMsg) > 20 && errMsg[:20] == "insufficient balance" {
			code = http.StatusPaymentRequired
		}
		c.JSON(code, types.ErrorResponse{
			Code:    code,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, tx)
}

// GetTransaction returns details of a specific transaction
// GET /api/v1/payments/transactions/:id
func (h *PaymentHandler) GetTransaction(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	txIDStr := c.Param("id")
	txID, err := strconv.ParseUint(txIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid transaction ID",
		})
		return
	}

	tx, err := h.paymentService.GetTransaction(userID, uint(txID))
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Transaction not found",
		})
		return
	}

	c.JSON(http.StatusOK, tx)
}

// ListTransactions returns all transactions for the current user
// GET /api/v1/payments/transactions
func (h *PaymentHandler) ListTransactions(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	txs, total, err := h.paymentService.ListTransactions(userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to fetch transactions",
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"data":       txs,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
	})
}

// GetBalance returns the current user's balance
// GET /api/v1/payments/balance
func (h *PaymentHandler) GetBalance(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	balance, err := h.paymentService.GetBalance(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get balance",
		})
		return
	}

	c.JSON(http.StatusOK, balance)
}

// GetStats returns user payment statistics
// GET /api/v1/payments/stats
func (h *PaymentHandler) GetStats(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	stats, err := h.paymentService.GetStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get stats",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetExchangeRate returns current exchange rates
// GET /api/v1/payments/rates
func (h *PaymentHandler) GetExchangeRate(c *gin.Context) {
	currency := c.Query("currency")
	if currency == "" {
		currency = "USD"
	}

	rate, err := h.exchangeService.GetRate(currency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to get rate for %s", currency),
		})
		return
	}

	btcPrice, err := h.exchangeService.GetBTCUSDPrice()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get BTC price",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"currency":    currency,
		"rate_to_usd": rate,
		"btc_usd":     btcPrice,
	})
}

// ConvertCurrency converts an amount between currencies
// POST /api/v1/payments/convert
func (h *PaymentHandler) ConvertCurrency(c *gin.Context) {
	var req struct {
		Amount       float64 `json:"amount" binding:"required,gt=0"`
		FromCurrency string  `json:"from_currency" binding:"required,len=3"`
		ToCurrency   string  `json:"to_currency" binding:"required,len=3"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	result, err := h.exchangeService.Convert(req.Amount, req.FromCurrency, req.ToCurrency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Conversion failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"amount":        req.Amount,
		"from_currency": req.FromCurrency,
		"to_currency":   req.ToCurrency,
		"result":        math.Round(result*100) / 100,
	})
}

// Webhook handles LND webhook callbacks for invoice updates
// POST /api/v1/payments/webhook
func (h *PaymentHandler) Webhook(c *gin.Context) {
	var req types.WebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid webhook payload",
		})
		return
	}

	if req.Status == "settled" || req.Status == "paid" {
		if err := h.paymentService.HandleInvoiceSettled(req.PaymentHash, req.Preimage, req.SettledAmt); err != nil {
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Failed to process webhook",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}