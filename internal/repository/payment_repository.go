package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"paytabs/internal/model"
)

// PaymentRepository defines payment persistence operations.
type PaymentRepository interface {
	Create(ctx context.Context, payment *model.Payment) error
	Update(ctx context.Context, payment *model.Payment) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Payment, error)
}

type paymentRepository struct {
	db *gorm.DB
}

// NewPaymentRepository creates a new payment repository.
func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

// Create creates a new payment record.
func (r *paymentRepository) Create(ctx context.Context, payment *model.Payment) error {
	return r.db.WithContext(ctx).Create(payment).Error
}

// Update updates an existing payment record.
func (r *paymentRepository) Update(ctx context.Context, payment *model.Payment) error {
	return r.db.WithContext(ctx).Save(payment).Error
}

// FindByID finds a payment by ID.
func (r *paymentRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	var payment model.Payment
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&payment).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

// PaymentLogRepository defines payment log persistence operations.
type PaymentLogRepository interface {
	Create(ctx context.Context, log *model.PaymentLog) error
	CreateBatch(ctx context.Context, logs []model.PaymentLog) error
}

type paymentLogRepository struct {
	db *gorm.DB
}

// NewPaymentLogRepository creates a new payment log repository.
func NewPaymentLogRepository(db *gorm.DB) PaymentLogRepository {
	return &paymentLogRepository{db: db}
}

// Create creates a new payment log entry.
func (r *paymentLogRepository) Create(ctx context.Context, log *model.PaymentLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// CreateBatch creates multiple payment log entries in a single transaction.
func (r *paymentLogRepository) CreateBatch(ctx context.Context, logs []model.PaymentLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(logs, 100).Error
}

