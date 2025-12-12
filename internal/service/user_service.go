package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"paytabs/internal/cache"
	"paytabs/internal/model"
	"paytabs/internal/repository"
)

const userCacheTTL = 5 * time.Minute

// UserService exposes domain operations.
type UserService interface {
	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
	GetUser(ctx context.Context, id uint) (*model.User, error)
	ListUsers(ctx context.Context) ([]model.User, error)
}

type userService struct {
	repo  repository.UserRepository
	cache *cache.Client
}

// NewUserService builds a UserService with repository and cache.
func NewUserService(repo repository.UserRepository, cache *cache.Client) UserService {
	return &userService{repo: repo, cache: cache}
}

func (s *userService) cacheKey(id uint) string {
	return fmt.Sprintf("user:%d", id)
}

func (s *userService) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	_ = s.cache.Delete(ctx, s.cacheKey(user.ID))
	return user, nil
}

func (s *userService) GetUser(ctx context.Context, id uint) (*model.User, error) {
	if data, _ := s.cache.Get(ctx, s.cacheKey(id)); data != nil {
		var cached model.User
		if err := json.Unmarshal(data, &cached); err == nil {
			return &cached, nil
		}
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if payload, err := json.Marshal(user); err == nil {
		_ = s.cache.Set(ctx, s.cacheKey(id), payload, userCacheTTL)
	}
	return user, nil
}

func (s *userService) ListUsers(ctx context.Context) ([]model.User, error) {
	return s.repo.List(ctx)
}
