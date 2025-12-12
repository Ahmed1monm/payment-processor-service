package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"paytabs/internal/cache"
	"paytabs/internal/errors"
	"paytabs/internal/model"
	"paytabs/internal/repository"
)

// TransferService handles account-to-account transfer operations.
type TransferService interface {
	ProcessTransfer(ctx context.Context, sourceAccountID, destinationAccountID uuid.UUID, amount decimal.Decimal) (*model.Transfer, error)
}

type transferService struct {
	accountRepo  repository.AccountRepository
	transferRepo repository.TransferRepository
	cache        *cache.Client
}

// NewTransferService creates a new transfer service.
func NewTransferService(
	accountRepo repository.AccountRepository,
	transferRepo repository.TransferRepository,
	cache *cache.Client,
) TransferService {
	return &transferService{
		accountRepo:  accountRepo,
		transferRepo: transferRepo,
		cache:        cache,
	}
}

// ProcessTransfer processes an account-to-account transfer with atomic balance updates.
func (s *transferService) ProcessTransfer(ctx context.Context, sourceAccountID, destinationAccountID uuid.UUID, amount decimal.Decimal) (*model.Transfer, error) {
	// Validate amount
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.ErrInvalidAmount
	}

	// Prevent self-transfer
	if sourceAccountID == destinationAccountID {
		return nil, fmt.Errorf("cannot transfer to the same account")
	}

	// Create transfer record
	transfer := &model.Transfer{
		SourceAccountID:      sourceAccountID,
		DestinationAccountID: destinationAccountID,
		Amount:               amount,
		Status:               model.TransferStatusPending,
	}

	// Use transaction for atomic balance updates
	err := s.accountRepo.WithTransaction(ctx, func(ctx context.Context, txRepo repository.AccountRepository) error {
		// Lock and fetch source account
		sourceAccount, err := txRepo.FindByIDForUpdate(ctx, sourceAccountID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				transfer.Status = model.TransferStatusFailed
				transfer.ErrorMessage = errors.ErrAccountNotFound.Error()
				return errors.ErrAccountNotFound
			}
			transfer.Status = model.TransferStatusFailed
			transfer.ErrorMessage = err.Error()
			return err
		}

		// Validate source account is active
		if !sourceAccount.Active {
			transfer.Status = model.TransferStatusFailed
			transfer.ErrorMessage = errors.ErrAccountInactive.Error()
			return errors.ErrAccountInactive
		}

		// Check sufficient balance
		if sourceAccount.Balance.LessThan(amount) {
			transfer.Status = model.TransferStatusFailed
			transfer.ErrorMessage = errors.ErrInsufficientBalance.Error()
			return errors.ErrInsufficientBalance
		}

		// Lock and fetch destination account
		destAccount, err := txRepo.FindByIDForUpdate(ctx, destinationAccountID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				transfer.Status = model.TransferStatusFailed
				transfer.ErrorMessage = errors.ErrAccountNotFound.Error()
				return errors.ErrAccountNotFound
			}
			transfer.Status = model.TransferStatusFailed
			transfer.ErrorMessage = err.Error()
			return err
		}

		// Validate destination account is active
		if !destAccount.Active {
			transfer.Status = model.TransferStatusFailed
			transfer.ErrorMessage = errors.ErrAccountInactive.Error()
			return errors.ErrAccountInactive
		}

		// Update balances atomically
		newSourceBalance := sourceAccount.Balance.Sub(amount)
		newDestBalance := destAccount.Balance.Add(amount)

		if err := txRepo.UpdateBalance(ctx, sourceAccountID, newSourceBalance); err != nil {
			transfer.Status = model.TransferStatusFailed
			transfer.ErrorMessage = fmt.Sprintf("failed to update source balance: %v", err)
			return err
		}

		if err := txRepo.UpdateBalance(ctx, destinationAccountID, newDestBalance); err != nil {
			transfer.Status = model.TransferStatusFailed
			transfer.ErrorMessage = fmt.Sprintf("failed to update destination balance: %v", err)
			return err
		}

		// Mark transfer as completed
		transfer.Status = model.TransferStatusCompleted
		return nil
	})

	// Create transfer record (regardless of success/failure)
	if err := s.transferRepo.Create(ctx, transfer); err != nil {
		return transfer, fmt.Errorf("create transfer record: %w", err)
	}

	// If transaction failed, return error
	if err != nil {
		return transfer, err
	}

	// Invalidate cache for both accounts
	_ = s.cache.Delete(ctx, fmt.Sprintf("account:%s", sourceAccountID.String()))
	_ = s.cache.Delete(ctx, fmt.Sprintf("account:%s", destinationAccountID.String()))

	return transfer, nil
}

