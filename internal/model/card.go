package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Card represents a payment card linked to an account.
type Card struct {
	ID          uuid.UUID       `json:"id" gorm:"type:char(36);primaryKey"`
	AccountID   uuid.UUID       `json:"account_id" gorm:"type:char(36);not null;index"`
	CardNumber  string          `json:"card_number" gorm:"size:19;not null"` // Masked card number
	CardExpiry  string          `json:"card_expiry" gorm:"size:5;not null"`  // MM/YY format
	Balance     decimal.Decimal `json:"balance" gorm:"type:decimal(20,2);not null;default:0"`
	Active      bool            `json:"active" gorm:"default:true;index"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   gorm.DeletedAt  `json:"-" gorm:"index"`

	// Relations
	Account Account `json:"-" gorm:"foreignKey:AccountID"`
}

// BeforeCreate sets UUID before creating the record.
func (c *Card) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

