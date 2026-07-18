package domain

import (
	"time"

	"github.com/google/uuid"
)

// Invitation statuses
const (
	InvitationStatusDraft     = "draft"
	InvitationStatusSent      = "sent"
	InvitationStatusOpened    = "opened"
	InvitationStatusResponded = "responded"
	InvitationStatusExpired   = "expired"
	InvitationStatusFailed    = "failed"
	InvitationStatusRevoked   = "revoked"
)

// ValidInvitationStatuses returns all valid invitation statuses.
func ValidInvitationStatuses() []string {
	return []string{
		InvitationStatusDraft,
		InvitationStatusSent,
		InvitationStatusOpened,
		InvitationStatusResponded,
		InvitationStatusExpired,
		InvitationStatusFailed,
		InvitationStatusRevoked,
	}
}

// IsValidInvitationStatus checks if the given status is a valid invitation status.
func IsValidInvitationStatus(s string) bool {
	for _, v := range ValidInvitationStatuses() {
		if v == s {
			return true
		}
	}
	return false
}

// Invitation represents a guest's invitation to an event.
type Invitation struct {
	Base
	TenantID        uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	EventID         uuid.UUID  `db:"event_id" json:"event_id"`
	GuestID         uuid.UUID  `db:"guest_id" json:"guest_id"`
	EventGuestID    *uuid.UUID `db:"event_guest_id" json:"event_guest_id,omitempty"`
	Token           string     `db:"token" json:"-"`      // Opaque token (private - returned only on creation)
	TokenHash       string     `db:"token_hash" json:"-"` // SHA-256 hash for lookup
	URL             string     `db:"url" json:"url"`      // Public URL
	MaxPax          int        `db:"max_pax" json:"max_pax"`
	Adults          int        `db:"adults" json:"adults"`
	Children        int        `db:"children" json:"children"`
	PlusOneAllowed  bool       `db:"plus_one_allowed" json:"plus_one_allowed"`
	PlusOneRequired bool       `db:"plus_one_required" json:"plus_one_required"`
	Status          string     `db:"status" json:"status"`
	SentAt          *time.Time `db:"sent_at" json:"sent_at,omitempty"`
	FailedReason    *string    `db:"failed_reason" json:"failed_reason,omitempty"`
	OpenedAt        *time.Time `db:"opened_at" json:"opened_at,omitempty"`
	RevokedAt       *time.Time `db:"revoked_at" json:"revoked_at,omitempty"`
	RevokedBy       *uuid.UUID `db:"revoked_by" json:"revoked_by,omitempty"`
	RevokeReason    *string    `db:"revoke_reason" json:"revoke_reason,omitempty"`
	ExpiresAt       *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	CreatedBy       uuid.UUID  `db:"created_by" json:"created_by"`
}

// InvitationCreateRequest input for creating invitations.
type InvitationCreateRequest struct {
	GuestIDs        []uuid.UUID `json:"guest_ids" validate:"required,min=1"`
	MaxPax          int         `json:"max_pax" validate:"required,min=1,max=50"`
	Adults          int         `json:"adults" validate:"omitempty,min=0"`
	Children        int         `json:"children" validate:"omitempty,min=0"`
	PlusOneAllowed  bool        `json:"plus_one_allowed"`
	PlusOneRequired bool        `json:"plus_one_required"`
	ExpiresAt       *time.Time  `json:"expires_at,omitempty"`
}

// InvitationListParams provides filtering and pagination for invitation lists.
type InvitationListParams struct {
	TenantID uuid.UUID
	EventID  uuid.UUID
	Status   string
	Page     int
	PerPage  int
}

// InvitationWithGuest extends Invitation with guest details for list views.
type InvitationWithGuest struct {
	Invitation
	GuestFullName              string     `db:"guest_full_name" json:"guest_full_name"`
	GuestEmail                 *string    `db:"guest_email" json:"guest_email,omitempty"`
	GuestPhone                 *string    `db:"guest_phone" json:"guest_phone,omitempty"`
	RSVPStatus                 string     `db:"rsvp_status" json:"rsvp_status"`
	DeliveryStatus             string     `db:"delivery_status" json:"delivery_status"`
	DeliveryChannel            *string    `db:"delivery_channel" json:"delivery_channel,omitempty"`
	DeliverySentAt             *time.Time `db:"delivery_sent_at" json:"delivery_sent_at,omitempty"`
	DeliveryDeliveredAt        *time.Time `db:"delivery_delivered_at" json:"delivery_delivered_at,omitempty"`
	DeliveryReadAt             *time.Time `db:"delivery_read_at" json:"delivery_read_at,omitempty"`
	DeliveryFailedAt           *time.Time `db:"delivery_failed_at" json:"delivery_failed_at,omitempty"`
	DeliveryErrorMessage       *string    `db:"delivery_error_message" json:"delivery_error_message,omitempty"`
	DeliveryExternalID         *string    `db:"delivery_external_id" json:"delivery_external_id,omitempty"`
	DeliveryProviderHTTPStatus *int       `db:"delivery_provider_http_status" json:"delivery_provider_http_status,omitempty"`
}

// QRCodeData represents the data returned for QR code generation.
type QRCodeData struct {
	URL       string    `json:"url"`
	TokenHash string    `json:"token_hash"`
	EventID   uuid.UUID `json:"event_id"`
	GuestID   uuid.UUID `json:"guest_id"`
}

// CredentialUsage represents a scan/usage of an invitation credential.
type CredentialUsage struct {
	Base
	InvitationID uuid.UUID  `db:"invitation_id" json:"invitation_id"`
	EventID      uuid.UUID  `db:"event_id" json:"event_id"`
	GuestID      uuid.UUID  `db:"guest_id" json:"guest_id"`
	Type         string     `db:"type" json:"type"` // checkin, rsvp, opened
	DeviceID     *string    `db:"device_id" json:"device_id,omitempty"`
	GateID       *uuid.UUID `db:"gate_id" json:"gate_id,omitempty"`
	OfficerID    *uuid.UUID `db:"officer_id" json:"officer_id,omitempty"`
	IPAddress    *string    `db:"ip_address" json:"ip_address,omitempty"`
	Metadata     JSONMap    `db:"metadata" json:"metadata,omitempty"`
}

// ErrInvitationNotFound is returned when an invitation does not exist.
var ErrInvitationNotFound = NewDomainError("invitation not found")

// ErrInvitationExpired is returned when an invitation has expired.
var ErrInvitationExpired = NewDomainError("invitation expired")

// ErrInvitationRevoked is returned when an invitation has been revoked.
var ErrInvitationRevoked = NewDomainError("invitation revoked")

// ErrInvitationAlreadyResponded is returned when an invitation has already been responded to.
var ErrInvitationAlreadyResponded = NewDomainError("invitation already responded")

// ErrTokenInvalid is returned when a token is invalid.
var ErrTokenInvalid = NewDomainError("invalid token")
