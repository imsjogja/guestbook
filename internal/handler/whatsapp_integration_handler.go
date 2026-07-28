package handler

import (
	"errors"
	"net/http"
	"strings"

	"guestflow/internal/service"
	"guestflow/internal/whatsapp"
	appresponse "guestflow/pkg/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type whatsappTestRequest struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

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

// Logout disconnects the tenant's currently paired WhatsApp session.
func (h *WhatsAppIntegrationHandler) Logout(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}
	status, err := h.service.Logout(c.Request().Context(), tenantID)
	if err != nil {
		if errors.Is(err, service.ErrWhatsAppIntegrationInvalid) {
			return appresponse.ValidationError(c, "WhatsApp belum aktif atau belum dikonfigurasi")
		}
		if errors.Is(err, service.ErrWhatsAppAuthInvalid) {
			return appresponse.ServiceUnavailable(c, "Koneksi WhatsApp belum dapat digunakan")
		}
		return appresponse.ServiceUnavailable(c, "Gagal logout perangkat WhatsApp")
	}
	return appresponse.Success(c, status)
}

// Test sends a tenant-scoped test message without requiring an event guest.
func (h *WhatsAppIntegrationHandler) Test(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}
	var req whatsappTestRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}
	if strings.TrimSpace(req.To) == "" {
		return appresponse.ValidationError(c, "Nomor tujuan wajib diisi")
	}
	if strings.TrimSpace(req.Message) == "" {
		return appresponse.ValidationError(c, "Pesan uji wajib diisi")
	}
	result, err := h.service.SendTest(c.Request().Context(), tenantID, req.To, req.Message)
	if err != nil {
		var providerErr *whatsapp.ProviderError
		if errors.As(err, &providerErr) {
			message := strings.ToLower(providerErr.Message)
			if strings.Contains(message, "not registered") || strings.Contains(message, "not on whatsapp") {
				return appresponse.ValidationError(c, "Nomor tujuan belum terdaftar di WhatsApp")
			}
			if providerErr.StatusCode == http.StatusUnauthorized {
				return appresponse.ServiceUnavailable(c, "Koneksi WhatsApp belum dapat digunakan")
			}
			return appresponse.ServiceUnavailable(c, "Pesan uji WhatsApp gagal dikirim")
		}
		switch {
		case errors.Is(err, service.ErrWhatsAppIntegrationInvalid):
			return appresponse.ValidationError(c, "Aktifkan dan lengkapi koneksi WhatsApp terlebih dahulu")
		case errors.Is(err, service.ErrWhatsAppNotReady), errors.Is(err, service.ErrWhatsAppAuthInvalid):
			return appresponse.ServiceUnavailable(c, "WhatsApp belum terhubung")
		case errors.Is(err, whatsapp.ErrPhoneMissing), errors.Is(err, whatsapp.ErrInvalidPhone):
			return appresponse.ValidationError(c, "Nomor tujuan WhatsApp tidak valid")
		case errors.Is(err, whatsapp.ErrNotConfigured):
			return appresponse.ServiceUnavailable(c, "WhatsApp belum dikonfigurasi")
		default:
			return appresponse.ServiceUnavailable(c, "Pesan uji WhatsApp gagal dikirim")
		}
	}
	return appresponse.Success(c, result)
}
