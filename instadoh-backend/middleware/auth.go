package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"instadoh-backend/config"
	"instadoh-backend/database"
	"instadoh-backend/models"
	"instadoh-backend/types"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	cfg *config.JWTConfig
}

// Claims represents the JWT claims
type Claims struct {
	UserID uint           `json:"user_id"`
	Email  string         `json:"email"`
	Role   types.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(cfg *config.JWTConfig) *AuthMiddleware {
	return &AuthMiddleware{cfg: cfg}
}

// RequireAuth is a middleware that checks for a valid JWT token
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := ExtractToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: err.Error(),
			})
			c.Abort()
			return
		}

		claims, err := m.validateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Check if user is still active
		var user models.User
		if err := database.GetDB().First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "User not found",
			})
			c.Abort()
			return
		}

		if !user.IsActive {
			c.JSON(http.StatusForbidden, types.ErrorResponse{
				Code:    http.StatusForbidden,
				Message: "Account is deactivated",
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// RequireRole is a middleware factory that checks for specific roles
func (m *AuthMiddleware) RequireRole(roles ...types.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "Authentication required",
			})
			c.Abort()
			return
		}

		role := userRole.(types.UserRole)
		authorized := false
		for _, allowedRole := range roles {
			if role == allowedRole {
				authorized = true
				break
			}
		}

		if !authorized {
			c.JSON(http.StatusForbidden, types.ErrorResponse{
				Code:    http.StatusForbidden,
				Message: "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GenerateToken creates a new JWT token for a user
func (m *AuthMiddleware) GenerateToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(m.cfg.Expiration) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "instadoh",
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(m.cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, nil
}

// RefreshToken refreshes an existing token
func (m *AuthMiddleware) RefreshToken(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.cfg.Secret), nil
	})

	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("token is invalid")
	}

	// Generate new token with extended expiration
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Duration(m.cfg.Expiration) * time.Hour))
	claims.IssuedAt = jwt.NewNumericDate(time.Now())

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	newTokenString, err := newToken.SignedString([]byte(m.cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}

	return newTokenString, nil
}

// GetUserID extracts the user ID from the gin context
func GetUserID(c *gin.Context) (uint, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, fmt.Errorf("user not authenticated")
	}
	return userID.(uint), nil
}

func (m *AuthMiddleware) validateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.cfg.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	return claims, nil
}

// ExtractToken extracts a JWT token from the request (Authorization header, query param, or cookie)
func ExtractToken(c *gin.Context) (string, error) {
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			return "", fmt.Errorf("invalid authorization header format")
		}
		return parts[1], nil
	}

	// Try query parameter
	token := c.Query("token")
	if token != "" {
		return token, nil
	}

	// Try cookie
	token, err := c.Cookie("auth_token")
	if err == nil && token != "" {
		return token, nil
	}

	return "", fmt.Errorf("no authentication token found")
}