package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// PaymentStatus represents the status of a payment.
type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusAccepted PaymentStatus = "accepted"
	PaymentStatusFailed   PaymentStatus = "failed"
)

// Payment represents a card-based payment transaction.
type Payment struct {
	ID                uuid.UUID       `json:"id" gorm:"type:char(36);primaryKey"`
	MerchantAccountID uuid.UUID       `json:"merchant_account_id" gorm:"type:char(36);not null;index"`
	Amount            decimal.Decimal `json:"amount" gorm:"type:decimal(20,2);not null"`
	CardNumber        string          `json:"card_number" gorm:"size:19;not null"` // Masked card number
	CardExpiry        string          `json:"card_expiry" gorm:"size:5;not null"`  // MM/YY format
	CardCVV           string          `json:"-" gorm:"size:4"`                     // Not exposed in JSON
	Status            PaymentStatus   `json:"status" gorm:"type:varchar(20);not null;default:'pending';index"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	DeletedAt         gorm.DeletedAt  `json:"-" gorm:"index"`

	// Relations
	MerchantAccount Account `json:"-" gorm:"foreignKey:MerchantAccountID"`
}

// BeforeCreate sets UUID before creating the record.
func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
