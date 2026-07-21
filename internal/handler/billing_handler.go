package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"guestflow/internal/domain"
	"guestflow/internal/middleware"
	"guestflow/internal/payment"
	"guestflow/internal/service"
	appresponse "guestflow/pkg/response"

	"github.com/labstack/echo/v4"
)

// BillingHandler handles payment and subscription endpoints.
type BillingHandler struct {
	billingSvc *service.BillingService
	midtrans   *payment.Client
}

// NewBillingHandler creates a new BillingHandler.
func NewBillingHandler(billingSvc *service.BillingService, midtrans *payment.Client) *BillingHandler {
	return &BillingHandler{billingSvc: billingSvc, midtrans: midtrans}
}

// ListPlans returns all available subscription plans.
// GET /api/v1/billing/plans (public)
func (h *BillingHandler) ListPlans(c echo.Context) error {
	plans, err := h.billingSvc.ListPlans(c.Request().Context())
	if err != nil {
		return appresponse.InternalError(c, "Failed to load plans")
	}
	return appresponse.Success(c, map[string]interface{}{
		"plans":      plans,
		"client_key": h.midtrans.ClientKey(),
		"snap_url":   h.midtrans.SnapURL(),
	})
}

// GetSubscriptionStatus returns the current subscription and trial status.
// GET /api/v1/billing/subscription (protected, tenant-scoped)
func (h *BillingHandler) GetSubscriptionStatus(c echo.Context) error {
	tenantID, err := middleware.MustGetTenantIDFromContext(c.Request().Context())
	if err != nil {
		return appresponse.Unauthorized(c, "Tenant context required")
	}

	status, err := h.billingSvc.GetSubscriptionStatus(c.Request().Context(), tenantID)
	if err != nil {
		return appresponse.InternalError(c, "Failed to get subscription status")
	}
	return appresponse.Success(c, status)
}

// Checkout initiates a Midtrans Snap payment.
// POST /api/v1/billing/checkout (protected, tenant-scoped)
func (h *BillingHandler) Checkout(c echo.Context) error {
	tenantID, err := middleware.MustGetTenantIDFromContext(c.Request().Context())
	if err != nil {
		return appresponse.Unauthorized(c, "Tenant context required")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	var req domain.CheckoutRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return appresponse.BadRequest(c, err.Error())
	}

	resp, err := h.billingSvc.Checkout(c.Request().Context(), tenantID, userID, req)
	if err != nil {
		return appresponse.InternalError(c, err.Error())
	}
	return appresponse.Success(c, resp)
}

// HandleWebhook receives Midtrans payment notification callbacks.
// POST /api/v1/billing/webhook (PUBLIC — no JWT or tenant header required)
func (h *BillingHandler) HandleWebhook(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to read body"})
	}

	var payload payment.NotificationPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}

	if err := h.billingSvc.HandleWebhookNotification(c.Request().Context(), payload, body); err != nil {
		// Log but return 200 to prevent Midtrans retries on signature errors
		c.Logger().Errorf("webhook error for order %s: %v", payload.OrderID, err)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GetPaymentHistory returns the payment history for the current tenant.
// GET /api/v1/billing/history (protected, tenant-scoped)
func (h *BillingHandler) GetPaymentHistory(c echo.Context) error {
	tenantID, err := middleware.MustGetTenantIDFromContext(c.Request().Context())
	if err != nil {
		return appresponse.Unauthorized(c, "Tenant context required")
	}

	payments, err := h.billingSvc.GetPaymentHistory(c.Request().Context(), tenantID)
	if err != nil {
		return appresponse.InternalError(c, "Failed to get payment history")
	}
	return appresponse.Success(c, payments)
}


