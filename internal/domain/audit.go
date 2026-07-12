package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog represents a single audit log entry tracking mutations in the system.
type AuditLog struct {
	ID         uuid.UUID  `db:"id" json:"id"`
	TenantID   *uuid.UUID `db:"tenant_id" json:"tenant_id,omitempty"`
	UserID     *uuid.UUID `db:"user_id" json:"user_id,omitempty"`
	Action     string     `db:"action" json:"action"`
	EntityType string     `db:"entity_type" json:"entity_type"`
	EntityID   *uuid.UUID `db:"entity_id" json:"entity_id,omitempty"`
	OldValues  JSONMap    `db:"old_values" json:"old_values,omitempty"`
	NewValues  JSONMap    `db:"new_values" json:"new_values,omitempty"`
	IPAddress  *string    `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent  *string    `db:"user_agent" json:"user_agent,omitempty"`
	Metadata   JSONMap    `db:"metadata" json:"metadata,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

// Audit action constants.
const (
	AuditActionCreate  = "create"
	AuditActionUpdate  = "update"
	AuditActionDelete  = "delete"
	AuditActionInvite  = "invite"
	AuditActionRemove  = "remove"
	AuditActionLogin   = "login"
	AuditActionLogout  = "logout"
	AuditActionExport  = "export"
	AuditActionImport  = "import"
	AuditActionSend    = "send"
	AuditActionApprove = "approve"
	AuditActionReject  = "reject"
)

// Entity type constants for audit logging.
const (
	EntityTypeTenant     = "tenant"
	EntityTypeUser       = "user"
	EntityTypeGuest      = "guest"
	EntityTypeEvent      = "event"
	EntityTypeInvitation = "invitation"
	EntityTypeRSVP       = "rsvp"
	EntityTypeCheckin    = "checkin"
	EntityTypeSetting    = "setting"
	EntityTypeMembership = "membership"
)

// NewAuditLog creates a new AuditLog entry with a generated ID and timestamp.
func NewAuditLog() *AuditLog {
	return &AuditLog{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
	}
}
