package handler

import (
	"errors"

	"guestflow/internal/domain"
	"guestflow/internal/service"
	apperrors "guestflow/pkg/errors"
	"guestflow/pkg/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type GuestGiftHandler struct {
	service *service.GuestGiftService
}

func NewGuestGiftHandler(guestGiftService *service.GuestGiftService) *GuestGiftHandler {
	return &GuestGiftHandler{service: guestGiftService}
}

func (h *GuestGiftHandler) List(c echo.Context) error {
	tenantID, eventID, err := parseEventScope(c)
	if err != nil {
		return response.Error(c, apperrors.BadRequest(err.Error()))
	}
	items, err := h.service.List(c.Request().Context(), tenantID, eventID)
	if err != nil {
		return response.Error(c, apperrors.WrapInternal(err, "failed to list guest gifts"))
	}
	return response.Success(c, items)
}

func (h *GuestGiftHandler) Upsert(c echo.Context) error {
	tenantID, eventID, err := parseEventScope(c)
	if err != nil {
		return response.Error(c, apperrors.BadRequest(err.Error()))
	}
	guestID, err := uuid.Parse(c.Param("guestId"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid guest id"))
	}
	var req domain.GuestGiftUpsertRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, apperrors.BadRequest("invalid request body"))
	}
	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return response.Error(c, apperrors.Unauthorized("unauthenticated"))
	}
	item, err := h.service.Upsert(c.Request().Context(), tenantID, eventID, guestID, userID, req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			return response.Error(c, apperrors.BadRequest(err.Error()))
		case errors.Is(err, domain.ErrNotFound):
			return response.Error(c, apperrors.NotFound("event guest"))
		default:
			return response.Error(c, apperrors.WrapInternal(err, "failed to save guest gift"))
		}
	}
	return response.Success(c, item)
}

func (h *GuestGiftHandler) Delete(c echo.Context) error {
	tenantID, eventID, err := parseEventScope(c)
	if err != nil {
		return response.Error(c, apperrors.BadRequest(err.Error()))
	}
	guestID, err := uuid.Parse(c.Param("guestId"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid guest id"))
	}
	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return response.Error(c, apperrors.Unauthorized("unauthenticated"))
	}
	if err := h.service.Delete(c.Request().Context(), tenantID, eventID, guestID, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.Error(c, apperrors.NotFound("guest gift"))
		}
		return response.Error(c, apperrors.WrapInternal(err, "failed to delete guest gift"))
	}
	return response.NoContent(c)
}
