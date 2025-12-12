package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"paytabs/internal/model"
)

// TransferRepository defines transfer persistence operations.
type TransferRepository interface {
	Create(ctx context.Context, transfer *model.Transfer) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Transfer, error)
}

type transferRepository struct {
	db *gorm.DB
}

// NewTransferRepository creates a new transfer repository.
func NewTransferRepository(db *gorm.DB) TransferRepository {
	return &transferRepository{db: db}
}

// Create creates a new transfer record.
func (r *transferRepository) Create(ctx context.Context, transfer *model.Transfer) error {
	return r.db.WithContext(ctx).Create(transfer).Error
}

// FindByID finds a transfer by ID.
func (r *transferRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Transfer, error) {
	var transfer model.Transfer
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&transfer).Error; err != nil {
		return nil, err
	}
	return &transfer, nil
}

