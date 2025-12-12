package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Account represents a merchant or user account in the payment system.
type Account struct {
	ID        uuid.UUID       `json:"id" gorm:"type:char(36);primaryKey"`
	Name      string          `json:"name" gorm:"size:255;not null;index"`
	Balance   decimal.Decimal `json:"balance" gorm:"type:decimal(20,2);not null;default:0"`
	Active    bool            `json:"active" gorm:"default:true;index"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt gorm.DeletedAt  `json:"-" gorm:"index"`
}

// BeforeCreate sets UUID before creating the record.
func (a *Account) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
