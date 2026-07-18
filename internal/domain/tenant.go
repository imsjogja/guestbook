package domain

import (
	"time"

	"github.com/google/uuid"
)

// Tenant domain model represents a tenant (organization/workspace) in the multi-tenant SaaS.
type Tenant struct {
	Base
	Name         string     `db:"name" json:"name"`
	Slug         string     `db:"slug" json:"slug"`
	Description  *string    `db:"description" json:"description,omitempty"`
	LogoURL      *string    `db:"logo_url" json:"logo_url,omitempty"`
	PrimaryColor string     `db:"primary_color" json:"primary_color"`
	Settings     JSONMap    `db:"settings" json:"settings"`
	Status       string     `db:"status" json:"status"` // trial, active, suspended, cancelled
	TrialEndsAt  *time.Time `db:"trial_ends_at" json:"trial_ends_at,omitempty"`
}

// TenantCreateRequest is the input payload for creating a new tenant.
type TenantCreateRequest struct {
	Name string `json:"name" validate:"required,min=2,max=255"`
	Slug string `json:"slug" validate:"required,min=2,max=100,alphanumdash"`
}

// TenantUpdateRequest is the input payload for updating an existing tenant.
type TenantUpdateRequest struct {
	Name         string  `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Description  string  `json:"description,omitempty"`
	PrimaryColor string  `json:"primary_color,omitempty" validate:"omitempty,hexcolor"`
	Settings     JSONMap `json:"settings,omitempty"`
}

// TenantMemberCreateRequest is the input for manually adding an active member.
type TenantMemberCreateRequest struct {
	FullName string `json:"full_name" validate:"required,min=2,max=255"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Phone    string `json:"phone,omitempty" validate:"omitempty,max=32"`
	Role     string `json:"role" validate:"required"`
}

// TenantRoleUpdateRequest is the input for updating a member's role.
type TenantRoleUpdateRequest struct {
	Role string `json:"role" validate:"required"`
}

// TenantMembership represents a user's membership within a tenant.
type TenantMembership struct {
	Base
	TenantID  uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	UserID    uuid.UUID  `db:"user_id" json:"user_id"`
	Role      string     `db:"role" json:"role"`
	InvitedBy *uuid.UUID `db:"invited_by" json:"invited_by,omitempty"`
	InvitedAt *time.Time `db:"invited_at" json:"invited_at,omitempty"`
	JoinedAt  *time.Time `db:"joined_at" json:"joined_at,omitempty"`
	Status    string     `db:"status" json:"status"` // active, pending, inactive
}

// TenantStatus constants.
const (
	TenantStatusTrial     = "trial"
	TenantStatusActive    = "active"
	TenantStatusSuspended = "suspended"
	TenantStatusCancelled = "cancelled"
)

// MembershipStatus constants.
const (
	MembershipStatusActive   = "active"
	MembershipStatusPending  = "pending"
	MembershipStatusInactive = "inactive"
)

// NewTenant creates a new Tenant with default values.
func NewTenant(name, slug string, createdBy uuid.UUID) *Tenant {
	now := time.Now().UTC()
	trialEndsAt := now.AddDate(0, 0, 14) // 14-day trial
	return &Tenant{
		Base:         NewBase(),
		Name:         name,
		Slug:         slug,
		PrimaryColor: "#3B82F6",
		Settings:     make(JSONMap),
		Status:       TenantStatusTrial,
		TrialEndsAt:  &trialEndsAt,
	}
}

// NewTenantMembership creates a new TenantMembership.
func NewTenantMembership(tenantID, userID uuid.UUID, role string, invitedBy *uuid.UUID) *TenantMembership {
	now := time.Now().UTC()
	var invitedAt *time.Time
	if invitedBy != nil {
		invitedAt = &now
	}
	return &TenantMembership{
		Base:      NewBase(),
		TenantID:  tenantID,
		UserID:    userID,
		Role:      role,
		InvitedBy: invitedBy,
		InvitedAt: invitedAt,
		Status:    MembershipStatusActive,
	}
}

// IsActive returns true if the tenant is in an active state.
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive || t.Status == TenantStatusTrial
}

// IsTrialExpired returns true if the trial period has ended.
func (t *Tenant) IsTrialExpired() bool {
	if t.Status != TenantStatusTrial || t.TrialEndsAt == nil {
		return false
	}
	return time.Now().UTC().After(*t.TrialEndsAt)
}
