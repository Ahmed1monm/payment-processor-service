package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"paytabs/internal/model"
)

// CardRepository defines card persistence operations.
type CardRepository interface {
	Create(ctx context.Context, card *model.Card) error
	Update(ctx context.Context, card *model.Card) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Card, error)
	FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*model.Card, error)
	FindByAccountID(ctx context.Context, accountID uuid.UUID) ([]model.Card, error)
	UpdateBalance(ctx context.Context, id uuid.UUID, newBalance interface{}) error
	FindByCardNumber(ctx context.Context, cardNumber string) (*model.Card, error)
	// Transaction methods
	WithTransaction(ctx context.Context, fn func(ctx context.Context, repo CardRepository) error) error
	FindByIDForUpdateTx(ctx context.Context, tx interface{}, id uuid.UUID) (*model.Card, error)
	UpdateBalanceTx(ctx context.Context, tx interface{}, id uuid.UUID, newBalance interface{}) error
}

type cardRepository struct {
	db *gorm.DB
}

// NewCardRepository creates a new card repository.
func NewCardRepository(db *gorm.DB) CardRepository {
	return &cardRepository{db: db}
}

// Create creates a new card.
func (r *cardRepository) Create(ctx context.Context, card *model.Card) error {
	return r.db.WithContext(ctx).Create(card).Error
}

// Update updates an existing card.
func (r *cardRepository) Update(ctx context.Context, card *model.Card) error {
	return r.db.WithContext(ctx).Save(card).Error
}

// FindByID finds a card by ID.
func (r *cardRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Card, error) {
	var card model.Card
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&card).Error; err != nil {
		return nil, err
	}
	return &card, nil
}

// FindByIDForUpdate finds a card by ID with row-level lock for update.
func (r *cardRepository) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*model.Card, error) {
	var card model.Card
	if err := r.db.WithContext(ctx).Set("gorm:query_option", "FOR UPDATE").
		Where("id = ?", id).First(&card).Error; err != nil {
		return nil, err
	}
	return &card, nil
}

// FindByAccountID finds all cards for an account.
func (r *cardRepository) FindByAccountID(ctx context.Context, accountID uuid.UUID) ([]model.Card, error) {
	var cards []model.Card
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Find(&cards).Error; err != nil {
		return nil, err
	}
	return cards, nil
}

// UpdateBalance updates the balance of a card.
func (r *cardRepository) UpdateBalance(ctx context.Context, id uuid.UUID, newBalance interface{}) error {
	return r.db.WithContext(ctx).Model(&model.Card{}).
		Where("id = ?", id).
		Update("balance", newBalance).Error
}

// FindByCardNumber finds a card by card number (for payment processing).
func (r *cardRepository) FindByCardNumber(ctx context.Context, cardNumber string) (*model.Card, error) {
	var card model.Card
	if err := r.db.WithContext(ctx).Where("card_number = ?", cardNumber).First(&card).Error; err != nil {
		return nil, err
	}
	return &card, nil
}

// FindByIDForUpdateTx finds a card by ID with row-level lock within a transaction.
func (r *cardRepository) FindByIDForUpdateTx(ctx context.Context, tx interface{}, id uuid.UUID) (*model.Card, error) {
	txDB := tx.(*gorm.DB)
	var card model.Card
	if err := txDB.WithContext(ctx).Set("gorm:query_option", "FOR UPDATE").
		Where("id = ?", id).First(&card).Error; err != nil {
		return nil, err
	}
	return &card, nil
}

// UpdateBalanceTx updates the balance within a transaction.
func (r *cardRepository) UpdateBalanceTx(ctx context.Context, tx interface{}, id uuid.UUID, newBalance interface{}) error {
	txDB := tx.(*gorm.DB)
	return txDB.WithContext(ctx).Model(&model.Card{}).
		Where("id = ?", id).
		Update("balance", newBalance).Error
}

// WithTransaction executes a function within a database transaction.
func (r *cardRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context, repo CardRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &cardRepository{db: tx}
		return fn(ctx, txRepo)
	})
}

