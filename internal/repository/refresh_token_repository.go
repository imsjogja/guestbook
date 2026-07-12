// Package repository provides data access layer implementations for GuestFlow.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"guestflow/internal/domain"
)

// RefreshTokenRepository provides data access operations for refresh tokens.
// It handles creation, retrieval by hash, and revocation of refresh tokens.
type RefreshTokenRepository struct {
	db *sqlx.DB
}

// NewRefreshTokenRepository creates a new RefreshTokenRepository instance.
func NewRefreshTokenRepository(db *sqlx.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create persists a new refresh token record to the database.
// The token is stored as a SHA-256 hash, never as the raw token value.
func (r *RefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, revoked_by, device_info, created_at)
		VALUES (:id, :user_id, :token_hash, :expires_at, :revoked_at, :revoked_by, :device_info, :created_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

// GetByHash retrieves a refresh token by its SHA-256 hash value.
// Returns sql.ErrNoRows if no matching token is found.
func (r *RefreshTokenRepository) GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var token domain.RefreshToken
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at, revoked_by, device_info, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &token, query, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("get refresh token by hash: %w", err)
	}
	return &token, nil
}

// Revoke marks a specific refresh token as revoked at the current time.
// The revokedBy parameter records which user initiated the revocation for audit purposes.
// This operation is idempotent - revoking an already-revoked token succeeds.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID, revokedBy uuid.UUID) error {
	now := time.Now().UTC()

	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1, revoked_by = $2
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, now, revokedBy, id)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// RevokeAllForUser invalidates all active refresh tokens belonging to a user.
// This is typically used for "log out everywhere" functionality or security incidents.
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()

	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1, revoked_by = $1
		WHERE user_id = $2 AND revoked_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, now, userID)
	if err != nil {
		return fmt.Errorf("revoke all user refresh tokens: %w", err)
	}
	return nil
}
