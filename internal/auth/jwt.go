// Package auth provides authentication utilities for GuestFlow including
// JWT token management, password hashing, and refresh token handling.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Default TTL values for JWT tokens.
const (
	// DefaultAccessTTL is the lifetime of an access token (15 minutes).
	DefaultAccessTTL = 15 * time.Minute
	// DefaultRefreshTTL is the lifetime of a refresh token (7 days).
	DefaultRefreshTTL = 7 * 24 * time.Hour
	// DefaultIssuer identifies the token issuer.
	DefaultIssuer = "guestflow"
)

// Common JWT validation errors.
var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token has expired")
	ErrInvalidSignature = errors.New("invalid token signature")
	ErrInvalidClaims    = errors.New("invalid token claims")
)

// JWTService provides JWT token generation and validation capabilities.
// It manages both access tokens (short-lived) and refresh tokens (long-lived).
type JWTService struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	issuer        string
}

// Claims defines the custom JWT claims used by GuestFlow.
// It embeds jwt.RegisteredClaims for standard JWT fields.
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Email    string    `json:"email"`
	TenantID uuid.UUID `json:"tenant_id,omitempty"`
	Role     string    `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// TokenPair holds both access and refresh tokens returned after authentication.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds until access token expires
}

// NewJWTService creates a new JWTService with the provided configuration.
//
// Parameters:
//   - accessSecret:  secret key for signing access tokens (must be kept secure)
//   - refreshSecret: secret key for signing refresh tokens (should differ from accessSecret)
//   - accessTTL:     lifetime duration for access tokens (e.g., 15 * time.Minute)
//   - refreshTTL:    lifetime duration for refresh tokens (e.g., 7 * 24 * time.Hour)
//
// Example:
//
//	jwtService := auth.NewJWTService(
//	    os.Getenv("JWT_ACCESS_SECRET"),
//	    os.Getenv("JWT_REFRESH_SECRET"),
//	    15*time.Minute,
//	    7*24*time.Hour,
//	)
func NewJWTService(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *JWTService {
	issuer := DefaultIssuer
	if accessTTL <= 0 {
		accessTTL = DefaultAccessTTL
	}
	if refreshTTL <= 0 {
		refreshTTL = DefaultRefreshTTL
	}
	return &JWTService{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		issuer:        issuer,
	}
}

// GenerateTokenPair creates a new access token and refresh token pair for the given user.
// The access token contains minimal claims (userID, email) while the refresh token
// is a simple bearer token for obtaining new access tokens.
func (s *JWTService) GenerateTokenPair(userID uuid.UUID, email string) (*TokenPair, error) {
	accessToken, err := s.GenerateAccessToken(userID, email, uuid.Nil, "")
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(userID, email)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.accessTTL.Seconds()),
	}, nil
}

// GenerateAccessToken creates a signed access token with the specified claims.
// Access tokens are short-lived and contain user context (tenant, role) for authorization.
//
// Parameters:
//   - userID:   the authenticated user's UUID
//   - email:    the user's email address
//   - tenantID: the tenant/organization UUID (optional, use uuid.Nil if not applicable)
//   - role:     the user's role within the tenant (optional, use empty string if not applicable)
func (s *JWTService) GenerateAccessToken(userID uuid.UUID, email string, tenantID uuid.UUID, role string) (string, error) {
	now := time.Now().UTC()

	claims := Claims{
		UserID:   userID,
		Email:    email,
		TenantID: tenantID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.issuer,
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.accessSecret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return tokenString, nil
}

// ValidateAccessToken validates and parses an access token string.
// Returns the parsed claims if the token is valid, or an error if validation fails.
func (s *JWTService) ValidateAccessToken(tokenString string) (*Claims, error) {
	return s.validateToken(tokenString, s.accessSecret)
}

// ValidateRefreshToken validates and parses a refresh token string.
// Returns the parsed claims if the token is valid, or an error if validation fails.
func (s *JWTService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return s.validateToken(tokenString, s.refreshSecret)
}

// generateRefreshToken creates a signed refresh token for the given user.
// Refresh tokens have minimal claims and a longer expiration time.
func (s *JWTService) generateRefreshToken(userID uuid.UUID, email string) (string, error) {
	now := time.Now().UTC()

	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.issuer,
			Subject:   userID.String(),
			ID:        uuid.New().String(), // unique token ID for revocation support
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.refreshSecret)
	if err != nil {
		return "", fmt.Errorf("sign refresh token: %w", err)
	}

	return tokenString, nil
}

// validateToken parses and validates a JWT token string against the provided secret.
func (s *JWTService) validateToken(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is HMAC-SHA256 as expected
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method %v", ErrInvalidToken, token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		// Provide more specific error messages for common failure cases
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("%w: %v", ErrTokenExpired, err)
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, fmt.Errorf("%w: %v", ErrInvalidSignature, err)
		}
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, fmt.Errorf("%w: malformed token", ErrInvalidToken)
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}
