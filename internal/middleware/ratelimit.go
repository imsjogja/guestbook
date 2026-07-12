// Package middleware provides rate limiting functionality for GuestFlow.
// Rate limiting is backed by Redis and supports per-IP and per-user rate limiting.
package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	apperrors "guestflow/pkg/errors"
	appresponse "guestflow/pkg/response"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

// RateLimiter defines the interface for rate limiting operations.
// This interface allows swapping Redis-based limiting with in-memory
// implementations for testing or local development.
type RateLimiter interface {
	// IsAllowed checks if a request with the given key is within the rate limit.
	// Returns true if the request is allowed, false if rate limited.
	IsAllowed(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// RateLimitConfig holds configuration for the rate limiting middleware.
type RateLimitConfig struct {
	// Redis client for rate limit storage (required)
	RedisClient *redis.Client

	// RequestsPerSecond is the maximum number of requests per time window (default: 10)
	RequestsPerSecond float64

	// Burst allows a short burst of requests above the rate (default: 20)
	Burst int

	// Window is the time window for rate limiting (default: 1 minute)
	Window time.Duration

	// KeyPrefix is prepended to all Redis keys (default: "ratelimit")
	KeyPrefix string

	// KeyExtractor returns the rate limit key for a request.
	// Defaults to client IP + path combination.
	KeyExtractor func(c echo.Context) string

	// Skipper defines paths that should skip rate limiting.
	// Health check and metrics endpoints are skipped by default.
	Skipper func(c echo.Context) bool
}

// DefaultRateLimitConfig returns a sensible default configuration.
func DefaultRateLimitConfig(redisClient *redis.Client) RateLimitConfig {
	return RateLimitConfig{
		RedisClient:       redisClient,
		RequestsPerSecond: 10,
		Burst:             20,
		Window:            time.Minute,
		KeyPrefix:         "ratelimit",
		KeyExtractor:      defaultKeyExtractor,
		Skipper:           defaultSkipper,
	}
}

// RateLimit returns rate limiting middleware that uses Redis for distributed
// rate limiting. Requests that exceed the limit receive a 429 Too Many Requests
// response with a Retry-After header.
func RateLimit(config RateLimitConfig) echo.MiddlewareFunc {
	if config.RedisClient == nil {
		panic("ratelimit: RedisClient is required")
	}
	if config.RequestsPerSecond <= 0 {
		config.RequestsPerSecond = 10
	}
	if config.Burst <= 0 {
		config.Burst = 20
	}
	if config.Window <= 0 {
		config.Window = time.Minute
	}
	if config.KeyPrefix == "" {
		config.KeyPrefix = "ratelimit"
	}
	if config.KeyExtractor == nil {
		config.KeyExtractor = defaultKeyExtractor
	}
	if config.Skipper == nil {
		config.Skipper = defaultSkipper
	}

	// Pre-calculate limit based on window size
	limit := int(config.RequestsPerSecond * config.Window.Seconds())
	if limit < 1 {
		limit = config.Burst
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if this path should skip rate limiting
			if config.Skipper(c) {
				return next(c)
			}

			key := fmt.Sprintf("%s:%s", config.KeyPrefix, config.KeyExtractor(c))

			allowed, err := checkRateLimit(c, config.RedisClient, key, limit, config.Window)
			if err != nil {
				slog.WarnContext(c.Request().Context(), "rate limit check failed",
					slog.String("error", err.Error()),
					slog.String("key", key),
				)
				// Fail open - allow request on Redis error
				return next(c)
			}

			if !allowed {
				// Set Retry-After header
				c.Response().Header().Set("Retry-After", fmt.Sprintf("%d", int(config.Window.Seconds())))
				c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
				c.Response().Header().Set("X-RateLimit-Window", fmt.Sprintf("%ds", int(config.Window.Seconds())))

				return appresponse.Error(c, apperrors.RateLimited(
					fmt.Sprintf("Rate limit exceeded. Try again in %d seconds.", int(config.Window.Seconds())),
				))
			}

			// Add rate limit headers
			c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			c.Response().Header().Set("X-RateLimit-Window", fmt.Sprintf("%ds", int(config.Window.Seconds())))

			return next(c)
		}
	}
}

// checkRateLimit uses Redis sorted sets to implement a sliding window counter.
// It returns true if the request is allowed, false if rate limited.
func checkRateLimit(c echo.Context, client *redis.Client, key string, limit int, window time.Duration) (bool, error) {
	ctx := c.Request().Context()
	now := time.Now().UnixMilli()
	windowStart := now - window.Milliseconds()

	pipe := client.Pipeline()

	// Remove entries outside the current window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// Count current entries in the window
	countCmd := pipe.ZCard(ctx, key)

	// Add current request timestamp
	member := fmt.Sprintf("%d-%s", now, c.Request().Header.Get(echo.HeaderXRequestID))
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: member,
	})

	// Set expiry on the key to auto-cleanup
	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("redis pipeline failed: %w", err)
	}

	// Check if count before adding exceeded the limit
	return countCmd.Val() < int64(limit), nil
}

// defaultKeyExtractor creates a rate limit key from the client IP and request path.
// This provides per-endpoint rate limiting per client.
func defaultKeyExtractor(c echo.Context) string {
	clientIP := c.RealIP()
	path := c.Request().URL.Path
	return fmt.Sprintf("%s:%s", clientIP, path)
}

// defaultSkipper returns true for paths that should not be rate limited.
func defaultSkipper(c echo.Context) bool {
	path := c.Request().URL.Path

	// Skip health checks and metrics
	skipPaths := []string{
		"/health",
		"/healthz",
		"/ready",
		"/metrics",
		"/favicon.ico",
		"/robots.txt",
	}

	for _, skip := range skipPaths {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}

	return false
}
