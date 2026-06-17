package handlers

import (
	"net/http"

	"instadoh-backend/config"
	"instadoh-backend/middleware"
	"instadoh-backend/models"
	"instadoh-backend/types"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserHandler handles user-related endpoints
type UserHandler struct {
	db   *gorm.DB
	cfg  *config.Config
	auth *middleware.AuthMiddleware
}

// NewUserHandler creates a new user handler
func NewUserHandler(db *gorm.DB, cfg *config.Config, auth *middleware.AuthMiddleware) *UserHandler {
	return &UserHandler{
		db:   db,
		cfg:  cfg,
		auth: auth,
	}
}

// Register handles user registration
func (h *UserHandler) Register(c *gin.Context) {
	var req types.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	if req.Role == "" {
		req.Role = types.RoleUser
	}

	if req.Role != types.RoleUser && req.Role != types.RoleMerchant {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid role. Must be 'user' or 'merchant'",
		})
		return
	}

	var country models.Country
	if err := h.db.Where("code = ?", req.CountryCode).First(&country).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Unsupported country code: " + req.CountryCode,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to validate country",
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to process password",
		})
		return
	}

	user := &models.User{
		Email:       req.Email,
		Phone:       req.Phone,
		Password:    string(hashedPassword),
		FullName:    req.FullName,
		CountryCode: req.CountryCode,
		Currency:    country.Currency,
		Role:        req.Role,
		IsActive:    true,
	}

	if err := h.db.Create(user).Error; err != nil {
		if isDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, types.ErrorResponse{
				Code:    http.StatusConflict,
				Message: "Email or phone already registered",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create user",
		})
		return
	}

	token, err := h.auth.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to generate token",
		})
		return
	}

	c.JSON(http.StatusCreated, types.AuthResponse{
		Token: token,
		User:  user.ToResponse(),
	})
}

// Login handles user authentication
func (h *UserHandler) Login(c *gin.Context) {
	var req types.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "Invalid email or password",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Login failed",
		})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, types.ErrorResponse{
			Code:    http.StatusForbidden,
			Message: "Account is deactivated",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Invalid email or password",
		})
		return
	}

	token, err := h.auth.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, types.AuthResponse{
		Token: token,
		User:  user.ToResponse(),
	})
}

// GetProfile returns the current user's profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// GetCountries returns all supported countries
func (h *UserHandler) GetCountries(c *gin.Context) {
	var countries []models.Country
	if err := h.db.Where("is_active = ?", true).Find(&countries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to fetch countries",
		})
		return
	}

	c.JSON(http.StatusOK, countries)
}

// RefreshToken refreshes the authentication token
func (h *UserHandler) RefreshToken(c *gin.Context) {
	// Reuse the middleware's extractToken function which handles Authorization header,
	// query params, and cookies
	tokenString, err := middleware.ExtractToken(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Token is required",
		})
		return
	}

	newToken, err := h.auth.RefreshToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Invalid or expired token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": newToken,
	})
}

// isDuplicateKeyError checks if the error is a database duplicate key error
func isDuplicateKeyError(err error) bool {
	return err != nil && (isPostgresDuplicateKey(err) || isSQLiteDuplicateKey(err))
}

func isPostgresDuplicateKey(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate key") ||
		contains(err.Error(), "unique constraint") ||
		contains(err.Error(), "23505"))
}

func isSQLiteDuplicateKey(err error) bool {
	return err != nil && contains(err.Error(), "UNIQUE constraint failed")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}