package domain

import (
	"time"

	"github.com/google/uuid"
)

// Check-in methods
const (
	CheckinMethodQRScan = "qr_scan"
	CheckinMethodManual = "manual_search"
	CheckinMethodWalkin = "walk_in"
	CheckinMethodKiosk  = "kiosk"
)

// Check-in statuses
const (
	CheckinStatusSuccess    = "success"
	CheckinStatusDuplicate  = "duplicate"
	CheckinStatusInvalid    = "invalid"
	CheckinStatusRevoked    = "revoked"
	CheckinStatusWrongEvent = "wrong_event"
	CheckinStatusExpired    = "expired"
)

// Checkin represents a check-in record
type Checkin struct {
	Base
	TenantID       uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	EventID        uuid.UUID  `db:"event_id" json:"event_id"`
	SessionID      *uuid.UUID `db:"session_id" json:"session_id,omitempty"`
	GuestID        uuid.UUID  `db:"guest_id" json:"guest_id"`
	EventGuestID   *uuid.UUID `db:"event_guest_id" json:"event_guest_id,omitempty"`
	InvitationID   *uuid.UUID `db:"invitation_id" json:"invitation_id,omitempty"`
	CredentialID   *uuid.UUID `db:"credential_id" json:"credential_id,omitempty"`
	Method         string     `db:"method" json:"method"`
	Status         string     `db:"status" json:"status"`
	DeviceID       *string    `db:"device_id" json:"device_id,omitempty"`
	GateID         *uuid.UUID `db:"gate_id" json:"gate_id,omitempty"`
	OfficerID      *uuid.UUID `db:"officer_id" json:"officer_id,omitempty"`
	ActualPax      int        `db:"actual_pax" json:"actual_pax"`
	Adults         int        `db:"adults" json:"adults"`
	Children       int        `db:"children" json:"children"`
	OverrideReason *string    `db:"override_reason" json:"override_reason,omitempty"`
	ApprovedBy     *uuid.UUID `db:"approved_by" json:"approved_by,omitempty"`
	IPAddress      *string    `db:"ip_address" json:"ip_address,omitempty"`
	Latitude       *float64   `db:"latitude" json:"latitude,omitempty"`
	Longitude      *float64   `db:"longitude" json:"longitude,omitempty"`
	Notes          *string    `db:"notes" json:"notes,omitempty"`
	OfflineSynced  bool       `db:"offline_synced" json:"offline_synced"`
}

// CheckinGate represents an entry point
type CheckinGate struct {
	Base
	TenantID uuid.UUID `db:"tenant_id" json:"tenant_id"`
	EventID  uuid.UUID `db:"event_id" json:"event_id"`
	Name     string    `db:"name" json:"name"`
	Code     string    `db:"code" json:"code"`
	Location *string   `db:"location" json:"location,omitempty"`
	IsActive bool      `db:"is_active" json:"is_active"`
}

// CheckinDevice represents a registered check-in device
type CheckinDevice struct {
	Base
	TenantID   uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	EventID    uuid.UUID  `db:"event_id" json:"event_id"`
	Name       string     `db:"name" json:"name"`
	DeviceCode string     `db:"device_code" json:"device_code"` // Unique device identifier
	DeviceType string     `db:"device_type" json:"device_type"` // tablet, phone, kiosk
	OfficerID  *uuid.UUID `db:"officer_id" json:"officer_id,omitempty"`
	LastSyncAt *time.Time `db:"last_sync_at" json:"last_sync_at,omitempty"`
	IsActive   bool       `db:"is_active" json:"is_active"`
}

// CheckinRequest input
type CheckinRequest struct {
	Method         string     `json:"method" validate:"required,oneof=qr_scan manual_search walk_in"`
	Token          string     `json:"token,omitempty"`    // For QR scan
	GuestID        *uuid.UUID `json:"guest_id,omitempty"` // For manual
	GateID         *uuid.UUID `json:"gate_id,omitempty"`
	DeviceID       string     `json:"device_id,omitempty"`
	ActualPax      int        `json:"actual_pax" validate:"required,min=1"`
	Adults         int        `json:"adults" validate:"omitempty,min=0"`
	Children       int        `json:"children" validate:"omitempty,min=0"`
	OverrideReason string     `json:"override_reason,omitempty"`
	ApprovedBy     *uuid.UUID `json:"approved_by,omitempty"`
	Notes          string     `json:"notes,omitempty"`
	Latitude       float64    `json:"latitude,omitempty"`
	Longitude      float64    `json:"longitude,omitempty"`
}

// WalkinRequest for walk-in registration
type WalkinRequest struct {
	FullName       string     `json:"full_name" validate:"required,min=2,max=255"`
	Phone          string     `json:"phone,omitempty"`
	Email          string     `json:"email,omitempty" validate:"omitempty,email"`
	GuestType      string     `json:"guest_type" validate:"required"`
	Segment        string     `json:"segment,omitempty"`
	ActualPax      int        `json:"actual_pax" validate:"required,min=1"`
	Adults         int        `json:"adults" validate:"omitempty,min=0"`
	Children       int        `json:"children" validate:"omitempty,min=0"`
	Reason         string     `json:"reason,omitempty"`
	OverrideReason string     `json:"override_reason,omitempty"`
	ApprovedBy     *uuid.UUID `json:"approved_by,omitempty"`
	Notes          string     `json:"notes,omitempty"`
}

// CheckinStats real-time stats
type CheckinStats struct {
	TotalExpected  int          `json:"total_expected"`
	TotalCheckedIn int          `json:"total_checked_in"`
	TotalPax       int          `json:"total_pax"`
	WalkIns        int          `json:"walk_ins"`
	NoShows        int          `json:"no_shows"`
	CheckInRate    float64      `json:"check_in_rate"`
	RecentCheckins []Checkin    `json:"recent_checkins,omitempty"`
	ByGate         []GateStat   `json:"by_gate,omitempty"`
	ByMethod       []MethodStat `json:"by_method,omitempty"`
	PeakHour       string       `json:"peak_hour,omitempty"`
}

// GateStat represents check-in counts per gate
type GateStat struct {
	GateID   uuid.UUID `db:"gate_id" json:"gate_id"`
	GateName string    `db:"gate_name" json:"gate_name"`
	Count    int       `db:"count" json:"count"`
	Pax      int       `db:"pax" json:"pax"`
}

// MethodStat represents check-in counts per method
type MethodStat struct {
	Method string `db:"method" json:"method"`
	Count  int    `db:"count" json:"count"`
}

// GuestSearchResult for manual check-in search
type GuestSearchResult struct {
	GuestID       uuid.UUID  `json:"guest_id"`
	FullName      string     `json:"full_name"`
	Nickname      *string    `json:"nickname,omitempty"`
	Phone         *string    `json:"phone,omitempty"`
	Email         *string    `json:"email,omitempty"`
	GuestType     string     `json:"guest_type"`
	Segment       *string    `json:"segment,omitempty"`
	InvitationID  *uuid.UUID `json:"invitation_id,omitempty"`
	HouseholdName *string    `json:"household_name,omitempty"`
	RSVPStatus    string     `json:"rsvp_status,omitempty"`
	IsCheckedIn   bool       `json:"is_checked_in"`
	TableName     *string    `json:"table_name,omitempty"`
	MaxPax        int        `json:"max_pax"`
}

// CheckinListParams for filtering and paginating check-in lists
type CheckinListParams struct {
	TenantID uuid.UUID
	EventID  uuid.UUID
	Method   string
	Status   string
	GateID   *uuid.UUID
	Page     int
	PerPage  int
}
