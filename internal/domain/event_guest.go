package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventGuestStatusActive    = "active"
	EventGuestStatusCancelled = "cancelled"

	EventGuestSourceManual        = "manual"
	EventGuestSourceImport        = "import"
	EventGuestSourceInvitation    = "invitation"
	EventGuestSourceRSVP          = "rsvp"
	EventGuestSourceCheckin       = "checkin"
	EventGuestSourceSeating       = "seating"
	EventGuestSourceCommunication = "communication"
	EventGuestSourceCopyEvent     = "copy_event"
	EventGuestSourceWalkIn        = "walk_in"
)

// EventGuest is the event-specific roster entry for a tenant guest.
type EventGuest struct {
	Base
	TenantID       uuid.UUID `db:"tenant_id" json:"tenant_id"`
	EventID        uuid.UUID `db:"event_id" json:"event_id"`
	GuestID        uuid.UUID `db:"guest_id" json:"guest_id"`
	Status         string    `db:"status" json:"status"`
	Source         string    `db:"source" json:"source"`
	MaxPax         int       `db:"max_pax" json:"max_pax"`
	Adults         int       `db:"adults" json:"adults"`
	Children       int       `db:"children" json:"children"`
	PlusOneAllowed bool      `db:"plus_one_allowed" json:"plus_one_allowed"`
	Notes          *string   `db:"notes" json:"notes,omitempty"`
	CreatedBy      uuid.UUID `db:"created_by" json:"created_by"`
	Guest          *Guest    `db:"-" json:"guest,omitempty"`
}

type EventGuestCreateRequest struct {
	GuestID        uuid.UUID `json:"guest_id" validate:"required"`
	Source         string    `json:"source,omitempty"`
	MaxPax         int       `json:"max_pax,omitempty" validate:"omitempty,min=1"`
	Adults         int       `json:"adults,omitempty" validate:"omitempty,min=0"`
	Children       int       `json:"children,omitempty" validate:"omitempty,min=0"`
	PlusOneAllowed bool      `json:"plus_one_allowed,omitempty"`
	Notes          string    `json:"notes,omitempty"`
}

type EventGuestListParams struct {
	TenantID uuid.UUID
	EventID  uuid.UUID
	Search   string
	Status   string
	Page     int
	PerPage  int
}

func (eg *EventGuest) Touch() {
	eg.UpdatedAt = time.Now().UTC()
}

func IsValidEventGuestSource(source string) bool {
	switch source {
	case EventGuestSourceManual, EventGuestSourceImport, EventGuestSourceInvitation,
		EventGuestSourceRSVP, EventGuestSourceCheckin, EventGuestSourceSeating,
		EventGuestSourceCommunication, EventGuestSourceCopyEvent, EventGuestSourceWalkIn:
		return true
	default:
		return false
	}
}
