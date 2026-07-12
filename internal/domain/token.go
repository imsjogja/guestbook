// Package domain contains shared domain models used across the GuestFlow application.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a stored refresh token in the database.
// Refresh tokens are stored as SHA-256 hashes for security.
// Tokens are revoked rather than physically deleted to maintain audit trails.
type RefreshToken struct {
	ID         uuid.UUID  `db:"id" json:"id"`
	UserID     uuid.UUID  `db:"user_id" json:"user_id"`
	TokenHash  string     `db:"token_hash" json:"-"`
	ExpiresAt  time.Time  `db:"expires_at" json:"expires_at"`
	RevokedAt  *time.Time `db:"revoked_at" json:"revoked_at,omitempty"`
	RevokedBy  *uuid.UUID `db:"revoked_by" json:"revoked_by,omitempty"`
	DeviceInfo *string    `db:"device_info" json:"device_info,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

// IsValid returns true if the token has not expired and has not been revoked.
func (rt *RefreshToken) IsValid() bool {
	now := time.Now().UTC()
	return now.Before(rt.ExpiresAt) && rt.RevokedAt == nil
}

// IsRevoked returns true if the token has been revoked.
func (rt *RefreshToken) IsRevoked() bool {
	return rt.RevokedAt != nil
}
