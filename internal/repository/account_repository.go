package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"paytabs/internal/model"
)

// AccountRepository defines account persistence operations.
type AccountRepository interface {
	Create(ctx context.Context, account *model.Account) error
	Update(ctx context.Context, account *model.Account) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Account, error)
	FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*model.Account, error)
	UpdateBalance(ctx context.Context, id uuid.UUID, newBalance interface{}) error
	ListActive(ctx context.Context) ([]model.Account, error)
	FindByIDOrCreate(ctx context.Context, account *model.Account) (*model.Account, error)
	// Transaction methods
	WithTransaction(ctx context.Context, fn func(ctx context.Context, repo AccountRepository) error) error
	FindByIDForUpdateTx(ctx context.Context, tx interface{}, id uuid.UUID) (*model.Account, error)
	UpdateBalanceTx(ctx context.Context, tx interface{}, id uuid.UUID, newBalance interface{}) error
}

type accountRepository struct {
	db *gorm.DB
}

// NewAccountRepository creates a new account repository.
func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &accountRepository{db: db}
}

// Create creates a new account.
func (r *accountRepository) Create(ctx context.Context, account *model.Account) error {
	return r.db.WithContext(ctx).Create(account).Error
}

// Update updates an existing account.
func (r *accountRepository) Update(ctx context.Context, account *model.Account) error {
	return r.db.WithContext(ctx).Save(account).Error
}

// FindByID finds an account by ID.
func (r *accountRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	var account model.Account
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

// FindByIDForUpdate finds an account by ID with row-level lock for update.
func (r *accountRepository) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	var account model.Account
	if err := r.db.WithContext(ctx).Set("gorm:query_option", "FOR UPDATE").
		Where("id = ?", id).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

// UpdateBalance updates the balance of an account.
func (r *accountRepository) UpdateBalance(ctx context.Context, id uuid.UUID, newBalance interface{}) error {
	return r.db.WithContext(ctx).Model(&model.Account{}).
		Where("id = ?", id).
		Update("balance", newBalance).Error
}

// ListActive lists all active accounts.
func (r *accountRepository) ListActive(ctx context.Context) ([]model.Account, error) {
	var accounts []model.Account
	if err := r.db.WithContext(ctx).Where("active = ?", true).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

// FindByIDOrCreate finds an account by ID or creates it if it doesn't exist.
func (r *accountRepository) FindByIDOrCreate(ctx context.Context, account *model.Account) (*model.Account, error) {
	var existing model.Account
	err := r.db.WithContext(ctx).Where("id = ?", account.ID).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Account doesn't exist, create it
	if err := r.db.WithContext(ctx).Create(account).Error; err != nil {
		return nil, err
	}
	return account, nil
}

// WithTransaction executes a function within a database transaction.
func (r *accountRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context, repo AccountRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &accountRepository{db: tx}
		return fn(ctx, txRepo)
	})
}

// FindByIDForUpdateTx finds an account by ID with row-level lock within a transaction.
func (r *accountRepository) FindByIDForUpdateTx(ctx context.Context, tx interface{}, id uuid.UUID) (*model.Account, error) {
	txDB := tx.(*gorm.DB)
	var account model.Account
	if err := txDB.WithContext(ctx).Set("gorm:query_option", "FOR UPDATE").
		Where("id = ?", id).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

// UpdateBalanceTx updates the balance within a transaction.
func (r *accountRepository) UpdateBalanceTx(ctx context.Context, tx interface{}, id uuid.UUID, newBalance interface{}) error {
	txDB := tx.(*gorm.DB)
	return txDB.WithContext(ctx).Model(&model.Account{}).
		Where("id = ?", id).
		Update("balance", newBalance).Error
}
