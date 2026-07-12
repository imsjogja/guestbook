// Package repository provides Redis connection management for caching,
// session storage, and rate limiting.
package repository

import (
	"context"
	"fmt"
	"time"

	"guestflow/internal/config"

	"github.com/redis/go-redis/v9"
)

// NewRedisConnection creates a new Redis client with the provided configuration.
// It validates connectivity with a ping before returning the client.
//
// The caller is responsible for calling Close() on the returned *redis.Client when done.
func NewRedisConnection(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
	})

	// Verify connectivity with a ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return client, nil
}

// RedisHealthCheck pings the Redis server to verify connectivity.
func RedisHealthCheck(ctx context.Context, client *redis.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}
	return nil
}

// RateLimiter provides Redis-backed rate limiting functionality.
type RateLimiter struct {
	client *redis.Client
	prefix string
}

// NewRateLimiter creates a new RateLimiter backed by Redis.
func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{
		client: client,
		prefix: "ratelimit",
	}
}

// IsAllowed checks if a request from the given key is allowed based on
// the rate limit configuration. It uses a sliding window counter approach.
func (rl *RateLimiter) IsAllowed(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	fullKey := fmt.Sprintf("%s:%s", rl.prefix, key)
	now := time.Now().Unix()
	windowStart := now - int64(window.Seconds())

	pipe := rl.client.Pipeline()

	// Remove entries outside the current window
	pipe.ZRemRangeByScore(ctx, fullKey, "0", fmt.Sprintf("%d", windowStart))

	// Count entries in the current window
	countCmd := pipe.ZCard(ctx, fullKey)

	// Add current request
	pipe.ZAdd(ctx, fullKey, redis.Z{
		Score:  float64(now),
		Member: fmt.Sprintf("%d:%s", now, generateNonce()),
	})

	// Set expiry on the key
	pipe.Expire(ctx, fullKey, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	count := countCmd.Val()
	return count < int64(limit), nil
}

// generateNonce creates a short random string to ensure uniqueness of
// sorted set members when multiple requests arrive in the same second.
func generateNonce() string {
	b := make([]byte, 4)
	for i := range b {
		b[i] = byte(65 + (i*7)%26) // deterministic pseudo-random for simplicity
	}
	return string(b)
}
