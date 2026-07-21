package domain

import (
	"time"

	"github.com/google/uuid"
)

// ─── Plan ──────────────────────────────────────────────────────────────────────

// PlanName constants for each subscription tier.
const (
	PlanNameStarter    = "starter"
	PlanNamePro        = "pro"
	PlanNameEnterprise = "enterprise"
)

// BillingCycle constants.
const (
	BillingCycleMonthly = "monthly"
	BillingCycleYearly  = "yearly"
)

// PlanFeatures represents the boolean feature flags for a plan.
type PlanFeatures struct {
	WhatsAppCampaign bool `json:"whatsapp_campaign"`
	CustomTemplate   bool `json:"custom_template"`
	Webhook          bool `json:"webhook"`
	AdvancedReports  bool `json:"advanced_reports"`
	RemoveBranding   bool `json:"remove_branding"`
	PrioritySupport  bool `json:"priority_support"`
}

// Plan represents a subscription plan/pricing tier.
type Plan struct {
	ID                   uuid.UUID    `db:"id"                     json:"id"`
	Name                 string       `db:"name"                   json:"name"`
	DisplayName          string       `db:"display_name"           json:"display_name"`
	BillingCycle         string       `db:"billing_cycle"          json:"billing_cycle"`
	PriceIDR             int          `db:"price_idr"              json:"price_idr"`
	MaxGuests            *int         `db:"max_guests"             json:"max_guests"`
	MaxEvents            *int         `db:"max_events"             json:"max_events"`
	MaxTeamMembers       *int         `db:"max_team_members"       json:"max_team_members"`
	MaxCampaignsPerMonth *int         `db:"max_campaigns_per_month" json:"max_campaigns_per_month"`
	MaxCSVImportRows     *int         `db:"max_csv_import_rows"    json:"max_csv_import_rows"`
	Features             PlanFeatures `db:"features"               json:"features"`
	IsActive             bool         `db:"is_active"              json:"is_active"`
	SortOrder            int          `db:"sort_order"             json:"sort_order"`
	CreatedAt            time.Time    `db:"created_at"             json:"created_at"`
	UpdatedAt            time.Time    `db:"updated_at"             json:"updated_at"`
}

// YearlyDiscountPercent returns the discount percentage for yearly vs monthly pricing.
func (p *Plan) YearlyDiscountPercent() int {
	return 17 // ~17% off monthly equivalent
}

// ─── Subscription ──────────────────────────────────────────────────────────────

// SubscriptionStatus constants.
const (
	SubscriptionStatusActive    = "active"
	SubscriptionStatusExpired   = "expired"
	SubscriptionStatusCancelled = "cancelled"
	SubscriptionStatusPending   = "pending"
)

// Subscription tracks the current active plan for a tenant.
type Subscription struct {
	ID                   uuid.UUID  `db:"id"                     json:"id"`
	TenantID             uuid.UUID  `db:"tenant_id"              json:"tenant_id"`
	PlanID               uuid.UUID  `db:"plan_id"                json:"plan_id"`
	Status               string     `db:"status"                 json:"status"`
	BillingCycle         string     `db:"billing_cycle"          json:"billing_cycle"`
	StartedAt            time.Time  `db:"started_at"             json:"started_at"`
	ExpiresAt            *time.Time `db:"expires_at"             json:"expires_at"`
	CancelledAt          *time.Time `db:"cancelled_at"           json:"cancelled_at"`
	MidtransOrderID      *string    `db:"midtrans_order_id"      json:"midtrans_order_id,omitempty"`
	MidtransTransactionID *string   `db:"midtrans_transaction_id" json:"midtrans_transaction_id,omitempty"`
	CreatedAt            time.Time  `db:"created_at"             json:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at"             json:"updated_at"`

	// Joined field (populated when fetching with plan details)
	Plan *Plan `db:"-" json:"plan,omitempty"`
}

// IsExpired returns true if the subscription has passed its expiry date.
func (s *Subscription) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*s.ExpiresAt)
}

// DaysUntilExpiry returns the number of days until expiry (negative if already expired).
func (s *Subscription) DaysUntilExpiry() int {
	if s.ExpiresAt == nil {
		return 9999
	}
	return int(time.Until(*s.ExpiresAt).Hours() / 24)
}

// ─── Payment ───────────────────────────────────────────────────────────────────

// PaymentStatus constants.
const (
	PaymentStatusPending   = "pending"
	PaymentStatusSuccess   = "success"
	PaymentStatusFailed    = "failed"
	PaymentStatusExpired   = "expired"
	PaymentStatusRefunded  = "refunded"
	PaymentStatusCancelled = "cancelled"
)

// Payment represents a single payment transaction record.
type Payment struct {
	ID                    uuid.UUID  `db:"id"                      json:"id"`
	TenantID              uuid.UUID  `db:"tenant_id"               json:"tenant_id"`
	SubscriptionID        *uuid.UUID `db:"subscription_id"         json:"subscription_id,omitempty"`
	PlanID                uuid.UUID  `db:"plan_id"                 json:"plan_id"`
	MidtransOrderID       string     `db:"midtrans_order_id"       json:"midtrans_order_id"`
	MidtransTransactionID *string    `db:"midtrans_transaction_id" json:"midtrans_transaction_id,omitempty"`
	AmountIDR             int        `db:"amount_idr"              json:"amount_idr"`
	BillingCycle          string     `db:"billing_cycle"           json:"billing_cycle"`
	Status                string     `db:"status"                  json:"status"`
	PaymentMethod         *string    `db:"payment_method"          json:"payment_method,omitempty"`
	VANumber              *string    `db:"va_number"               json:"va_number,omitempty"`
	PaidAt                *time.Time `db:"paid_at"                 json:"paid_at,omitempty"`
	ExpiredAt             *time.Time `db:"expired_at"              json:"expired_at,omitempty"`
	RawNotification       JSONMap    `db:"raw_notification"        json:"-"`
	CreatedAt             time.Time  `db:"created_at"              json:"created_at"`
	UpdatedAt             time.Time  `db:"updated_at"              json:"updated_at"`

	// Joined field
	Plan *Plan `db:"-" json:"plan,omitempty"`
}

// ─── Checkout Request/Response ─────────────────────────────────────────────────

// CheckoutRequest is the input payload for initiating a payment.
type CheckoutRequest struct {
	PlanID       uuid.UUID `json:"plan_id" validate:"required"`
	BillingCycle string    `json:"billing_cycle" validate:"required,oneof=monthly yearly"`
}

// CheckoutResponse is returned after creating a Midtrans Snap transaction.
type CheckoutResponse struct {
	SnapToken       string `json:"snap_token"`
	MidtransOrderID string `json:"midtrans_order_id"`
	PlanName        string `json:"plan_name"`
	AmountIDR       int    `json:"amount_idr"`
}

// SubscriptionStatus is the API response for the current subscription state.
type SubscriptionStatusResponse struct {
	Status          string  `json:"status"`           // 'trial', 'active', 'trial_expired', 'suspended', 'cancelled'
	PlanName        string  `json:"plan_name"`        // 'trial', 'starter', 'pro', 'enterprise'
	DisplayName     string  `json:"display_name"`     // 'Trial', 'Starter', 'Pro', 'Enterprise'
	BillingCycle    string  `json:"billing_cycle"`    // 'monthly', 'yearly', 'trial'
	DaysLeft        int     `json:"days_left"`        // days until expiry (-1 if expired, 9999 if unlimited)
	ExpiresAt       *string `json:"expires_at"`       // ISO8601 string
	MaxGuests       *int    `json:"max_guests"`
	MaxEvents       *int    `json:"max_events"`
	MaxTeamMembers  *int    `json:"max_team_members"`
	Features        PlanFeatures `json:"features"`
}
