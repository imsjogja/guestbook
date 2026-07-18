package domain

import (
	"time"

	"github.com/google/uuid"
)

// Event types
const (
	EventTypeWedding    = "wedding"
	EventTypeCorporate  = "corporate"
	EventTypeSeminar    = "seminar"
	EventTypeConference = "conference"
	EventTypeGathering  = "gathering"
	EventTypeGovernment = "government"
	EventTypeCommunity  = "community"
	EventTypeVIP        = "vip"
	EventTypeFamily     = "family"
)

// Event statuses
const (
	EventStatusDraft     = "draft"
	EventStatusPublished = "published"
	EventStatusOngoing   = "ongoing"
	EventStatusCompleted = "completed"
	EventStatusArchived  = "archived"
	EventStatusCancelled = "cancelled"
)

// ValidEventTypes returns all valid event types.
func ValidEventTypes() []string {
	return []string{
		EventTypeWedding,
		EventTypeCorporate,
		EventTypeSeminar,
		EventTypeConference,
		EventTypeGathering,
		EventTypeGovernment,
		EventTypeCommunity,
		EventTypeVIP,
		EventTypeFamily,
	}
}

// ValidEventStatuses returns all valid event statuses.
func ValidEventStatuses() []string {
	return []string{
		EventStatusDraft,
		EventStatusPublished,
		EventStatusOngoing,
		EventStatusCompleted,
		EventStatusArchived,
		EventStatusCancelled,
	}
}

// IsValidEventType checks if the given type is a valid event type.
func IsValidEventType(t string) bool {
	for _, v := range ValidEventTypes() {
		if v == t {
			return true
		}
	}
	return false
}

// IsValidEventStatus checks if the given status is a valid event status.
func IsValidEventStatus(s string) bool {
	for _, v := range ValidEventStatuses() {
		if v == s {
			return true
		}
	}
	return false
}

// Event represents an event within a tenant.
type Event struct {
	TenantBase
	Name              string     `db:"name" json:"name"`
	Type              string     `db:"type" json:"type"`
	Description       *string    `db:"description" json:"description,omitempty"`
	CoverURL          *string    `db:"cover_url" json:"cover_url,omitempty"`
	Status            string     `db:"status" json:"status"`
	StartDate         time.Time  `db:"start_date" json:"start_date"`
	EndDate           *time.Time `db:"end_date" json:"end_date,omitempty"`
	RSVPDeadline      *time.Time `db:"rsvp_deadline" json:"rsvp_deadline,omitempty"`
	Capacity          *int       `db:"capacity" json:"capacity,omitempty"`
	GuestCount        int        `db:"guest_count" json:"guest_count"`
	TargetInvites     *int       `db:"target_invites" json:"target_invites,omitempty"`
	TargetAttendance  *int       `db:"target_attendance" json:"target_attendance,omitempty"`
	PrimaryLocationID *uuid.UUID `db:"primary_location_id" json:"primary_location_id,omitempty"`
	DressCode         *string    `db:"dress_code" json:"dress_code,omitempty"`
	PrivacyNotice     *string    `db:"privacy_notice" json:"privacy_notice,omitempty"`
	GuestPolicy       *string    `db:"guest_policy" json:"guest_policy,omitempty"`
	Settings          JSONMap    `db:"settings" json:"settings"`
	CreatedBy         uuid.UUID  `db:"created_by" json:"created_by"`
}

// EventCreateRequest is the input payload for creating a new event.
type EventCreateRequest struct {
	Name             string     `json:"name" validate:"required,min=2,max=255"`
	Type             string     `json:"type" validate:"required,oneof=wedding corporate seminar conference gathering government community vip family"`
	Description      string     `json:"description,omitempty"`
	StartDate        time.Time  `json:"start_date" validate:"required"`
	EndDate          *time.Time `json:"end_date,omitempty"`
	RSVPDeadline     *time.Time `json:"rsvp_deadline,omitempty"`
	Capacity         *int       `json:"capacity,omitempty" validate:"omitempty,min=1"`
	TargetInvites    *int       `json:"target_invites,omitempty" validate:"omitempty,min=1"`
	TargetAttendance *int       `json:"target_attendance,omitempty" validate:"omitempty,min=1"`
	DressCode        string     `json:"dress_code,omitempty"`
	PrivacyNotice    string     `json:"privacy_notice,omitempty"`
	GuestPolicy      string     `json:"guest_policy,omitempty"`
}

// EventUpdateRequest is the input payload for updating an existing event.
type EventUpdateRequest struct {
	Name             string     `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Type             string     `json:"type,omitempty" validate:"omitempty,oneof=wedding corporate seminar conference gathering government community vip family"`
	Description      string     `json:"description,omitempty"`
	Status           string     `json:"status,omitempty" validate:"omitempty,oneof=draft published ongoing completed archived cancelled"`
	StartDate        *time.Time `json:"start_date,omitempty"`
	EndDate          *time.Time `json:"end_date,omitempty"`
	RSVPDeadline     *time.Time `json:"rsvp_deadline,omitempty"`
	Capacity         *int       `json:"capacity,omitempty" validate:"omitempty,min=1"`
	TargetInvites    *int       `json:"target_invites,omitempty" validate:"omitempty,min=1"`
	TargetAttendance *int       `json:"target_attendance,omitempty" validate:"omitempty,min=1"`
	DressCode        string     `json:"dress_code,omitempty"`
	PrivacyNotice    string     `json:"privacy_notice,omitempty"`
	GuestPolicy      string     `json:"guest_policy,omitempty"`
	Settings         JSONMap    `json:"settings,omitempty"`
}

// EventFilter provides filtering options for listing events.
type EventFilter struct {
	Status    string
	Type      string
	StartFrom *time.Time
	StartTo   *time.Time
	Page      int
	PerPage   int
}

// EventSession represents a sub-event session (e.g., Akad, Resepsi for weddings).
type EventSession struct {
	Base
	EventID     uuid.UUID  `db:"event_id" json:"event_id"`
	Name        string     `db:"name" json:"name"`
	Description *string    `db:"description" json:"description,omitempty"`
	StartTime   time.Time  `db:"start_time" json:"start_time"`
	EndTime     *time.Time `db:"end_time" json:"end_time,omitempty"`
	LocationID  *uuid.UUID `db:"location_id" json:"location_id,omitempty"`
	Capacity    *int       `db:"capacity" json:"capacity,omitempty"`
	SortOrder   int        `db:"sort_order" json:"sort_order"`
}

// EventSessionCreateRequest is the input payload for creating an event session.
type EventSessionCreateRequest struct {
	Name        string     `json:"name" validate:"required,min=2,max=255"`
	Description string     `json:"description,omitempty"`
	StartTime   time.Time  `json:"start_time" validate:"required"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	LocationID  *uuid.UUID `json:"location_id,omitempty"`
	Capacity    *int       `json:"capacity,omitempty" validate:"omitempty,min=1"`
	SortOrder   int        `json:"sort_order" default:"0"`
}

// EventSessionUpdateRequest is the input payload for updating an event session.
type EventSessionUpdateRequest struct {
	Name        string     `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Description string     `json:"description,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	LocationID  *uuid.UUID `json:"location_id,omitempty"`
	Capacity    *int       `json:"capacity,omitempty" validate:"omitempty,min=1"`
	SortOrder   *int       `json:"sort_order,omitempty"`
}

// EventLocation represents a venue or location for events.
type EventLocation struct {
	Base
	TenantID     uuid.UUID `db:"tenant_id" json:"tenant_id"`
	Name         string    `db:"name" json:"name"`
	Address      *string   `db:"address" json:"address,omitempty"`
	City         *string   `db:"city" json:"city,omitempty"`
	MapsURL      *string   `db:"maps_url" json:"maps_url,omitempty"`
	Latitude     *float64  `db:"latitude" json:"latitude,omitempty"`
	Longitude    *float64  `db:"longitude" json:"longitude,omitempty"`
	Instructions *string   `db:"instructions" json:"instructions,omitempty"`
}

// EventLocationCreateRequest is the input payload for creating an event location.
type EventLocationCreateRequest struct {
	Name         string   `json:"name" validate:"required,min=2,max=255"`
	Address      string   `json:"address,omitempty"`
	City         string   `json:"city,omitempty"`
	MapsURL      string   `json:"maps_url,omitempty"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
	Instructions string   `json:"instructions,omitempty"`
}

// EventLocationUpdateRequest is the input payload for updating an event location.
type EventLocationUpdateRequest struct {
	Name         string   `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Address      string   `json:"address,omitempty"`
	City         string   `json:"city,omitempty"`
	MapsURL      string   `json:"maps_url,omitempty"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
	Instructions string   `json:"instructions,omitempty"`
}

// ErrEventNotFound is returned when an event does not exist.
var ErrEventNotFound = NewDomainError("event not found")

// ErrEventSessionNotFound is returned when an event session does not exist.
var ErrEventSessionNotFound = NewDomainError("event session not found")

// ErrEventLocationNotFound is returned when an event location does not exist.
var ErrEventLocationNotFound = NewDomainError("event location not found")

// ErrEventInvalidStatusTransition is returned when an invalid status transition is attempted.
var ErrEventInvalidStatusTransition = NewDomainError("invalid event status transition")

// ErrEventCannotModify is returned when an event cannot be modified.
var ErrEventCannotModify = NewDomainError("event cannot be modified")

// ErrEventCannotDelete is returned when an event cannot be deleted.
var ErrEventCannotDelete = NewDomainError("event cannot be deleted")

// DomainError is a domain-specific error.
type DomainError struct {
	Message string
}

// Error implements the error interface.
func (e *DomainError) Error() string {
	return e.Message
}

// NewDomainError creates a new DomainError.
func NewDomainError(msg string) error {
	return &DomainError{Message: msg}
}
