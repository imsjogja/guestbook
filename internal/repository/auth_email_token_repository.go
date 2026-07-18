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

// AuthEmailTokenRepository manages password reset and magic login tokens.
type AuthEmailTokenRepository struct {
	db *sqlx.DB
}

func NewAuthEmailTokenRepository(db *sqlx.DB) *AuthEmailTokenRepository {
	return &AuthEmailTokenRepository{db: db}
}

func (r *AuthEmailTokenRepository) InvalidateActive(ctx context.Context, exec sqlx.ExtContext, userID uuid.UUID, purpose string) error {
	_, err := exec.ExecContext(ctx, `
		UPDATE auth_email_tokens
		SET used_at = NOW()
		WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL
	`, userID, purpose)
	if err != nil {
		return fmt.Errorf("invalidate auth email tokens: %w", err)
	}
	return nil
}

func (r *AuthEmailTokenRepository) Create(ctx context.Context, exec sqlx.ExtContext, token *domain.AuthEmailToken) error {
	_, err := sqlx.NamedExecContext(ctx, exec, `
		INSERT INTO auth_email_tokens
			(id, user_id, purpose, token_hash, expires_at, used_at, created_at)
		VALUES (:id, :user_id, :purpose, :token_hash, :expires_at, :used_at, :created_at)
	`, token)
	if err != nil {
		return fmt.Errorf("create auth email token: %w", err)
	}
	return nil
}

// Consume atomically marks a valid token as used and returns its user.
func (r *AuthEmailTokenRepository) Consume(ctx context.Context, tokenHash, purpose string, now time.Time) (uuid.UUID, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin auth email token transaction: %w", err)
	}
	defer tx.Rollback()

	var token domain.AuthEmailToken
	err = tx.GetContext(ctx, &token, `
		SELECT id, user_id, purpose, token_hash, expires_at, used_at, created_at
		FROM auth_email_tokens
		WHERE token_hash = $1 AND purpose = $2
		FOR UPDATE
	`, tokenHash, purpose)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, domain.ErrInvalidInput
		}
		return uuid.Nil, fmt.Errorf("get auth email token: %w", err)
	}
	if token.UsedAt != nil || !token.ExpiresAt.After(now) {
		return uuid.Nil, domain.ErrInvalidInput
	}

	if _, err := tx.ExecContext(ctx, `UPDATE auth_email_tokens SET used_at = $1 WHERE id = $2`, now, token.ID); err != nil {
		return uuid.Nil, fmt.Errorf("consume auth email token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("commit auth email token transaction: %w", err)
	}
	return token.UserID, nil
}
