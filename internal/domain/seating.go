package domain

import (
	"time"

	"github.com/google/uuid"
)

// Seating modes
const (
	SeatingModeStanding  = "standing"
	SeatingModeTableOnly = "table_only"
	SeatingModeTableSeat = "table_and_seat"
	SeatingModeZone      = "zone_based"
	SeatingModeOpen      = "open_seating"
	SeatingModeVipLounge = "vip_lounge"
)

// Table shapes
const (
	TableShapeRound       = "round"
	TableShapeRectangular = "rectangular"
	TableShapeSquare      = "square"
	TableShapeOval        = "oval"
	TableShapeUShape      = "u_shape"
)

// VenueZone represents an area in the venue
type VenueZone struct {
	TenantBase
	EventID     uuid.UUID `db:"event_id" json:"event_id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description,omitempty"`
	SortOrder   int       `db:"sort_order" json:"sort_order"`
}

// Table represents a table
type Table struct {
	Base
	TenantID      uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	EventID       uuid.UUID  `db:"event_id" json:"event_id"`
	ZoneID        *uuid.UUID `db:"zone_id" json:"zone_id,omitempty"`
	Name          string     `db:"name" json:"name"`
	Capacity      int        `db:"capacity" json:"capacity"`
	Shape         string     `db:"shape" json:"shape"` // round, rectangular, square
	PositionX     *float64   `db:"position_x" json:"position_x,omitempty"`
	PositionY     *float64   `db:"position_y" json:"position_y,omitempty"`
	IsLocked      bool       `db:"is_locked" json:"is_locked"`
	Accessibility bool       `db:"accessibility" json:"accessibility"`
	VIPOnly       bool       `db:"vip_only" json:"vip_only"`
	Notes         *string    `db:"notes" json:"notes,omitempty"`
}

// SeatAssignment represents a guest assigned to a table
type SeatAssignment struct {
	TableID      uuid.UUID  `db:"table_id" json:"table_id"`
	GuestID      uuid.UUID  `db:"guest_id" json:"guest_id"`
	EventGuestID *uuid.UUID `db:"event_guest_id" json:"event_guest_id,omitempty"`
	SeatNumber   *int       `db:"seat_number" json:"seat_number,omitempty"`
	AssignedBy   uuid.UUID  `db:"assigned_by" json:"assigned_by"`
	AssignedAt   time.Time  `db:"assigned_at" json:"assigned_at"`
}

// TableCreateRequest input for creating a table
type TableCreateRequest struct {
	ZoneID        *uuid.UUID `json:"zone_id,omitempty"`
	Name          string     `json:"name" validate:"required,min=1,max=100"`
	Capacity      int        `json:"capacity" validate:"required,min=1,max=999"`
	Shape         string     `json:"shape,omitempty" validate:"omitempty,oneof=round rectangular square oval u_shape"`
	PositionX     *float64   `json:"position_x,omitempty"`
	PositionY     *float64   `json:"position_y,omitempty"`
	IsLocked      bool       `json:"is_locked,omitempty"`
	Accessibility bool       `json:"accessibility,omitempty"`
	VIPOnly       bool       `json:"vip_only,omitempty"`
	Notes         string     `json:"notes,omitempty"`
}

// TableUpdateRequest input for updating a table
type TableUpdateRequest struct {
	ZoneID        *uuid.UUID `json:"zone_id,omitempty"`
	Name          string     `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Capacity      *int       `json:"capacity,omitempty" validate:"omitempty,min=1,max=999"`
	Shape         string     `json:"shape,omitempty" validate:"omitempty,oneof=round rectangular square oval u_shape"`
	PositionX     *float64   `json:"position_x,omitempty"`
	PositionY     *float64   `json:"position_y,omitempty"`
	IsLocked      *bool      `json:"is_locked,omitempty"`
	Accessibility *bool      `json:"accessibility,omitempty"`
	VIPOnly       *bool      `json:"vip_only,omitempty"`
	Notes         string     `json:"notes,omitempty"`
}

// TableWithOccupancy extends Table with occupancy info
type TableWithOccupancy struct {
	Table
	Occupancy      int             `db:"occupancy" json:"occupancy"`
	AssignedGuests []AssignedGuest `json:"assigned_guests,omitempty"`
}

// AssignedGuest represents a guest assigned to a table for layout view
type AssignedGuest struct {
	GuestID    uuid.UUID `db:"guest_id" json:"guest_id"`
	FullName   string    `db:"full_name" json:"full_name"`
	GuestType  string    `db:"guest_type" json:"guest_type"`
	SeatNumber *int      `db:"seat_number" json:"seat_number,omitempty"`
	AssignedAt time.Time `db:"assigned_at" json:"assigned_at"`
}

// SeatingLayout represents the full seating layout for an event
type SeatingLayout struct {
	EventID     uuid.UUID            `json:"event_id"`
	Zones       []VenueZone          `json:"zones,omitempty"`
	Tables      []TableWithOccupancy `json:"tables"`
	Unassigned  int                  `json:"unassigned_guests"`
	TotalGuests int                  `json:"total_guests"`
}

// AssignGuestRequest input for assigning a guest to a table
type AssignGuestRequest struct {
	GuestID    uuid.UUID `json:"guest_id" validate:"required"`
	SeatNumber *int      `json:"seat_number,omitempty"`
}
