// Package middleware provides HTTP middleware for the GuestFlow Echo server.
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"guestflow/internal/auth"
)

// Context key type to avoid collisions with other context keys.
type contextKey string

const (
	// userContextKey is the key used to store user claims in the Echo context.
	userContextKey contextKey = "user_claims"
)

// JWTAuth returns middleware that validates JWT access tokens from the
// Authorization header (Bearer scheme). Valid tokens populate the context
// with user claims for downstream handlers.
//
// Invalid, expired, or missing tokens result in a 401 Unauthorized response.
func JWTAuth(jwtService *auth.JWTService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenString := extractBearerToken(c)
			if tokenString == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "missing authorization header",
				})
			}

			claims, err := jwtService.ValidateAccessToken(tokenString)
			if err != nil {
				status := http.StatusUnauthorized
				msg := "invalid token"

				if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, auth.ErrTokenExpired) {
					msg = "token has expired"
				}

				return c.JSON(status, map[string]string{"error": msg})
			}

			// Store validated claims in context
			c.Set(string(userContextKey), claims)

			return next(c)
		}
	}
}

// OptionalAuth returns middleware that attempts to validate a JWT access token
// but does not fail if the token is missing or invalid. This is useful for
// endpoints that have different behavior for authenticated vs anonymous users.
//
// When a valid token is present, user claims are populated in the context.
// When no token is present or the token is invalid, the context is left unchanged
// and the request proceeds normally.
func OptionalAuth(jwtService *auth.JWTService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenString := extractBearerToken(c)
			if tokenString == "" {
				// No token provided, continue without authentication
				return next(c)
			}

			claims, err := jwtService.ValidateAccessToken(tokenString)
			if err != nil {
				// Token invalid but optional, continue without authentication
				return next(c)
			}

			// Valid token, store claims in context
			c.Set(string(userContextKey), claims)

			return next(c)
		}
	}
}

// GetUserID extracts the user UUID from the Echo context.
// Returns uuid.Nil if no authenticated user is present.
func GetUserID(c echo.Context) uuid.UUID {
	claims := getClaims(c)
	if claims == nil {
		return uuid.Nil
	}
	return claims.UserID
}

// GetTenantID extracts the tenant UUID from the Echo context.
// Returns uuid.Nil if no tenant context is present.
func GetTenantID(c echo.Context) uuid.UUID {
	claims := getClaims(c)
	if claims == nil {
		return uuid.Nil
	}
	return claims.TenantID
}

// GetRole extracts the user's role from the Echo context.
// Returns an empty string if no role is present.
func GetRole(c echo.Context) string {
	claims := getClaims(c)
	if claims == nil {
		return ""
	}
	return claims.Role
}

// GetEmail extracts the user's email from the Echo context.
// Returns an empty string if no authenticated user is present.
func GetEmail(c echo.Context) string {
	claims := getClaims(c)
	if claims == nil {
		return ""
	}
	return claims.Email
}

// GetClaims extracts the full JWT claims from the Echo context.
// Returns nil if no claims are present.
func GetClaims(c echo.Context) *auth.Claims {
	return getClaims(c)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// extractBearerToken extracts the JWT token from the Authorization header.
// Expects the format: "Bearer <token>". Returns empty string if not found.
func extractBearerToken(c echo.Context) string {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return ""
	}

	return strings.TrimSpace(authHeader[len(prefix):])
}

// getClaims retrieves the JWT claims from the Echo context.
func getClaims(c echo.Context) *auth.Claims {
	val := c.Get(string(userContextKey))
	if val == nil {
		return nil
	}

	claims, ok := val.(*auth.Claims)
	if !ok {
		return nil
	}

	return claims
}
