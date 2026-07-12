package domain

import (
	"time"

	"github.com/google/uuid"
)

// Base provides common fields for all domain models.
type Base struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

// NewBase creates a new Base with generated ID and timestamps.
func NewBase() Base {
	now := time.Now().UTC()
	return Base{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Touch updates the UpdatedAt timestamp.
func (b *Base) Touch() {
	b.UpdatedAt = time.Now().UTC()
}
