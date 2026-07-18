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

// EmailVerificationRepository manages one-time email verification tokens.
type EmailVerificationRepository struct {
	db *sqlx.DB
}

func NewEmailVerificationRepository(db *sqlx.DB) *EmailVerificationRepository {
	return &EmailVerificationRepository{db: db}
}

func (r *EmailVerificationRepository) InvalidateActive(ctx context.Context, exec sqlx.ExtContext, userID uuid.UUID) error {
	_, err := exec.ExecContext(ctx, `
		UPDATE email_verification_tokens
		SET used_at = NOW()
		WHERE user_id = $1 AND used_at IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("invalidate email verification tokens: %w", err)
	}
	return nil
}

func (r *EmailVerificationRepository) Create(ctx context.Context, exec sqlx.ExtContext, token *domain.EmailVerificationToken) error {
	_, err := sqlx.NamedExecContext(ctx, exec, `
		INSERT INTO email_verification_tokens
			(id, user_id, token_hash, expires_at, used_at, created_at)
		VALUES (:id, :user_id, :token_hash, :expires_at, :used_at, :created_at)
	`, token)
	if err != nil {
		return fmt.Errorf("create email verification token: %w", err)
	}
	return nil
}

// Consume verifies and atomically consumes a token, then marks the user as
// verified. A transaction prevents a token from being used twice concurrently.
func (r *EmailVerificationRepository) Consume(ctx context.Context, tokenHash string, now time.Time) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin email verification transaction: %w", err)
	}
	defer tx.Rollback()

	var token domain.EmailVerificationToken
	err = tx.GetContext(ctx, &token, `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM email_verification_tokens
		WHERE token_hash = $1
		FOR UPDATE
	`, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrInvalidInput
		}
		return fmt.Errorf("get email verification token: %w", err)
	}
	if token.UsedAt != nil || !token.ExpiresAt.After(now) {
		return domain.ErrInvalidInput
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET email_verified_at = $1, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`, now, token.UserID); err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE email_verification_tokens
		SET used_at = $1
		WHERE id = $2
	`, now, token.ID); err != nil {
		return fmt.Errorf("consume email verification token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit email verification transaction: %w", err)
	}
	return nil
}
