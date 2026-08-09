package cache

import (
	"context"
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

// Set, Get and Delete used to live here and nothing ever called them: the
// server connects to Redis, logs whether it answered, and closes it again. They
// were removed rather than tested, because a test over a store no caller reads
// proves nothing except that the test runs (ticket 12 / ticket 13's rule about
// how the coverage number may be raised).
//
// Reinstate them together with the first read path that needs caching, so the
// shape they take is the one that call site actually wants.

// Close closes the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}
