package models

import (
	"time"

	"instadoh-backend/types"

	"gorm.io/gorm"
)

// User represents a user in the system (individual or merchant)
type User struct {
	ID          uint           `gorm:"primaryKey"`
	Email       string         `gorm:"uniqueIndex;size:255;not null"`
	Phone       string         `gorm:"uniqueIndex;size:50;not null"`
	Password    string         `gorm:"size:255;not null"`
	FullName    string         `gorm:"size:255;not null"`
	CountryCode string         `gorm:"size:2;not null"`
	Currency    string         `gorm:"size:3;not null;default:'USD'"`
	Balance     float64        `gorm:"type:decimal(20,8);default:0"`
	Role        types.UserRole `gorm:"size:20;default:'user'"`
	LNAddress   string         `gorm:"size:255"`
	IsActive    bool           `gorm:"default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	Transactions []Transaction `gorm:"foreignKey:UserID"`
}

// Country defines supported countries with their currencies
type Country struct {
	Code         string `gorm:"primaryKey;size:2"`
	Name         string `gorm:"size:255;not null"`
	Currency     string `gorm:"size:3;not null"`
	CurrencyName string `gorm:"size:100;not null"`
	Flag         string `gorm:"size:10"`
	IsActive     bool   `gorm:"default:true"`
}

// Transaction records every payment made/received
type Transaction struct {
	ID             uint                    `gorm:"primaryKey"`
	UserID         uint                    `gorm:"index;not null"`
	Amount         float64                 `gorm:"type:decimal(20,2);not null"`
	Currency       string                  `gorm:"size:3;not null"`
	AmountBTC      int64                   `gorm:"not null"` // amount in millisatoshis
	Direction      types.PaymentDirection  `gorm:"size:20;not null"`
	Status         types.TransactionStatus `gorm:"size:20;default:'pending'"`
	PaymentHash    string                  `gorm:"uniqueIndex;size:255"`
	PaymentRequest string                  `gorm:"size:1024"`
	Preimage       string                  `gorm:"size:255"`
	CounterpartyID *uint                   `gorm:"index"`
	Counterparty   string                  `gorm:"size:255"`
	Description    string                  `gorm:"size:500"`
	FeeMsat        int64                   `gorm:"default:0"`
	ExchangeRate   float64                 `gorm:"type:decimal(20,8)"`
	SettledAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// APIKey stores API keys for programmatic access
type APIKey struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"index;not null"`
	Key       string `gorm:"uniqueIndex;size:255;not null"`
	Name      string `gorm:"size:100;not null"`
	IsActive  bool   `gorm:"default:true"`
	LastUsed  *time.Time
	ExpiresAt *time.Time
	CreatedAt time.Time
}

// --- Table name customizations ---

func (User) TableName() string {
	return "users"
}

func (Country) TableName() string {
	return "countries"
}

func (Transaction) TableName() string {
	return "transactions"
}

func (APIKey) TableName() string {
	return "api_keys"
}

// --- Model to Response converters ---

func (u *User) ToResponse() types.UserResponse {
	return types.UserResponse{
		ID:          u.ID,
		Email:       u.Email,
		Phone:       u.Phone,
		FullName:    u.FullName,
		CountryCode: u.CountryCode,
		Currency:    u.Currency,
		Balance:     u.Balance,
		Role:        u.Role,
		CreatedAt:   u.CreatedAt,
	}
}

func (t *Transaction) ToResponse() types.TransactionResponse {
	resp := types.TransactionResponse{
		ID:             t.ID,
		UserID:         t.UserID,
		Amount:         t.Amount,
		Currency:       t.Currency,
		AmountBTC:      t.AmountBTC,
		Direction:      t.Direction,
		Status:         t.Status,
		PaymentHash:    t.PaymentHash,
		PaymentRequest: t.PaymentRequest,
		Counterparty:   t.Counterparty,
		Description:    t.Description,
		CreatedAt:      t.CreatedAt,
		CompletedAt:    t.SettledAt,
	}
	return resp
}
