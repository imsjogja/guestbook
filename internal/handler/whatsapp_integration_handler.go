package handler

import (
	"errors"

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
		return appresponse.InternalError(c, "Failed to update WhatsApp integration")
	}
	return appresponse.Success(c, status)
}
