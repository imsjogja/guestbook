package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"guestflow/internal/domain"
	"guestflow/internal/email"
	"guestflow/internal/payment"
	"guestflow/internal/repository"

	"github.com/google/uuid"
)

// BillingService orchestrates subscription lifecycle and payment processing.
type BillingService struct {
	planRepo       *repository.PlanRepository
	subRepo        *repository.SubscriptionRepository
	payRepo        *repository.PaymentRepository
	tenantRepo     *repository.TenantRepository
	tenantUserRepo *repository.TenantUserRepository
	userRepo       *repository.UserRepository
	midtrans       *payment.Client
	mailer         email.Mailer
}

// NewBillingService creates a new BillingService.
func NewBillingService(
	planRepo *repository.PlanRepository,
	subRepo *repository.SubscriptionRepository,
	payRepo *repository.PaymentRepository,
	tenantRepo *repository.TenantRepository,
	tenantUserRepo *repository.TenantUserRepository,
	userRepo *repository.UserRepository,
	midtrans *payment.Client,
	mailer email.Mailer,
) *BillingService {
	return &BillingService{
		planRepo:       planRepo,
		subRepo:        subRepo,
		payRepo:        payRepo,
		tenantRepo:     tenantRepo,
		tenantUserRepo: tenantUserRepo,
		userRepo:       userRepo,
		midtrans:       midtrans,
		mailer:         mailer,
	}
}

// ListPlans returns all active plans grouped by billing cycle.
func (s *BillingService) ListPlans(ctx context.Context) ([]*domain.Plan, error) {
	plans, err := s.planRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	return plans, nil
}

// GetSubscriptionStatus returns the current subscription status for a tenant.
// It handles trial detection, trial expiry, and active subscription lookup.
func (s *BillingService) GetSubscriptionStatus(ctx context.Context, tenantID uuid.UUID) (*domain.SubscriptionStatusResponse, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get subscription status: %w", err)
	}

	resp := &domain.SubscriptionStatusResponse{}

	// Case 1: Active paid subscription
	sub, err := s.subRepo.GetActiveByTenantID(ctx, tenantID)
	if err == nil {
		plan, planErr := s.planRepo.GetByID(ctx, sub.PlanID)
		if planErr != nil {
			return nil, fmt.Errorf("get subscription plan: %w", planErr)
		}

		// Check if subscription has expired
		if sub.IsExpired() {
			_ = s.subRepo.UpdateStatus(ctx, sub.ID, domain.SubscriptionStatusExpired)
			_ = s.updateTenantStatus(ctx, tenantID, domain.TenantStatusSuspended)
			resp.Status = "suspended"
			resp.PlanName = plan.Name
			resp.DisplayName = plan.DisplayName
			resp.DaysLeft = -1
			resp.Features = plan.Features
			return resp, nil
		}

		daysLeft := sub.DaysUntilExpiry()
		var expiresStr *string
		if sub.ExpiresAt != nil {
			s := sub.ExpiresAt.Format(time.RFC3339)
			expiresStr = &s
		}

		resp.Status = "active"
		resp.PlanName = plan.Name
		resp.DisplayName = plan.DisplayName
		resp.BillingCycle = sub.BillingCycle
		resp.DaysLeft = daysLeft
		resp.ExpiresAt = expiresStr
		resp.MaxGuests = plan.MaxGuests
		resp.MaxEvents = plan.MaxEvents
		resp.MaxTeamMembers = plan.MaxTeamMembers
		resp.Features = plan.Features
		return resp, nil
	}

	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("get active subscription: %w", err)
	}

	// Case 2: Trial period
	if tenant.Status == domain.TenantStatusTrial && tenant.TrialEndsAt != nil {
		if tenant.IsTrialExpired() {
			_ = s.updateTenantStatus(ctx, tenantID, domain.TenantStatusSuspended)
			resp.Status = "trial_expired"
			resp.PlanName = "trial"
			resp.DisplayName = "Trial Berakhir"
			resp.DaysLeft = -1
			resp.Features = domain.PlanFeatures{} // no features
			return resp, nil
		}

		daysLeft := int(time.Until(*tenant.TrialEndsAt).Hours() / 24)
		expiresStr := tenant.TrialEndsAt.Format(time.RFC3339)

		// Trial has full Pro features
		resp.Status = "trial"
		resp.PlanName = "trial"
		resp.DisplayName = "Trial"
		resp.BillingCycle = "trial"
		resp.DaysLeft = daysLeft
		resp.ExpiresAt = &expiresStr
		resp.Features = domain.PlanFeatures{
			WhatsAppCampaign: true,
			CustomTemplate:   true,
			Webhook:          true,
			AdvancedReports:  true,
			RemoveBranding:   false,
			PrioritySupport:  false,
		}
		return resp, nil
	}

	// Case 3: Suspended/Cancelled
	resp.Status = string(tenant.Status)
	resp.PlanName = "none"
	resp.DisplayName = "Tidak Aktif"
	resp.DaysLeft = -1
	resp.Features = domain.PlanFeatures{}
	return resp, nil
}

// Checkout initiates a Midtrans Snap payment transaction.
func (s *BillingService) Checkout(ctx context.Context, tenantID, userID uuid.UUID, req domain.CheckoutRequest) (*domain.CheckoutResponse, error) {
	// Get plan
	plan, err := s.planRepo.GetByID(ctx, req.PlanID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	// Validate billing cycle matches plan
	if plan.BillingCycle != req.BillingCycle {
		return nil, fmt.Errorf("billing cycle mismatch: plan is %s but requested %s", plan.BillingCycle, req.BillingCycle)
	}

	// Get user info for Midtrans customer details
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user for checkout: %w", err)
	}

	// Create Midtrans order
	orderID := payment.CreateOrderID(tenantID)
	snapToken, err := s.midtrans.CreateTransaction(payment.CreateSnapRequest{
		OrderID:       orderID,
		AmountIDR:     int64(plan.PriceIDR),
		PlanName:      plan.DisplayName,
		BillingCycle:  req.BillingCycle,
		TenantID:      tenantID,
		CustomerName:  user.FullName,
		CustomerEmail: user.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("create midtrans transaction: %w", err)
	}

	// Record pending payment
	now := time.Now().UTC()
	expiredAt := now.Add(24 * time.Hour)
	pay := &domain.Payment{
		ID:              uuid.New(),
		TenantID:        tenantID,
		PlanID:          plan.ID,
		MidtransOrderID: orderID,
		AmountIDR:       plan.PriceIDR,
		BillingCycle:    req.BillingCycle,
		Status:          domain.PaymentStatusPending,
		ExpiredAt:       &expiredAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.payRepo.Create(ctx, pay); err != nil {
		slog.ErrorContext(ctx, "failed to record pending payment", slog.String("error", err.Error()))
		// Non-fatal: Midtrans already has the transaction
	}

	return &domain.CheckoutResponse{
		SnapToken:       snapToken,
		MidtransOrderID: orderID,
		PlanName:        plan.DisplayName,
		AmountIDR:       plan.PriceIDR,
	}, nil
}

// HandleWebhookNotification processes incoming Midtrans payment notifications.
func (s *BillingService) HandleWebhookNotification(ctx context.Context, payload payment.NotificationPayload, rawBody []byte) error {
	// Verify signature
	if !s.midtrans.VerifySignature(
		payload.OrderID,
		"200", // status code for settlement
		payload.GrossAmount,
		payload.SignatureKey,
	) {
		// Try other status codes
		verified := false
		for _, code := range []string{"200", "201", "407"} {
			if s.midtrans.VerifySignature(payload.OrderID, code, payload.GrossAmount, payload.SignatureKey) {
				verified = true
				break
			}
		}
		if !verified {
			return fmt.Errorf("invalid midtrans signature for order %s", payload.OrderID)
		}
	}

	// Get existing payment record
	existingPay, err := s.payRepo.GetByMidtransOrderID(ctx, payload.OrderID)
	if err != nil {
		return fmt.Errorf("payment record not found for order %s", payload.OrderID)
	}

	// Parse raw notification into JSONMap
	var rawMap domain.JSONMap
	_ = json.Unmarshal(rawBody, &rawMap)

	// Handle based on transaction status
	switch {
	case payment.IsSuccessStatus(payload.TransactionStatus, payload.FraudStatus):
		return s.activateSubscription(ctx, existingPay, payload, rawMap)

	case payment.IsFailedStatus(payload.TransactionStatus):
		return s.payRepo.UpdateOnFailed(ctx, payload.OrderID, domain.PaymentStatusFailed, rawMap)

	case payment.IsExpiredStatus(payload.TransactionStatus):
		return s.payRepo.UpdateOnFailed(ctx, payload.OrderID, domain.PaymentStatusExpired, rawMap)
	}

	return nil
}

// activateSubscription creates or extends the tenant's subscription after successful payment.
func (s *BillingService) activateSubscription(ctx context.Context, pay *domain.Payment, notif payment.NotificationPayload, rawMap domain.JSONMap) error {
	// Expire any existing active subscriptions
	_ = s.subRepo.ExpireAllActive(ctx, pay.TenantID)

	// Calculate expiry date
	plan, err := s.planRepo.GetByID(ctx, pay.PlanID)
	if err != nil {
		return fmt.Errorf("get plan for activation: %w", err)
	}

	now := time.Now().UTC()
	var expiresAt time.Time
	switch pay.BillingCycle {
	case domain.BillingCycleYearly:
		expiresAt = now.AddDate(1, 0, 0)
	default: // monthly
		expiresAt = now.AddDate(0, 1, 0)
	}

	orderID := notif.OrderID
	txID := notif.TransactionID

	// Create new active subscription
	sub := &domain.Subscription{
		ID:                    uuid.New(),
		TenantID:              pay.TenantID,
		PlanID:                plan.ID,
		Status:                domain.SubscriptionStatusActive,
		BillingCycle:          pay.BillingCycle,
		StartedAt:             now,
		ExpiresAt:             &expiresAt,
		MidtransOrderID:       &orderID,
		MidtransTransactionID: &txID,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := s.subRepo.Create(ctx, sub); err != nil {
		return fmt.Errorf("create subscription: %w", err)
	}

	// Extract VA number if available
	vaNumber := ""
	if len(notif.VANumbers) > 0 {
		vaNumber = notif.VANumbers[0].VANumber
	}

	// Update payment record
	if vaNumber != "" {
		_ = s.payRepo.UpdateOnSuccess(ctx, orderID, txID, notif.PaymentType, sub.ID, rawMap)
	} else {
		_ = s.payRepo.UpdateOnSuccess(ctx, orderID, txID, notif.PaymentType, sub.ID, rawMap)
	}

	// Send email receipt asynchronously
	go s.sendReceiptEmail(context.Background(), pay.TenantID, pay, plan)

	// Update tenant status to 'active'
	return s.updateTenantStatus(ctx, pay.TenantID, domain.TenantStatusActive)
}

// updateTenantStatus updates the tenant's status field.
func (s *BillingService) updateTenantStatus(ctx context.Context, tenantID uuid.UUID, status string) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	tenant.Status = status
	return s.tenantRepo.Update(ctx, tenant)
}

// GetPaymentHistory returns the list of payments for a tenant.
func (s *BillingService) GetPaymentHistory(ctx context.Context, tenantID uuid.UUID) ([]*domain.Payment, error) {
	return s.payRepo.ListByTenantID(ctx, tenantID)
}

func (s *BillingService) sendReceiptEmail(ctx context.Context, tenantID uuid.UUID, pay *domain.Payment, plan *domain.Plan) {
	// Find all owners/admins for this tenant
	members, err := s.tenantUserRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return
	}

	for _, member := range members {
		if member.Role == "owner" || member.Role == "admin" {
			user, err := s.userRepo.GetByID(ctx, member.UserID)
			if err == nil {
				subject := "Kwitansi Pembayaran GuestFlow: " + plan.DisplayName
				body := fmt.Sprintf("Halo %s,\n\nTerima kasih, pembayaran Anda sebesar Rp %d untuk paket %s (%s) telah berhasil.\nLangganan Anda sekarang aktif.\n\nSalam,\nTim GuestFlow", user.FullName, pay.AmountIDR, plan.DisplayName, pay.BillingCycle)
				_ = s.mailer.Send(context.Background(), user.Email, subject, body)
			}
		}
	}
}

// ProcessExpiredSubscriptions checks and updates tenants whose trial or subscription has expired.
// It also sends reminder emails for subscriptions ending in 3 days.
func (s *BillingService) ProcessExpiredSubscriptions(ctx context.Context) error {
	// Process trials that just expired
	tenants, err := s.tenantRepo.ListActive(ctx)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, t := range tenants {
		if t.Status == domain.TenantStatusTrial && t.TrialEndsAt != nil && t.TrialEndsAt.Before(now) {
			t.Status = "trial_expired"
			_ = s.tenantRepo.Update(ctx, t)
		}
	}

	// Process active subscriptions
	activeSubs, err := s.subRepo.ListActive(ctx)
	if err != nil {
		return err
	}

	reminderTime := now.AddDate(0, 0, 3)

	for _, sub := range activeSubs {
		if sub.ExpiresAt == nil {
			continue
		}

		if sub.ExpiresAt.Before(now) {
			// Expired! Update tenant status
			_ = s.subRepo.UpdateStatus(ctx, sub.ID, domain.SubscriptionStatusExpired)
			_ = s.updateTenantStatus(ctx, sub.TenantID, domain.TenantStatusSuspended)
			continue
		}

		// Check if it expires in exactly 3 days (within a 24h window for cron)
		if sub.ExpiresAt.After(now) && sub.ExpiresAt.Before(reminderTime) {
			// In a real app we'd check if we already sent the reminder (e.g. by storing a flag in DB).
			// Here we just send it if it's within the 3 day window.
			s.sendReminderEmail(ctx, sub.TenantID, sub)
		}
	}

	return nil
}

func (s *BillingService) sendReminderEmail(ctx context.Context, tenantID uuid.UUID, sub *domain.Subscription) {
	members, err := s.tenantUserRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return
	}

	for _, member := range members {
		if member.Role == "owner" || member.Role == "admin" {
			user, err := s.userRepo.GetByID(ctx, member.UserID)
			if err == nil {
				subject := "Pengingat: Langganan GuestFlow Anda akan segera berakhir"
				body := fmt.Sprintf("Halo %s,\n\nMasa berlangganan paket Anda akan berakhir pada %s.\nHarap segera lakukan perpanjangan melalui halaman Paket & Penagihan agar akses Anda tidak terputus.\n\nSalam,\nTim GuestFlow", user.FullName, sub.ExpiresAt.Format("02 Jan 2006"))
				_ = s.mailer.Send(context.Background(), user.Email, subject, body)
			}
		}
	}
}
