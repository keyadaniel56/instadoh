package types

import "time"

// UserRole defines the role of a user in the system
type UserRole string

const (
	RoleUser    UserRole = "user"
	RoleMerchant UserRole = "merchant"
	RoleAdmin   UserRole = "admin"
)

// TransactionStatus represents the state of a transaction
type TransactionStatus string

const (
	TxStatusPending   TransactionStatus = "pending"
	TxStatusCompleted TransactionStatus = "completed"
	TxStatusFailed    TransactionStatus = "failed"
	TxStatusExpired   TransactionStatus = "expired"
)

// PaymentDirection indicates if payment is incoming or outgoing
type PaymentDirection string

const (
	DirectionIncoming PaymentDirection = "incoming"
	DirectionOutgoing PaymentDirection = "outgoing"
)

// --- Request DTOs ---

type RegisterRequest struct {
	Email       string   `json:"email" binding:"required,email"`
	Phone       string   `json:"phone" binding:"required"`
	Password    string   `json:"password" binding:"required,min=8"`
	FullName    string   `json:"full_name" binding:"required"`
	CountryCode string   `json:"country_code" binding:"required,len=2"`
	Role        UserRole `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type CreateInvoiceRequest struct {
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	Currency      string  `json:"currency" binding:"required,len=3"`
	Description   string  `json:"description"`
	ExpirySeconds int64   `json:"expiry_seconds"`
}

type SendPaymentRequest struct {
	Invoice       string  `json:"invoice" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	Currency      string  `json:"currency" binding:"required,len=3"`
}

type WebhookRequest struct {
	PaymentHash string `json:"payment_hash" binding:"required"`
	Status      string `json:"status" binding:"required"`
	Preimage    string `json:"preimage"`
	SettledAmt  int64  `json:"settled_amt"`
}

// --- Response DTOs ---

type AuthResponse struct {
	Token string      `json:"token"`
	User  UserResponse `json:"user"`
}

type UserResponse struct {
	ID          uint      `json:"id"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	FullName    string    `json:"full_name"`
	CountryCode string    `json:"country_code"`
	Currency    string    `json:"currency"`
	Balance     float64   `json:"balance"`
	Role        UserRole  `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

type InvoiceResponse struct {
	ID             uint      `json:"id"`
	PaymentRequest string    `json:"payment_request"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	Description    string    `json:"description"`
	Status         string    `json:"status"`
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type TransactionResponse struct {
	ID              uint                `json:"id"`
	UserID          uint                `json:"user_id"`
	Amount          float64             `json:"amount"`
	Currency        string              `json:"currency"`
	AmountBTC       int64               `json:"amount_btc"`
	Direction       PaymentDirection    `json:"direction"`
	Status          TransactionStatus   `json:"status"`
	PaymentHash     string              `json:"payment_hash"`
	PaymentRequest  string              `json:"payment_request"`
	Counterparty    string              `json:"counterparty"`
	Description     string              `json:"description"`
	CreatedAt       time.Time           `json:"created_at"`
	CompletedAt     *time.Time          `json:"completed_at,omitempty"`
}

type BalanceResponse struct {
	Currency string  `json:"currency"`
	Balance  float64 `json:"balance"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}