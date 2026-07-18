package domain

import (
	"time"

	"github.com/google/uuid"
)

// RSVP statuses
const (
	RSVPStatusNotSent      = "not_sent"
	RSVPStatusPending      = "pending"
	RSVPStatusAttending    = "attending"
	RSVPStatusNotAttending = "not_attending"
	RSVPStatusMaybe        = "maybe"
	RSVPStatusWaitlist     = "waitlist"
	RSVPStatusCancelled    = "cancelled"
	RSVPStatusNoResponse   = "no_response"
)

// ValidRSVPStatuses returns all valid RSVP statuses.
func ValidRSVPStatuses() []string {
	return []string{
		RSVPStatusNotSent,
		RSVPStatusPending,
		RSVPStatusAttending,
		RSVPStatusNotAttending,
		RSVPStatusMaybe,
		RSVPStatusWaitlist,
		RSVPStatusCancelled,
		RSVPStatusNoResponse,
	}
}

// IsValidRSVPStatus checks if the given status is a valid RSVP status.
func IsValidRSVPStatus(s string) bool {
	for _, v := range ValidRSVPStatuses() {
		if v == s {
			return true
		}
	}
	return false
}

// RSVPResponse represents a guest's RSVP.
type RSVPResponse struct {
	Base
	TenantID           uuid.UUID   `db:"tenant_id" json:"tenant_id"`
	EventID            uuid.UUID   `db:"event_id" json:"event_id"`
	InvitationID       uuid.UUID   `db:"invitation_id" json:"invitation_id"`
	GuestID            uuid.UUID   `db:"guest_id" json:"guest_id"`
	EventGuestID       *uuid.UUID  `db:"event_guest_id" json:"event_guest_id,omitempty"`
	Status             string      `db:"status" json:"status"`
	AttendingPax       int         `db:"attending_pax" json:"attending_pax"`
	Adults             int         `db:"adults" json:"adults"`
	Children           int         `db:"children" json:"children"`
	AttendingSessions  []uuid.UUID `db:"-" json:"attending_sessions,omitempty"`
	MenuChoice         *string     `db:"menu_choice" json:"menu_choice,omitempty"`
	Allergies          *string     `db:"allergies" json:"allergies,omitempty"`
	AccessibilityNeeds *string     `db:"accessibility_needs" json:"accessibility_needs,omitempty"`
	Transportation     *string     `db:"transportation" json:"transportation,omitempty"`
	Notes              *string     `db:"notes" json:"notes,omitempty"`
	RespondedAt        *time.Time  `db:"responded_at" json:"responded_at,omitempty"`
	EditedAt           *time.Time  `db:"edited_at" json:"edited_at,omitempty"`
	EditedBy           *uuid.UUID  `db:"edited_by" json:"edited_by,omitempty"`
	IPAddress          *string     `db:"ip_address" json:"ip_address,omitempty"`
}

// RSVPResponseWithGuest extends RSVPResponse with guest details.
type RSVPResponseWithGuest struct {
	RSVPResponse
	GuestFullName string  `db:"guest_full_name" json:"guest_full_name"`
	GuestEmail    *string `db:"guest_email" json:"guest_email,omitempty"`
	GuestPhone    *string `db:"guest_phone" json:"guest_phone,omitempty"`
	GuestType     string  `db:"guest_type" json:"guest_type"`
}

// RSVPDashboard stats.
type RSVPDashboard struct {
	TotalInvited  int     `json:"total_invited"`
	TotalSent     int     `json:"total_sent"`
	Opened        int     `json:"opened"`
	Responded     int     `json:"responded"`
	Attending     int     `json:"attending"`
	AttendingPax  int     `json:"attending_pax"`
	NotAttending  int     `json:"not_attending"`
	Maybe         int     `json:"maybe"`
	NoResponse    int     `json:"no_response"`
	Waitlist      int     `json:"waitlist"`
	ResponseRate  float64 `json:"response_rate"`
	CapacityUsed  int     `json:"capacity_used"`
	CapacityTotal int     `json:"capacity_total"`
}

// RSVPSubmitRequest public form submission.
type RSVPSubmitRequest struct {
	Token              string      `json:"token" validate:"required"`
	Status             string      `json:"status" validate:"required,oneof=attending not_attending maybe"`
	AttendingPax       int         `json:"attending_pax" validate:"required,min=1"`
	Adults             int         `json:"adults" validate:"omitempty,min=0"`
	Children           int         `json:"children" validate:"omitempty,min=0"`
	AttendingSessions  []uuid.UUID `json:"attending_sessions,omitempty"`
	MenuChoice         string      `json:"menu_choice,omitempty"`
	Allergies          string      `json:"allergies,omitempty"`
	AccessibilityNeeds string      `json:"accessibility_needs,omitempty"`
	Transportation     string      `json:"transportation,omitempty"`
	Notes              string      `json:"notes,omitempty"`
}

// RSVPUpdateRequest officer manual update.
type RSVPUpdateRequest struct {
	Status             string      `json:"status" validate:"required,oneof=attending not_attending maybe waitlist cancelled"`
	AttendingPax       int         `json:"attending_pax" validate:"required,min=0"`
	Adults             int         `json:"adults" validate:"omitempty,min=0"`
	Children           int         `json:"children" validate:"omitempty,min=0"`
	AttendingSessions  []uuid.UUID `json:"attending_sessions,omitempty"`
	MenuChoice         string      `json:"menu_choice,omitempty"`
	Allergies          string      `json:"allergies,omitempty"`
	AccessibilityNeeds string      `json:"accessibility_needs,omitempty"`
	Transportation     string      `json:"transportation,omitempty"`
	Notes              string      `json:"notes,omitempty"`
}

// RSVPQuestion for custom RSVP questions.
type RSVPQuestion struct {
	Base
	EventID   uuid.UUID `db:"event_id" json:"event_id"`
	Question  string    `db:"question" json:"question"`
	Type      string    `db:"type" json:"type"` // text, choice, multichoice, number
	Options   []string  `db:"options" json:"options,omitempty"`
	Required  bool      `db:"required" json:"required"`
	SortOrder int       `db:"sort_order" json:"sort_order"`
}

// RSVPQuestionAnswer stores answers to custom RSVP questions.
type RSVPQuestionAnswer struct {
	Base
	RSVPID     uuid.UUID `db:"rsvp_id" json:"rsvp_id"`
	QuestionID uuid.UUID `db:"question_id" json:"question_id"`
	Answer     string    `db:"answer" json:"answer"`
}

// ErrRSVPNotFound is returned when an RSVP does not exist.
var ErrRSVPNotFound = NewDomainError("rsvp not found")

// ErrRSVPDeadlinePassed is returned when the RSVP deadline has passed.
var ErrRSVPDeadlinePassed = NewDomainError("rsvp deadline has passed")

// ErrEventAtCapacity is returned when the event has reached its capacity.
var ErrEventAtCapacity = NewDomainError("event has reached capacity")

// ErrInvalidRSVPStatus is returned when an invalid RSVP status is provided.
var ErrInvalidRSVPStatus = NewDomainError("invalid rsvp status")
