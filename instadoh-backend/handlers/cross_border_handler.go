package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"

	"instadoh-backend/middleware"
	"instadoh-backend/services"
	"instadoh-backend/types"

	"github.com/gin-gonic/gin"
)

// CrossBorderHandler handles cross-border payment endpoints
type CrossBorderHandler struct {
	crossBorderService *services.CrossBorderService
}

// NewCrossBorderHandler creates a new cross-border handler
func NewCrossBorderHandler(crossBorderService *services.CrossBorderService) *CrossBorderHandler {
	return &CrossBorderHandler{
		crossBorderService: crossBorderService,
	}
}

// GetQuote returns a quote for a cross-border transfer
// GET /api/v1/cross-border/quote?from_currency=KES&to_currency=UGX&amount=1000
func (h *CrossBorderHandler) GetQuote(c *gin.Context) {
	fromCurrency := c.Query("from_currency")
	toCurrency := c.Query("to_currency")
	amountStr := c.Query("amount")

	if fromCurrency == "" || toCurrency == "" || amountStr == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "from_currency, to_currency, and amount are required",
		})
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid amount",
		})
		return
	}

	quote, err := h.crossBorderService.QuoteForTransfer(fromCurrency, toCurrency, amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, quote)
}

// SendCrossBorder initiates a cross-border payment
// POST /api/v1/cross-border/send
func (h *CrossBorderHandler) SendCrossBorder(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	var req types.CrossBorderSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	tx, err := h.crossBorderService.SendCrossBorderPayment(userID, &req)
	if err != nil {
		code := http.StatusBadRequest
		c.JSON(code, types.ErrorResponse{
			Code:    code,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction":     tx,
		"message":         "Cross-border transfer initiated successfully",
	})
}

// ListCrossBorderTransactions lists cross-border transactions for the user
// GET /api/v1/cross-border/transactions
func (h *CrossBorderHandler) ListCrossBorderTransactions(c *gin.Context) {
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

	txs, total, err := h.crossBorderService.ListCrossBorderTransactions(userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to fetch cross-border transactions",
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

// GetCrossBorderTransaction returns details of a specific cross-border transaction
// GET /api/v1/cross-border/transactions/:id
func (h *CrossBorderHandler) GetCrossBorderTransaction(c *gin.Context) {
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

	tx, err := h.crossBorderService.GetCrossBorderTransaction(userID, uint(txID))
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Transaction not found",
		})
		return
	}

	c.JSON(http.StatusOK, tx)
}

// DepositMpesa initiates an M-Pesa STK Push deposit for Kenyan users
// POST /api/v1/mpesa/deposit
func (h *CrossBorderHandler) DepositMpesa(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	var req types.MpesaSTKPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	resp, err := h.crossBorderService.DepositKES(userID, req.PhoneNumber, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// WithdrawMpesa initiates an M-Pesa B2C withdrawal for Kenyan users
// POST /api/v1/mpesa/withdraw
func (h *CrossBorderHandler) WithdrawMpesa(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	var req types.MpesaWithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	resp, err := h.crossBorderService.WithdrawKES(userID, req.PhoneNumber, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// MpesaCallback handles M-Pesa STK Push callbacks from Safaricom
// POST /api/v1/mpesa/callback
func (h *CrossBorderHandler) MpesaCallback(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ResultCode": 1, "ResultDesc": "Failed to read body"})
		return
	}

	var callbackData services.STKCallbackData
	if err := json.Unmarshal(body, &callbackData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ResultCode": 1, "ResultDesc": "Invalid callback data"})
		return
	}

	checkoutRequestID, amount, err := services.ProcessSTKCallbackData(&callbackData)
	if err != nil {
		// Log the error but still respond to Safaricom with success
		fmt.Printf("M-Pesa callback processing error: %v\n", err)
		c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Success"})
		return
	}

	if err := h.crossBorderService.HandleMpesaDepositCallback(checkoutRequestID, amount); err != nil {
		fmt.Printf("M-Pesa deposit callback handler error: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Success"})
}

// MpesaResultCallback handles M-Pesa B2C result callbacks
// POST /api/v1/mpesa/result
func (h *CrossBorderHandler) MpesaResultCallback(c *gin.Context) {
	// Parse B2C callback result
	var result struct {
		Result struct {
			ConversationID string `json:"ConversationID"`
			ResultCode     int    `json:"ResultCode"`
			ResultDesc     string `json:"ResultDesc"`
			TransactionID  string `json:"TransactionID"`
		} `json:"Result"`
	}

	if err := c.ShouldBindJSON(&result); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ResultCode": 1, "ResultDesc": "Invalid data"})
		return
	}

	success := result.Result.ResultCode == 0
	if err := h.crossBorderService.HandleMpesaWithdrawalCallback(result.Result.ConversationID, success); err != nil {
		fmt.Printf("M-Pesa withdrawal callback error: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Success"})
}

// DepositUgandaMobile initiates a Uganda mobile money deposit
// POST /api/v1/uganda-mobile/deposit
func (h *CrossBorderHandler) DepositUgandaMobile(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	var req types.UgandaMobileDepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	resp, err := h.crossBorderService.DepositUGX(userID, req.PhoneNumber, req.Provider, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// WithdrawUgandaMobile initiates a Uganda mobile money withdrawal
// POST /api/v1/uganda-mobile/withdraw
func (h *CrossBorderHandler) WithdrawUgandaMobile(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	var req types.UgandaMobileDepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	resp, err := h.crossBorderService.WithdrawUGX(userID, req.PhoneNumber, req.Provider, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UgandaMobileCallback handles Uganda mobile money callbacks
// POST /api/v1/uganda-mobile/callback
func (h *CrossBorderHandler) UgandaMobileCallback(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Failed to read body"})
		return
	}

	var callbackData struct {
		TransactionID string `json:"transaction_id"`
		Status        string `json:"status"`
		Reference     string `json:"reference"`
		Amount        float64 `json:"amount"`
		PhoneNumber   string `json:"phone_number"`
	}

	if err := json.Unmarshal(body, &callbackData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid callback data"})
		return
	}

	if callbackData.Status == "successful" && callbackData.Reference != "" {
		// This is a deposit callback - credit the user
		if err := h.crossBorderService.HandleMpesaDepositCallback(callbackData.TransactionID, callbackData.Amount); err != nil {
			fmt.Printf("Uganda mobile callback error: %v\n", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}