package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	AuthEmailTokenPasswordReset = "password_reset"
	AuthEmailTokenMagicLogin    = "magic_login"
)

// AuthEmailToken stores only the hashed value of a password reset or magic
// login token.
type AuthEmailToken struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	Purpose   string     `db:"purpose"`
	TokenHash string     `db:"token_hash"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time  `db:"created_at"`
}
