package domain

import (
	"time"

	"github.com/google/uuid"
)

// EmailVerificationToken is the persisted, hashed representation of a token.
type EmailVerificationToken struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	TokenHash string     `db:"token_hash"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time  `db:"created_at"`
}
