package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventMemberStatusActive   = "active"
	EventMemberStatusInactive = "inactive"
)

// EventMember represents an event-scoped staff assignment.
type EventMember struct {
	Base
	TenantID   uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	EventID    uuid.UUID  `db:"event_id" json:"event_id"`
	UserID     uuid.UUID  `db:"user_id" json:"user_id"`
	Role       string     `db:"role" json:"role"`
	Status     string     `db:"status" json:"status"`
	InvitedBy  *uuid.UUID `db:"invited_by" json:"invited_by,omitempty"`
	AssignedAt time.Time  `db:"assigned_at" json:"assigned_at"`
}

// EventMemberCreateRequest assigns an existing tenant user to an event.
type EventMemberCreateRequest struct {
	UserID uuid.UUID `json:"user_id" validate:"required"`
	Role   string    `json:"role" validate:"required"`
}

// EventMemberUpdateRequest changes an event assignment role.
type EventMemberUpdateRequest struct {
	Role string `json:"role" validate:"required"`
}

// IsValidEventMemberRole returns true for roles that can be assigned per event.
func IsValidEventMemberRole(role string) bool {
	switch role {
	case RoleRSVPOfficer, RoleRegistrationOfficer, RoleUsher, RoleGiftOfficer, RoleViewer:
		return true
	default:
		return false
	}
}
