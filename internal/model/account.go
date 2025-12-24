package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Account represents a merchant or user account in the payment system.
type Account struct {
	ID           uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey"`
	Name         string          `json:"name" gorm:"size:255;not null;index"`
	Email        string          `json:"email" gorm:"uniqueIndex;size:255;not null"`
	PasswordHash string          `json:"-" gorm:"size:255;not null"` // Never expose in JSON
	IsMerchant   bool            `json:"is_merchant" gorm:"default:false;index"`
	Active       bool            `json:"active" gorm:"default:true;index"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	DeletedAt    gorm.DeletedAt  `json:"-" gorm:"index"`

	// Relations
	Cards []Card `json:"cards,omitempty" gorm:"foreignKey:AccountID"`
}

// BeforeCreate sets UUID before creating the record.
func (a *Account) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
