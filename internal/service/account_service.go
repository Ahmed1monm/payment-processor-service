package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"paytabs/internal/cache"
	"paytabs/internal/errors"
	"paytabs/internal/model"
	"paytabs/internal/repository"
)

const accountCacheTTL = 5 * time.Minute

// AccountService handles account operations.
type AccountService interface {
	GetAccount(ctx context.Context, id uuid.UUID) (*model.Account, error)
	GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error)
	SeedAccounts(ctx context.Context, accounts []model.Account) (int, error)
}

type accountService struct {
	repo  repository.AccountRepository
	cache *cache.Client
}

// NewAccountService creates a new account service.
func NewAccountService(repo repository.AccountRepository, cache *cache.Client) AccountService {
	return &accountService{
		repo:  repo,
		cache: cache,
	}
}

func (s *accountService) cacheKey(id uuid.UUID) string {
	return fmt.Sprintf("account:%s", id.String())
}

// GetAccount retrieves an account by ID with caching.
func (s *accountService) GetAccount(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	// Try cache first
	if data, _ := s.cache.Get(ctx, s.cacheKey(id)); data != nil {
		var cached model.Account
		if err := json.Unmarshal(data, &cached); err == nil {
			return &cached, nil
		}
	}

	// Fetch from database
	account, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrAccountNotFound
		}
		return nil, err
	}

	// Cache the result
	if payload, err := json.Marshal(account); err == nil {
		_ = s.cache.Set(ctx, s.cacheKey(id), payload, accountCacheTTL)
	}

	return account, nil
}

// GetBalance retrieves the current balance of an account.
func (s *accountService) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	account, err := s.GetAccount(ctx, id)
	if err != nil {
		if err == errors.ErrAccountNotFound {
			return decimal.Zero, err
		}
		return decimal.Zero, fmt.Errorf("get account: %w", err)
	}
	return account.Balance, nil
}

// SeedAccounts creates or updates accounts from external data.
func (s *accountService) SeedAccounts(ctx context.Context, accounts []model.Account) (int, error) {
	count := 0
	for _, account := range accounts {
		// Check if account exists
		existing, err := s.repo.FindByID(ctx, account.ID)
		if err != nil && err != gorm.ErrRecordNotFound {
			return count, fmt.Errorf("seed account %s: %w", account.ID, err)
		}

		if existing != nil {
			// Update existing account with new data
			existing.Name = account.Name
			existing.Balance = account.Balance
			existing.Active = account.Active
			if err := s.repo.Update(ctx, existing); err != nil {
				return count, fmt.Errorf("update account %s: %w", account.ID, err)
			}
		} else {
			// Create new account
			if err := s.repo.Create(ctx, &account); err != nil {
				return count, fmt.Errorf("create account %s: %w", account.ID, err)
			}
		}

		// Invalidate cache
		_ = s.cache.Delete(ctx, s.cacheKey(account.ID))
		count++
	}
	return count, nil
}
