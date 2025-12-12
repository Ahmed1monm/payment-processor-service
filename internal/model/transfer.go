package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// TransferStatus represents the status of a transfer.
type TransferStatus string

const (
	TransferStatusPending   TransferStatus = "pending"
	TransferStatusCompleted TransferStatus = "completed"
	TransferStatusFailed    TransferStatus = "failed"
)

// Transfer represents an account-to-account money transfer.
type Transfer struct {
	ID                 uuid.UUID       `json:"id" gorm:"type:char(36);primaryKey"`
	SourceAccountID    uuid.UUID       `json:"source_account_id" gorm:"type:char(36);not null;index"`
	DestinationAccountID uuid.UUID     `json:"destination_account_id" gorm:"type:char(36);not null;index"`
	Amount             decimal.Decimal  `json:"amount" gorm:"type:decimal(20,2);not null"`
	Status             TransferStatus  `json:"status" gorm:"type:varchar(20);not null;default:'pending';index"`
	ErrorMessage       string          `json:"error_message,omitempty" gorm:"type:text"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	DeletedAt          gorm.DeletedAt  `json:"-" gorm:"index"`

	// Relations
	SourceAccount      Account `json:"-" gorm:"foreignKey:SourceAccountID"`
	DestinationAccount Account `json:"-" gorm:"foreignKey:DestinationAccountID"`
}

// BeforeCreate sets UUID before creating the record.
func (t *Transfer) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

