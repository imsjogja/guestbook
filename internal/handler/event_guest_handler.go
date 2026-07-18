package handler

import (
	"errors"
	"io"
	"strconv"

	"guestflow/internal/domain"
	"guestflow/internal/service"
	apperrors "guestflow/pkg/errors"
	"guestflow/pkg/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type EventGuestHandler struct {
	service *service.EventGuestService
}

func NewEventGuestHandler(eventGuestService *service.EventGuestService) *EventGuestHandler {
	return &EventGuestHandler{service: eventGuestService}
}

func (h *EventGuestHandler) Create(c echo.Context) error {
	tenantID, eventID, err := parseEventScope(c)
	if err != nil {
		return response.Error(c, apperrors.BadRequest(err.Error()))
	}
	var req domain.EventGuestCreateRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, apperrors.BadRequest("invalid request body"))
	}
	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return response.Error(c, apperrors.Unauthorized("unauthenticated"))
	}
	item, err := h.service.Create(c.Request().Context(), tenantID, eventID, userID, req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			return response.Error(c, apperrors.BadRequest("invalid event guest source"))
		case errors.Is(err, domain.ErrAlreadyExists):
			return response.Error(c, apperrors.Conflict("guest is already in this event"))
		case errors.Is(err, domain.ErrEventNotFound), errors.Is(err, domain.ErrNotFound):
			return response.Error(c, apperrors.NotFound("event or guest"))
		default:
			return response.Error(c, apperrors.WrapInternal(err, "failed to add guest to event"))
		}
	}
	return response.Created(c, item)
}

func (h *EventGuestHandler) List(c echo.Context) error {
	tenantID, eventID, err := parseEventScope(c)
	if err != nil {
		return response.Error(c, apperrors.BadRequest(err.Error()))
	}
	params := domain.EventGuestListParams{TenantID: tenantID, EventID: eventID, Search: c.QueryParam("search"), Status: c.QueryParam("status"), Page: 1, PerPage: 20}
	if value, parseErr := strconv.Atoi(c.QueryParam("page")); parseErr == nil && value > 0 {
		params.Page = value
	}
	if value, parseErr := strconv.Atoi(c.QueryParam("per_page")); parseErr == nil && value > 0 {
		params.PerPage = value
	}
	items, total, err := h.service.List(c.Request().Context(), params)
	if err != nil {
		return response.Error(c, apperrors.WrapInternal(err, "failed to list event guests"))
	}
	return response.Paginated(c, items, response.Meta{CurrentPage: params.Page, PerPage: params.PerPage, Total: total, TotalPages: (total + params.PerPage - 1) / params.PerPage})
}

func (h *EventGuestHandler) ImportCSV(c echo.Context) error {
	tenantID, eventID, err := parseEventScope(c)
	if err != nil {
		return response.Error(c, apperrors.BadRequest(err.Error()))
	}
	file, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, apperrors.BadRequest("CSV file is required (form field: 'file')"))
	}
	src, err := file.Open()
	if err != nil {
		return response.Error(c, apperrors.Internal("failed to read uploaded file"))
	}
	defer src.Close()
	content, err := io.ReadAll(src)
	if err != nil {
		return response.Error(c, apperrors.Internal("failed to read file content"))
	}
	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return response.Error(c, apperrors.Unauthorized("unauthenticated"))
	}
	result, err := h.service.ImportCSV(c.Request().Context(), tenantID, eventID, userID, content)
	if err != nil {
		return response.Error(c, apperrors.WrapInternal(err, "failed to import event guests"))
	}
	return response.Success(c, result)
}

func (h *EventGuestHandler) Cancel(c echo.Context) error {
	tenantID, eventID, err := parseEventScope(c)
	if err != nil {
		return response.Error(c, apperrors.BadRequest(err.Error()))
	}
	eventGuestID, err := uuid.Parse(c.Param("eventGuestId"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid event guest id"))
	}
	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return response.Error(c, apperrors.Unauthorized("unauthenticated"))
	}
	if err := h.service.Cancel(c.Request().Context(), tenantID, eventID, eventGuestID, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.Error(c, apperrors.NotFound("event guest"))
		}
		return response.Error(c, apperrors.WrapInternal(err, "failed to remove guest from event"))
	}
	return response.NoContent(c)
}

func parseEventScope(c echo.Context) (uuid.UUID, uuid.UUID, error) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, errors.New("invalid tenant id")
	}
	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, errors.New("invalid event id")
	}
	return tenantID, eventID, nil
}
