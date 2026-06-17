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

// MobileMoneyProvider indicates the mobile money provider
type MobileMoneyProvider string

const (
	MMProviderMpesa       MobileMoneyProvider = "mpesa"
	MMProviderUgandaMobile MobileMoneyProvider = "uganda_mobile"
)

// MobileMoneyTransactionStatus tracks mobile money payment state
type MobileMoneyStatus string

const (
	MMStatusPending   MobileMoneyStatus = "pending"
	MMStatusCompleted MobileMoneyStatus = "completed"
	MMStatusFailed    MobileMoneyStatus = "failed"
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

// MpesaSTKPushRequest initiates an M-Pesa STK Push
type MpesaSTKPushRequest struct {
	PhoneNumber string  `json:"phone_number" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
}

// MpesaWithdrawRequest initiates an M-Pesa B2C withdrawal
type MpesaWithdrawRequest struct {
	PhoneNumber string  `json:"phone_number" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
}

// UgandaMobileDepositRequest initiates a Uganda mobile money deposit
type UgandaMobileDepositRequest struct {
	PhoneNumber string  `json:"phone_number" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Provider    string  `json:"provider" binding:"required"` // e.g., "mtn", "airtel"
}

// CrossBorderSendRequest handles sending cross-border
type CrossBorderSendRequest struct {
	RecipientPhone string  `json:"recipient_phone" binding:"required"`
	RecipientCountry string `json:"recipient_country" binding:"required,len=2"` // KE or UG
	Amount         float64 `json:"amount" binding:"required,gt=0"`
	Currency       string  `json:"currency" binding:"required,len=3"` // Sender's currency (KES or UGX)
	Description    string  `json:"description"`
}

// CrossBorderQuoteRequest gets a quote for a cross-border transfer
type CrossBorderQuoteRequest struct {
	FromCurrency string  `json:"from_currency" binding:"required,len=3"`
	ToCurrency   string  `json:"to_currency" binding:"required,len=3"`
	Amount       float64 `json:"amount" binding:"required,gt=0"`
}

// --- Response DTOs ---

type AuthResponse struct {
	Token string       `json:"token"`
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

// MpesaResponse represents the response from an M-Pesa transaction
type MpesaResponse struct {
	CheckoutRequestID string  `json:"checkout_request_id,omitempty"`
	ResponseCode      string  `json:"response_code"`
	ResponseDesc      string  `json:"response_description"`
	MerchantRequestID string  `json:"merchant_request_id,omitempty"`
	Amount            float64 `json:"amount"`
	PhoneNumber       string  `json:"phone_number"`
	Status            string  `json:"status"`
}

// CrossBorderQuote holds a quote for cross-border transfer
type CrossBorderQuote struct {
	FromCurrency  string  `json:"from_currency"`
	ToCurrency    string  `json:"to_currency"`
	SendAmount    float64 `json:"send_amount"`
	ReceiveAmount float64 `json:"receive_amount"`
	ExchangeRate  float64 `json:"exchange_rate"`
	Fee           float64 `json:"fee"`
	TotalInFiat   float64 `json:"total_in_fiat"`
	ValidUntil    string  `json:"valid_until"`
}