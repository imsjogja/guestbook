package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"guestflow/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PlanRepository handles database operations for plans.
type PlanRepository struct {
	db *sqlx.DB
}

// NewPlanRepository creates a new PlanRepository.
func NewPlanRepository(db *sqlx.DB) *PlanRepository {
	return &PlanRepository{db: db}
}

// ListActive returns all active plans ordered by sort_order.
func (r *PlanRepository) ListActive(ctx context.Context) ([]*domain.Plan, error) {
	rows, err := r.db.QueryxContext(ctx, `
		SELECT id, name, display_name, billing_cycle, price_idr,
		       max_guests, max_events, max_team_members,
		       max_campaigns_per_month, max_csv_import_rows,
		       features, is_active, sort_order, created_at, updated_at
		FROM plans
		WHERE is_active = TRUE
		ORDER BY sort_order ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list active plans: %w", err)
	}
	defer rows.Close()

	var plans []*domain.Plan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, fmt.Errorf("list active plans: scan: %w", err)
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
}

// GetByID retrieves a plan by its UUID.
func (r *PlanRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	row := r.db.QueryRowxContext(ctx, `
		SELECT id, name, display_name, billing_cycle, price_idr,
		       max_guests, max_events, max_team_members,
		       max_campaigns_per_month, max_csv_import_rows,
		       features, is_active, sort_order, created_at, updated_at
		FROM plans WHERE id = $1
	`, id)
	p, err := scanPlan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return p, err
}

// GetByNameAndCycle retrieves a plan by its name and billing cycle.
func (r *PlanRepository) GetByNameAndCycle(ctx context.Context, name, cycle string) (*domain.Plan, error) {
	row := r.db.QueryRowxContext(ctx, `
		SELECT id, name, display_name, billing_cycle, price_idr,
		       max_guests, max_events, max_team_members,
		       max_campaigns_per_month, max_csv_import_rows,
		       features, is_active, sort_order, created_at, updated_at
		FROM plans WHERE name = $1 AND billing_cycle = $2 AND is_active = TRUE
	`, name, cycle)
	p, err := scanPlan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return p, err
}

// rowScanner is a common interface for sqlx.Row and sqlx.Rows.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanPlan(row rowScanner) (*domain.Plan, error) {
	var p domain.Plan
	var featuresRaw []byte
	err := row.Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.BillingCycle, &p.PriceIDR,
		&p.MaxGuests, &p.MaxEvents, &p.MaxTeamMembers,
		&p.MaxCampaignsPerMonth, &p.MaxCSVImportRows,
		&featuresRaw, &p.IsActive, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(featuresRaw, &p.Features); err != nil {
		return nil, fmt.Errorf("unmarshal plan features: %w", err)
	}
	return &p, nil
}

// ─── SubscriptionRepository ────────────────────────────────────────────────────

// SubscriptionRepository handles database operations for subscriptions.
type SubscriptionRepository struct {
	db *sqlx.DB
}

// NewSubscriptionRepository creates a new SubscriptionRepository.
func NewSubscriptionRepository(db *sqlx.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// GetActivByTenantID returns the current active subscription for a tenant.
func (r *SubscriptionRepository) GetActiveByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.Subscription, error) {
	var s domain.Subscription
	err := r.db.QueryRowxContext(ctx, `
		SELECT id, tenant_id, plan_id, status, billing_cycle,
		       started_at, expires_at, cancelled_at,
		       midtrans_order_id, midtrans_transaction_id,
		       created_at, updated_at
		FROM subscriptions
		WHERE tenant_id = $1 AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`, tenantID).StructScan(&s)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get active subscription: %w", err)
	}
	return &s, nil
}

// Create inserts a new subscription record.
func (r *SubscriptionRepository) Create(ctx context.Context, s *domain.Subscription) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO subscriptions (
			id, tenant_id, plan_id, status, billing_cycle,
			started_at, expires_at, midtrans_order_id, midtrans_transaction_id,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`,
		s.ID, s.TenantID, s.PlanID, s.Status, s.BillingCycle,
		s.StartedAt, s.ExpiresAt, s.MidtransOrderID, s.MidtransTransactionID,
		s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create subscription: %w", err)
	}
	return nil
}

// UpdateStatus updates the status of a subscription.
func (r *SubscriptionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE subscriptions SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	return err
}

// ListActive lists all active subscriptions across all tenants.
func (r *SubscriptionRepository) ListActive(ctx context.Context) ([]*domain.Subscription, error) {
	var subs []*domain.Subscription
	err := r.db.SelectContext(ctx, &subs, `
		SELECT id, tenant_id, plan_id, status, billing_cycle,
		       started_at, expires_at, cancelled_at,
		       midtrans_order_id, midtrans_transaction_id,
		       created_at, updated_at
		FROM subscriptions
		WHERE status = 'active'
	`)
	if err != nil {
		return nil, fmt.Errorf("list active subscriptions: %w", err)
	}
	return subs, nil
}

// ExpireAllActive marks all active subscriptions for a tenant as expired.
func (r *SubscriptionRepository) ExpireAllActive(ctx context.Context, tenantID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE subscriptions SET status = 'expired', updated_at = NOW()
		 WHERE tenant_id = $1 AND status = 'active'`,
		tenantID,
	)
	return err
}

// ListByTenantID returns all subscriptions for a tenant (for history).
func (r *SubscriptionRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*domain.Subscription, error) {
	var subs []*domain.Subscription
	err := r.db.SelectContext(ctx, &subs, `
		SELECT id, tenant_id, plan_id, status, billing_cycle,
		       started_at, expires_at, cancelled_at,
		       midtrans_order_id, midtrans_transaction_id,
		       created_at, updated_at
		FROM subscriptions
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	return subs, nil
}

// ─── PaymentRepository ─────────────────────────────────────────────────────────

// PaymentRepository handles database operations for payments.
type PaymentRepository struct {
	db *sqlx.DB
}

// NewPaymentRepository creates a new PaymentRepository.
func NewPaymentRepository(db *sqlx.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create inserts a new payment record.
func (r *PaymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	notifJSON, _ := json.Marshal(p.RawNotification)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO payments (
			id, tenant_id, plan_id, midtrans_order_id, amount_idr,
			billing_cycle, status, expired_at, raw_notification, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`,
		p.ID, p.TenantID, p.PlanID, p.MidtransOrderID, p.AmountIDR,
		p.BillingCycle, p.Status, p.ExpiredAt, notifJSON, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create payment: %w", err)
	}
	return nil
}

// GetByMidtransOrderID retrieves a payment by Midtrans order ID.
func (r *PaymentRepository) GetByMidtransOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	var p domain.Payment
	var notifRaw []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, subscription_id, plan_id,
		       midtrans_order_id, midtrans_transaction_id, amount_idr, billing_cycle,
		       status, payment_method, va_number, paid_at, expired_at, raw_notification,
		       created_at, updated_at
		FROM payments WHERE midtrans_order_id = $1
	`, orderID).Scan(
		&p.ID, &p.TenantID, &p.SubscriptionID, &p.PlanID,
		&p.MidtransOrderID, &p.MidtransTransactionID, &p.AmountIDR, &p.BillingCycle,
		&p.Status, &p.PaymentMethod, &p.VANumber, &p.PaidAt, &p.ExpiredAt, &notifRaw,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get payment by order id: %w", err)
	}
	_ = json.Unmarshal(notifRaw, &p.RawNotification)
	return &p, nil
}

// UpdateOnSuccess updates payment after successful Midtrans notification.
func (r *PaymentRepository) UpdateOnSuccess(ctx context.Context, orderID, transactionID, method string, subID uuid.UUID, rawNotif domain.JSONMap) error {
	notifJSON, _ := json.Marshal(rawNotif)
	_, err := r.db.ExecContext(ctx, `
		UPDATE payments
		SET status = 'success',
		    midtrans_transaction_id = $1,
		    payment_method = $2,
		    subscription_id = $3,
		    paid_at = NOW(),
		    raw_notification = $4,
		    updated_at = NOW()
		WHERE midtrans_order_id = $5
	`, transactionID, method, subID, notifJSON, orderID)
	return err
}

// UpdateOnFailed marks payment as failed/expired/cancelled.
func (r *PaymentRepository) UpdateOnFailed(ctx context.Context, orderID, status string, rawNotif domain.JSONMap) error {
	notifJSON, _ := json.Marshal(rawNotif)
	_, err := r.db.ExecContext(ctx, `
		UPDATE payments
		SET status = $1, raw_notification = $2, updated_at = NOW()
		WHERE midtrans_order_id = $3
	`, status, notifJSON, orderID)
	return err
}

// ListByTenantID returns all payments for a tenant (for billing history UI).
func (r *PaymentRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*domain.Payment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.tenant_id, p.subscription_id, p.plan_id,
		       p.midtrans_order_id, p.midtrans_transaction_id, p.amount_idr, p.billing_cycle,
		       p.status, p.payment_method, p.va_number, p.paid_at, p.expired_at,
		       p.created_at, p.updated_at,
		       pl.name, pl.display_name
		FROM payments p
		JOIN plans pl ON pl.id = p.plan_id
		WHERE p.tenant_id = $1
		ORDER BY p.created_at DESC
		LIMIT 50
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		var pay domain.Payment
		var plan domain.Plan
		err := rows.Scan(
			&pay.ID, &pay.TenantID, &pay.SubscriptionID, &pay.PlanID,
			&pay.MidtransOrderID, &pay.MidtransTransactionID, &pay.AmountIDR, &pay.BillingCycle,
			&pay.Status, &pay.PaymentMethod, &pay.VANumber, &pay.PaidAt, &pay.ExpiredAt,
			&pay.CreatedAt, &pay.UpdatedAt,
			&plan.Name, &plan.DisplayName,
		)
		if err != nil {
			return nil, err
		}
		pay.Plan = &plan
		payments = append(payments, &pay)
	}
	return payments, rows.Err()
}
