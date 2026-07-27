package handler

import (
	"errors"
	"net/http"

	"guestflow/internal/service"
	appresponse "guestflow/pkg/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// WhatsAppIntegrationHandler exposes safe tenant-scoped provider settings.
type WhatsAppIntegrationHandler struct {
	service *service.WhatsAppIntegrationService
}

func NewWhatsAppIntegrationHandler(integrationService *service.WhatsAppIntegrationService) *WhatsAppIntegrationHandler {
	return &WhatsAppIntegrationHandler{service: integrationService}
}

func (h *WhatsAppIntegrationHandler) Get(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}
	status, err := h.service.GetStatus(c.Request().Context(), tenantID)
	if err != nil {
		if errors.Is(err, service.ErrWhatsAppDeviceConflict) {
			return appresponse.Conflict(c, "WhatsApp device sudah digunakan tenant lain")
		}
		return appresponse.InternalError(c, "Failed to retrieve WhatsApp integration")
	}
	return appresponse.Success(c, status)
}

func (h *WhatsAppIntegrationHandler) Update(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}
	var req service.WhatsAppIntegrationUpdateRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}
	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}
	status, err := h.service.Update(c.Request().Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, service.ErrWhatsAppIntegrationInvalid) {
			return appresponse.ValidationError(c, err.Error())
		}
		if errors.Is(err, service.ErrWhatsAppDeviceConflict) {
			return appresponse.Conflict(c, "WhatsApp device sudah digunakan tenant lain")
		}
		return appresponse.InternalError(c, "Failed to update WhatsApp integration")
	}
	return appresponse.Success(c, status)
}

// StartPairing starts a short-lived GOWA QR pairing session for the tenant.
func (h *WhatsAppIntegrationHandler) StartPairing(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}
	pairing, err := h.service.StartPairing(c.Request().Context(), tenantID)
	if err != nil {
		if errors.Is(err, service.ErrWhatsAppIntegrationInvalid) {
			return appresponse.ValidationError(c, err.Error())
		}
		if errors.Is(err, service.ErrWhatsAppDeviceConflict) {
			return appresponse.Conflict(c, "WhatsApp device sudah digunakan tenant lain")
		}
		if errors.Is(err, service.ErrWhatsAppAuthInvalid) {
			return appresponse.ValidationError(c, "WhatsApp belum dapat diautentikasi oleh platform")
		}
		return appresponse.ServiceUnavailable(c, "GOWA pairing is unavailable")
	}
	pairing.QRURL = "/api/v1/tenants/" + tenantID.String() + "/integrations/whatsapp/qr"
	return appresponse.Success(c, pairing)
}

// GetPairingQR serves the current GOWA QR image through the authenticated API.
func (h *WhatsAppIntegrationHandler) GetPairingQR(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}
	body, contentType, err := h.service.GetPairingQR(c.Request().Context(), tenantID)
	if err != nil {
		return appresponse.NotFound(c, "WhatsApp pairing QR")
	}
	return c.Blob(http.StatusOK, contentType, body)
}
