package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentLog represents a log entry for a payment attempt.
// All payment attempts are logged regardless of success or failure.
type PaymentLog struct {
	ID          uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey"`
	PaymentID   uuid.UUID      `json:"payment_id" gorm:"type:char(36);not null;index"`
	Status      PaymentStatus  `json:"status" gorm:"type:varchar(20);not null;index"`
	ErrorMessage string        `json:"error_message,omitempty" gorm:"type:text"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Payment Payment `json:"-" gorm:"foreignKey:PaymentID"`
}

// BeforeCreate sets UUID before creating the record.
func (pl *PaymentLog) BeforeCreate(tx *gorm.DB) error {
	if pl.ID == uuid.Nil {
		pl.ID = uuid.New()
	}
	return nil
}

