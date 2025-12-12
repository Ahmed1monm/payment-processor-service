package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps redis.Client but fails safe by swallowing connectivity errors.
type Client struct {
	client *redis.Client
}

// New creates a new Redis client.
func New(addr, password string, db int) *Client {
	opts := &redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	}
	return &Client{client: redis.NewClient(opts)}
}

// Get returns value or nil if missing or redis unavailable.
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	if c == nil || c.client == nil {
		return nil, nil
	}
	res, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		// fail safe: behave like cache miss
		return nil, nil
	}
	return res, nil
}

// Set stores value with TTL, ignoring redis errors.
func (c *Client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		// fail safe: ignore redis errors
		return nil
	}
	return nil
}

// Delete removes a key, ignoring redis errors.
func (c *Client) Delete(ctx context.Context, key string) error {
	if c == nil || c.client == nil {
		return nil
	}
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return nil
	}
	return nil
}
