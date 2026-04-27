package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps a Redis client with helper methods.
type Client struct {
	rdb *redis.Client
}

// NewClient creates a new Redis cache client.
func NewClient(addr, password string, db int) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: failed to ping: %w", err)
	}

	return &Client{rdb: rdb}, nil
}

// Set stores a JSON-encoded value with a TTL.
func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: marshal error: %w", err)
	}
	return c.rdb.Set(ctx, key, b, ttl).Err()
}

// Get retrieves and JSON-decodes a cached value. Returns (false, nil) on a cache miss.
func (c *Client) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	b, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache: get error: %w", err)
	}
	if err := json.Unmarshal(b, dest); err != nil {
		return false, fmt.Errorf("cache: unmarshal error: %w", err)
	}
	return true, nil
}

// Delete removes a key from the cache.
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}
