package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"paytabs/internal/errors"
	"paytabs/internal/repository"
)

// CardService handles card operations.
type CardService interface {
	GetBalance(ctx context.Context, cardID uuid.UUID) (decimal.Decimal, error)
	GetAccountTotalBalance(ctx context.Context, accountID uuid.UUID) (decimal.Decimal, error)
}

type cardService struct {
	cardRepo repository.CardRepository
}

// NewCardService creates a new card service.
func NewCardService(cardRepo repository.CardRepository) CardService {
	return &cardService{
		cardRepo: cardRepo,
	}
}

// GetBalance retrieves the current balance of a card.
func (s *cardService) GetBalance(ctx context.Context, cardID uuid.UUID) (decimal.Decimal, error) {
	card, err := s.cardRepo.FindByID(ctx, cardID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return decimal.Zero, errors.ErrCardNotFound
		}
		return decimal.Zero, fmt.Errorf("get card: %w", err)
	}
	return card.Balance, nil
}

// GetAccountTotalBalance calculates the total balance across all cards for an account.
func (s *cardService) GetAccountTotalBalance(ctx context.Context, accountID uuid.UUID) (decimal.Decimal, error) {
	cards, err := s.cardRepo.FindByAccountID(ctx, accountID)
	if err != nil {
		return decimal.Zero, fmt.Errorf("get cards: %w", err)
	}

	total := decimal.Zero
	for _, card := range cards {
		if card.Active {
			total = total.Add(card.Balance)
		}
	}

	return total, nil
}
