// Package middleware provides tenant resolution and validation middleware
// for GuestFlow's multi-tenant architecture.
package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	apperrors "guestflow/pkg/errors"
	appresponse "guestflow/pkg/response"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

// tenantCtxKey is the unexported key type for storing tenant ID in request context.
type tenantCtxKey struct{}

// tenantContextKey is the singleton key instance used with context.WithValue.
var tenantContextKey = tenantCtxKey{}

// TenantResolverConfig holds configuration for the tenant resolution middleware.
type TenantResolverConfig struct {
	// DB is the database connection for validating tenant existence (required)
	DB *sqlx.DB

	// HeaderName is the HTTP header that contains the tenant ID (default: X-Tenant-ID)
	HeaderName string

	// Skipper defines paths that skip tenant resolution
	// Public endpoints like auth register/login are skipped by default
	Skipper func(c echo.Context) bool
}

// DefaultTenantResolverConfig returns a sensible default configuration.
func DefaultTenantResolverConfig(db *sqlx.DB) TenantResolverConfig {
	return TenantResolverConfig{
		DB:         db,
		HeaderName: "X-Tenant-ID",
		Skipper:    defaultTenantSkipper,
	}
}

// TenantResolver returns middleware that resolves and validates the tenant
// from the request. It reads the tenant ID from the X-Tenant-ID header,
// validates that the tenant exists and is active, and stores the tenant ID
// in the request context for downstream handlers.
func TenantResolver(config TenantResolverConfig) echo.MiddlewareFunc {
	if config.DB == nil {
		panic("tenant: DB is required")
	}
	if config.HeaderName == "" {
		config.HeaderName = "X-Tenant-ID"
	}
	if config.Skipper == nil {
		config.Skipper = defaultTenantSkipper
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if this path should skip tenant resolution
			if config.Skipper(c) {
				return next(c)
			}

			// Extract tenant ID from header
			tenantIDStr := c.Request().Header.Get(config.HeaderName)
			if tenantIDStr == "" {
				return appresponse.Error(c, apperrors.TenantRequired())
			}

			// Validate UUID format
			tenantID, err := uuid.Parse(tenantIDStr)
			if err != nil {
				slog.WarnContext(c.Request().Context(), "invalid tenant UUID format",
					slog.String("tenant_id", tenantIDStr),
					slog.String("error", err.Error()),
				)
				return appresponse.Error(c, apperrors.InvalidTenant())
			}

			// Validate tenant exists and is active
			valid, err := validateTenantExists(c.Request().Context(), config.DB, tenantID)
			if err != nil {
				slog.ErrorContext(c.Request().Context(), "tenant validation query failed",
					slog.String("error", err.Error()),
					slog.String("tenant_id", tenantID.String()),
				)
				return appresponse.Error(c, apperrors.Internal("Failed to validate tenant"))
			}

			if !valid {
				slog.WarnContext(c.Request().Context(), "tenant access denied",
					slog.String("tenant_id", tenantID.String()),
				)
				return appresponse.Error(c, apperrors.InvalidTenant())
			}

			// Store tenant ID in context for downstream use
			ctx := context.WithValue(c.Request().Context(), tenantContextKey, tenantID)
			c.SetRequest(c.Request().WithContext(ctx))

			// Also store in Echo context for easy access by other middleware
			c.Set("tenant_id", tenantID)

			return next(c)
		}
	}
}

// TenantIDFromContext retrieves the tenant ID from the request context.
// Returns the tenant UUID and true if a tenant is present, or a zero UUID
// and false if no tenant was resolved.
func TenantIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	val := ctx.Value(tenantContextKey)
	if val == nil {
		return uuid.Nil, false
	}
	tenantID, ok := val.(uuid.UUID)
	return tenantID, ok
}

// MustGetTenantIDFromContext retrieves the tenant ID from the request context.
// Returns the tenant UUID if present, or an error if no tenant was resolved.
func MustGetTenantIDFromContext(ctx context.Context) (uuid.UUID, error) {
	tenantID, ok := TenantIDFromContext(ctx)
	if !ok {
		return uuid.Nil, apperrors.TenantRequired()
	}
	return tenantID, nil
}

// validateTenantExists checks if a tenant with the given ID exists and has an
// active status in the database.
func validateTenantExists(ctx context.Context, db *sqlx.DB, tenantID uuid.UUID) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM tenants
			WHERE id = $1
			  AND status IN ('active', 'trial')
			  AND deleted_at IS NULL
		)
	`
	err := db.QueryRowContext(ctx, query, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to query tenant: %w", err)
	}
	return exists, nil
}

// defaultTenantSkipper returns true for paths that do not require tenant resolution.
func defaultTenantSkipper(c echo.Context) bool {
	path := c.Request().URL.Path
	method := c.Request().Method

	// Public authentication endpoints (no tenant needed)
	publicPaths := []string{
		"/api/v1/auth/register",
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
		"/api/v1/auth/forgot-password",
		"/api/v1/auth/reset-password",
	}

	for _, pp := range publicPaths {
		if path == pp {
			return true
		}
	}

	// Health, readiness, and metrics endpoints
	if strings.HasPrefix(path, "/health") ||
		strings.HasPrefix(path, "/healthz") ||
		strings.HasPrefix(path, "/ready") ||
		strings.HasPrefix(path, "/metrics") {
		return true
	}

	// Swagger/OpenAPI docs
	if strings.HasPrefix(path, "/swagger") ||
		strings.HasPrefix(path, "/api/docs") {
		return true
	}

	// OPTIONS requests (CORS preflight)
	if method == "OPTIONS" {
		return true
	}

	return false
}
