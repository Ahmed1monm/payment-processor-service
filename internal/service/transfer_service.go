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

// TransferService handles card-to-card transfer operations.
type TransferService interface {
	ProcessTransfer(ctx context.Context, sourceCardID, destinationCardID uuid.UUID, amount decimal.Decimal) (*model.Transfer, error)
}

type transferService struct {
	cardRepo     repository.CardRepository
	transferRepo repository.TransferRepository
	cache        *cache.Client
}

// NewTransferService creates a new transfer service.
func NewTransferService(
	cardRepo repository.CardRepository,
	transferRepo repository.TransferRepository,
	cache *cache.Client,
) TransferService {
	return &transferService{
		cardRepo:     cardRepo,
		transferRepo: transferRepo,
		cache:        cache,
	}
}

// ProcessTransfer processes a card-to-card transfer with atomic balance updates.
func (s *transferService) ProcessTransfer(ctx context.Context, sourceCardID, destinationCardID uuid.UUID, amount decimal.Decimal) (*model.Transfer, error) {
	// Validate amount
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.ErrInvalidAmount
	}

	// Prevent self-transfer
	if sourceCardID == destinationCardID {
		return nil, fmt.Errorf("cannot transfer to the same card")
	}

	// Create transfer record
	transfer := &model.Transfer{
		SourceCardID:      sourceCardID,
		DestinationCardID: destinationCardID,
		Amount:            amount,
		Status:            model.TransferStatusPending,
	}

	// Use transaction for atomic balance updates
	err := s.cardRepo.WithTransaction(ctx, func(ctx context.Context, txRepo repository.CardRepository) error {
		// Lock and fetch source card
		sourceCard, err := txRepo.FindByIDForUpdate(ctx, sourceCardID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				transfer.Status = model.TransferStatusFailed
				transfer.ErrorMessage = "source card not found"
				return fmt.Errorf("source card not found")
			}
			transfer.Status = model.TransferStatusFailed
			transfer.ErrorMessage = err.Error()
			return err
		}

		// Validate source card is active
		if !sourceCard.Active {
			transfer.Status = model.TransferStatusFailed
			transfer.ErrorMessage = "source card is not active"
			return fmt.Errorf("source card is not active")
		}

		// Check sufficient balance
		if sourceCard.Balance.LessThan(amount) {
			transfer.Status = model.TransferStatusFailed
			transfer.ErrorMessage = errors.ErrInsufficientBalance.Error()
			return errors.ErrInsufficientBalance
		}

		// Lock and fetch destination card
		destCard, err := txRepo.FindByIDForUpdate(ctx, destinationCardID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				transfer.Status = model.TransferStatusFailed
				transfer.ErrorMessage = "destination card not found"
				return fmt.Errorf("destination card not found")
			}
			transfer.Status = model.TransferStatusFailed
			transfer.ErrorMessage = err.Error()
			return err
		}

		// Validate destination card is active
		if !destCard.Active {
			transfer.Status = model.TransferStatusFailed
			transfer.ErrorMessage = "destination card is not active"
			return fmt.Errorf("destination card is not active")
		}

		// Update balances atomically
		newSourceBalance := sourceCard.Balance.Sub(amount)
		newDestBalance := destCard.Balance.Add(amount)

		if err := txRepo.UpdateBalance(ctx, sourceCardID, newSourceBalance); err != nil {
			transfer.Status = model.TransferStatusFailed
			transfer.ErrorMessage = fmt.Sprintf("failed to update source balance: %v", err)
			return err
		}

		if err := txRepo.UpdateBalance(ctx, destinationCardID, newDestBalance); err != nil {
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

	// Invalidate cache for both cards
	_ = s.cache.Delete(ctx, fmt.Sprintf("card:%s", sourceCardID.String()))
	_ = s.cache.Delete(ctx, fmt.Sprintf("card:%s", destinationCardID.String()))

	return transfer, nil
}

