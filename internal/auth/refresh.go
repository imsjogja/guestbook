// Package auth provides authentication utilities for GuestFlow including
// JWT token management, password hashing, and refresh token handling.
package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"guestflow/internal/domain"
	"guestflow/pkg/crypto"
)

// RefreshTokenService manages the lifecycle of refresh tokens including
// creation, validation, revocation, and rotation.
//
// Tokens are stored as SHA-256 hashes in the database. The raw token is only
// returned once at creation time - it cannot be retrieved later.
type RefreshTokenService struct {
	db *sqlx.DB
}

// NewRefreshTokenService creates a new RefreshTokenService.
func NewRefreshTokenService(db *sqlx.DB) *RefreshTokenService {
	return &RefreshTokenService{db: db}
}

// Create generates a new refresh token, stores its hash in the database, and
// returns the raw token (which is the only time the raw value is available).
//
// The deviceInfo parameter can be used to store device fingerprint information
// for display in token management UIs (e.g., "Chrome on macOS").
func (s *RefreshTokenService) Create(ctx context.Context, userID uuid.UUID, deviceInfo string) (*domain.RefreshToken, string, error) {
	return s.create(ctx, s.db, userID, deviceInfo)
}

// CreateWithExecutor creates a refresh token using an existing database
// transaction. This keeps token creation atomic with user registration.
func (s *RefreshTokenService) CreateWithExecutor(ctx context.Context, executor sqlx.ExtContext, userID uuid.UUID, deviceInfo string) (*domain.RefreshToken, string, error) {
	return s.create(ctx, executor, userID, deviceInfo)
}

func (s *RefreshTokenService) create(ctx context.Context, executor sqlx.ExtContext, userID uuid.UUID, deviceInfo string) (*domain.RefreshToken, string, error) {
	// Generate a cryptographically secure random token
	rawToken := uuid.New().String() + "/" + uuid.New().String()

	// Hash the token with SHA-256 for database storage
	hash := crypto.SHA256Hash(rawToken)

	// Set expiration to 7 days from now
	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)

	token := &domain.RefreshToken{
		ID:         uuid.New(),
		UserID:     userID,
		TokenHash:  hash,
		ExpiresAt:  expiresAt,
		DeviceInfo: &deviceInfo,
		CreatedAt:  time.Now().UTC(),
	}

	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, device_info, created_at)
		VALUES (:id, :user_id, :token_hash, :expires_at, :device_info, :created_at)
	`
	_, err := sqlx.NamedExecContext(ctx, executor, query, token)
	if err != nil {
		return nil, "", fmt.Errorf("store refresh token: %w", err)
	}

	return token, rawToken, nil
}

// Validate checks if the provided raw token exists in the database, matches the
// stored hash, has not expired, and has not been revoked.
//
// Returns the stored refresh token record if valid.
func (s *RefreshTokenService) Validate(ctx context.Context, tokenString string) (*domain.RefreshToken, error) {
	hash := crypto.SHA256Hash(tokenString)

	var token domain.RefreshToken
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at, revoked_by, device_info, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
		LIMIT 1
	`
	err := s.db.GetContext(ctx, &token, query, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: token not found", ErrInvalidToken)
		}
		return nil, fmt.Errorf("lookup refresh token: %w", err)
	}

	if token.IsRevoked() {
		return nil, fmt.Errorf("%w: token has been revoked", ErrInvalidToken)
	}

	if !token.IsValid() {
		return nil, fmt.Errorf("%w: token has expired", ErrTokenExpired)
	}

	return &token, nil
}

// Revoke invalidates a single refresh token by marking it as revoked.
// The revokedBy parameter tracks which user initiated the revocation
// (useful for audit logs and "log out all devices" functionality).
func (s *RefreshTokenService) Revoke(ctx context.Context, tokenID uuid.UUID, revokedBy uuid.UUID) error {
	now := time.Now().UTC()

	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1, revoked_by = $2
		WHERE id = $3 AND revoked_at IS NULL
	`
	result, err := s.db.ExecContext(ctx, query, now, revokedBy, tokenID)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check revoke result: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("token already revoked or not found")
	}

	return nil
}

// RevokeAllUserTokens invalidates all active refresh tokens for a given user.
// This is useful for "log out everywhere" functionality or when a user's
// credentials have been compromised.
func (s *RefreshTokenService) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()

	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1, revoked_by = $1
		WHERE user_id = $2 AND revoked_at IS NULL
	`
	_, err := s.db.ExecContext(ctx, query, now, userID)
	if err != nil {
		return fmt.Errorf("revoke all user tokens: %w", err)
	}

	return nil
}

// Rotate performs a refresh token rotation: validates the old token, revokes it,
// and creates a new token pair. This is the recommended approach per OWASP guidelines
// to mitigate token theft and replay attacks.
//
// Returns the new raw refresh token and its database record.
// If rotation fails after the old token is revoked, the user will need to re-authenticate.
//
// The caller is responsible for generating a new JWT access token after rotation.
func (s *RefreshTokenService) Rotate(ctx context.Context, oldToken string, deviceInfo string) (*domain.RefreshToken, string, error) {
	// Step 1: Validate the old token
	storedToken, err := s.Validate(ctx, oldToken)
	if err != nil {
		return nil, "", fmt.Errorf("validate old token: %w", err)
	}

	// Step 2: Begin transaction to ensure atomicity
	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, "", fmt.Errorf("begin rotation transaction: %w", err)
	}
	defer tx.Rollback()

	// Step 3: Revoke the old token within the transaction
	now := time.Now().UTC()
	revokeQuery := `
		UPDATE refresh_tokens
		SET revoked_at = $1, revoked_by = $2
		WHERE id = $3 AND revoked_at IS NULL
	`
	_, err = tx.ExecContext(ctx, revokeQuery, now, storedToken.UserID, storedToken.ID)
	if err != nil {
		return nil, "", fmt.Errorf("revoke old token: %w", err)
	}

	// Step 4: Create new refresh token
	newRawToken := uuid.New().String() + "/" + uuid.New().String()
	newHash := crypto.SHA256Hash(newRawToken)
	newExpiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)

	newToken := &domain.RefreshToken{
		ID:         uuid.New(),
		UserID:     storedToken.UserID,
		TokenHash:  newHash,
		ExpiresAt:  newExpiresAt,
		DeviceInfo: &deviceInfo,
		CreatedAt:  time.Now().UTC(),
	}

	insertQuery := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, device_info, created_at)
		VALUES (:id, :user_id, :token_hash, :expires_at, :device_info, :created_at)
	`
	_, err = tx.NamedExecContext(ctx, insertQuery, newToken)
	if err != nil {
		return nil, "", fmt.Errorf("store new refresh token: %w", err)
	}

	// Step 5: Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, "", fmt.Errorf("commit rotation transaction: %w", err)
	}

	return newToken, newRawToken, nil
}
