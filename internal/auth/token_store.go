package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"paytabs/internal/cache"
)

const (
	refreshTokenKeyPrefix = "refresh_token:"
	accessTokenKeyPrefix  = "blacklist:access_token:"
)

// TokenStoreInterface defines the interface for token storage operations.
type TokenStoreInterface interface {
	StoreRefreshToken(ctx context.Context, tokenID string, userID uint, email string, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, tokenID string) (userID uint, email string, err error)
	DeleteRefreshToken(ctx context.Context, tokenID string) error
	BlacklistAccessToken(ctx context.Context, tokenID string, ttl time.Duration) error
	IsAccessTokenBlacklisted(ctx context.Context, tokenID string) (bool, error)
}

// TokenStore handles storage and retrieval of tokens in Redis.
type TokenStore struct {
	cache *cache.Client
}

// Ensure TokenStore implements TokenStoreInterface
var _ TokenStoreInterface = (*TokenStore)(nil)

// NewTokenStore creates a new token store.
func NewTokenStore(cache *cache.Client) *TokenStore {
	return &TokenStore{cache: cache}
}

// StoreRefreshToken stores a refresh token in Redis with TTL.
func (s *TokenStore) StoreRefreshToken(ctx context.Context, tokenID string, userID uint, email string, ttl time.Duration) error {
	data := map[string]interface{}{
		"user_id": userID,
		"email":   email,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal token data: %w", err)
	}

	key := refreshTokenKeyPrefix + tokenID
	return s.cache.Set(ctx, key, payload, ttl)
}

// GetRefreshToken retrieves refresh token data from Redis.
func (s *TokenStore) GetRefreshToken(ctx context.Context, tokenID string) (userID uint, email string, err error) {
	key := refreshTokenKeyPrefix + tokenID
	data, err := s.cache.Get(ctx, key)
	if err != nil || data == nil {
		return 0, "", fmt.Errorf("refresh token not found")
	}

	var tokenData map[string]interface{}
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return 0, "", fmt.Errorf("unmarshal token data: %w", err)
	}

	// Extract user_id and email
	uid, ok := tokenData["user_id"].(float64)
	if !ok {
		return 0, "", fmt.Errorf("invalid user_id in token data")
	}
	userID = uint(uid)

	email, ok = tokenData["email"].(string)
	if !ok {
		return 0, "", fmt.Errorf("invalid email in token data")
	}

	return userID, email, nil
}

// DeleteRefreshToken removes a refresh token from Redis.
func (s *TokenStore) DeleteRefreshToken(ctx context.Context, tokenID string) error {
	key := refreshTokenKeyPrefix + tokenID
	return s.cache.Delete(ctx, key)
}

// BlacklistAccessToken adds an access token to the blacklist until it expires.
func (s *TokenStore) BlacklistAccessToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	key := accessTokenKeyPrefix + tokenID
	// Store a simple marker
	return s.cache.Set(ctx, key, []byte("1"), ttl)
}

// IsAccessTokenBlacklisted checks if an access token is blacklisted.
func (s *TokenStore) IsAccessTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	key := accessTokenKeyPrefix + tokenID
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return false, nil // Not blacklisted if error (fail safe)
	}
	return data != nil, nil
}

