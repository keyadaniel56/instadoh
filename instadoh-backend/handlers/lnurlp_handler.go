package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"instadoh-backend/config"
	"instadoh-backend/models"
	"instadoh-backend/services"
	"instadoh-backend/types"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	lnurlpMinSendableMsat = 1000    // 1 sat
	lnurlpMaxSendableMsat = 1000000000000 // 1 BTC
	lnurlpCommentLimit    = 200
)

// LNURLPHandler serves the public Lightning Address (lnurl-pay) endpoints so
// any Lightning wallet in the world can pay an InstaDoh user by their address
// (e.g. bob@instadoh.com). This is the global receive path — no mobile money
// provider required.
type LNURLPHandler struct {
	db            *gorm.DB
	cfg           *config.Config
	paymentSvc    *services.PaymentService
}

// NewLNURLPHandler creates the lnurl-pay handler.
func NewLNURLPHandler(db *gorm.DB, cfg *config.Config, paymentSvc *services.PaymentService) *LNURLPHandler {
	return &LNURLPHandler{
		db:         db,
		cfg:        cfg,
		paymentSvc: paymentSvc,
	}
}

// GetLNURLP serves GET /.well-known/lnurlp/:username — the LUD-16 metadata
// document describing how to pay a user's Lightning Address.
func (h *LNURLPHandler) GetLNURLP(c *gin.Context) {
	username := strings.ToLower(strings.TrimSpace(c.Param("username")))
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "ERROR", "reason": "username required"})
		return
	}

	address := username + "@" + h.cfg.App.LNURLDomain

	var user models.User
	if err := h.db.Where("LOWER(ln_address) = LOWER(?)", address).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"status": "ERROR", "reason": "lightning address not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": "ERROR", "reason": "internal error"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusNotFound, gin.H{"status": "ERROR", "reason": "account deactivated"})
		return
	}

	callback := fmt.Sprintf("%s/api/v1/lnurlp/callback?user_id=%d", h.cfg.App.BaseURL, user.ID)

	c.JSON(http.StatusOK, gin.H{
		"callback":        callback,
		"maxSendable":     lnurlpMaxSendableMsat,
		"minSendable":     lnurlpMinSendableMsat,
		"metadata":        fmt.Sprintf(`[["text/plain","%s"]`, user.FullName) + `]`,
		"commentAllowed":  lnurlpCommentLimit,
		"tag":             "payRequest",
	})
}

// Callback is the lnurl-pay callback that mints an invoice for the requested
// amount and returns it to the payer's wallet.
func (h *LNURLPHandler) Callback(c *gin.Context) {
	userIDStr := c.Query("user_id")
	amountStr := c.Query("amount")
	comment := c.Query("comment")

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "ERROR", "reason": "invalid user_id"})
		return
	}
	amountMsat, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amountMsat < lnurlpMinSendableMsat {
		c.JSON(http.StatusBadRequest, gin.H{"status": "ERROR", "reason": "invalid amount"})
		return
	}
	if amountMsat > lnurlpMaxSendableMsat {
		c.JSON(http.StatusBadRequest, gin.H{"status": "ERROR", "reason": "amount exceeds maximum"})
		return
	}
	if len(comment) > lnurlpCommentLimit {
		comment = comment[:lnurlpCommentLimit]
	}

	invoice, err := h.paymentSvc.CreateLNURLPInvoice(uint(userID), amountMsat, comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "ERROR", "reason": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pr":     invoice.PaymentRequest,
		"routes": []any{},
	})
}

// GetConfig reports which on/off-ramps are available. Lightning is always the
// global rail; M-Pesa and Uganda mobile are optional, region-specific extras.
func (h *LNURLPHandler) GetConfig(c *gin.Context) {
	mpesaEnabled := h.cfg.Mpesa.ConsumerKey != ""
	ugandaEnabled := h.cfg.UgandaMobile.APIUsername != "" && h.cfg.UgandaMobile.APIPassword != ""

	c.JSON(http.StatusOK, types.FeatureConfig{
		LightningEnabled:    true,
		LNURLDomain:         h.cfg.App.LNURLDomain,
		MpesaEnabled:        mpesaEnabled,
		UgandaMobileEnabled: ugandaEnabled,
	})
}