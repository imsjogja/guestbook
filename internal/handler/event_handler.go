package handler

import (
	stderrors "errors"
	"strconv"
	"strings"
	"time"

	"guestflow/internal/domain"
	"guestflow/internal/service"
	apperrors "guestflow/pkg/errors"
	"guestflow/pkg/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// EventHandler handles HTTP requests for event operations.
type EventHandler struct {
	eventService       *service.EventService
	eventAccessService *service.EventAccessService
	publicURL          string
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(eventService *service.EventService, eventAccessService *service.EventAccessService, publicURL string) *EventHandler {
	return &EventHandler{
		eventService:       eventService,
		eventAccessService: eventAccessService,
		publicURL:          strings.TrimRight(publicURL, "/"),
	}
}

// Create handles POST /api/v1/tenants/:id/events - creates a new event.
func (h *EventHandler) Create(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid tenant id"))
	}

	var req domain.EventCreateRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, apperrors.BadRequest("invalid request body"))
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return response.Error(c, apperrors.Unauthorized("unauthenticated"))
	}

	event, err := h.eventService.Create(c.Request().Context(), tenantID, userID, req)
	if err != nil {
		if stderrors.Is(err, domain.ErrInvalidInput) {
			return response.Error(c, apperrors.BadRequest(err.Error()))
		}
		return response.Error(c, apperrors.WrapInternal(err, "failed to create event"))
	}

	return response.Created(c, event)
}

// List handles GET /api/v1/tenants/:id/events - lists events for a tenant.
func (h *EventHandler) List(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid tenant id"))
	}

	filter := parseEventFilter(c)
	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return response.Error(c, apperrors.Unauthorized("unauthenticated"))
	}

	events, total, err := h.eventAccessService.ListAccessibleEvents(c.Request().Context(), tenantID, userID, filter)
	if err != nil {
		return response.Error(c, apperrors.WrapInternal(err, "failed to list events"))
	}

	meta := response.Meta{
		CurrentPage: filter.Page,
		PerPage:     filter.PerPage,
		Total:       total,
		TotalPages:  (total + filter.PerPage - 1) / filter.PerPage,
	}

	return response.Paginated(c, events, meta)
}

// Get handles GET /api/v1/tenants/:id/events/:eventId - retrieves an event.
func (h *EventHandler) Get(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid tenant id"))
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid event id"))
	}

	event, err := h.eventService.Get(c.Request().Context(), tenantID, eventID)
	if err != nil {
		if stderrors.Is(err, domain.ErrEventNotFound) {
			return response.Error(c, apperrors.NotFound("event"))
		}
		return response.Error(c, apperrors.WrapInternal(err, "failed to retrieve event"))
	}

	return response.Success(c, event)
}

// GetSelfCheckinQR returns the unique public QR URL for an event display.
func (h *EventHandler) GetSelfCheckinQR(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid tenant id"))
	}
	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid event id"))
	}
	event, err := h.eventService.Get(c.Request().Context(), tenantID, eventID)
	if err != nil {
		if stderrors.Is(err, domain.ErrEventNotFound) {
			return response.Error(c, apperrors.NotFound("event"))
		}
		return response.Error(c, apperrors.WrapInternal(err, "failed to retrieve event"))
	}
	if event.SelfCheckinToken == "" {
		return response.Error(c, apperrors.WrapInternal(domain.ErrInvalidInput, "event self check-in token is unavailable"))
	}
	baseURL := h.publicURL
	if baseURL == "" {
		baseURL = "https://guestflow.id"
	}
	return response.Success(c, domain.EventSelfCheckinQR{
		EventID: event.ID,
		URL:     baseURL + "/checkin/event/" + event.SelfCheckinToken,
	})
}

// Update handles PATCH /api/v1/tenants/:id/events/:eventId - updates an event.
func (h *EventHandler) Update(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid tenant id"))
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid event id"))
	}

	var req domain.EventUpdateRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, apperrors.BadRequest("invalid request body"))
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return response.Error(c, apperrors.Unauthorized("unauthenticated"))
	}

	event, err := h.eventService.Update(c.Request().Context(), tenantID, userID, eventID, req)
	if err != nil {
		if stderrors.Is(err, domain.ErrEventNotFound) {
			return response.Error(c, apperrors.NotFound("event"))
		}
		if stderrors.Is(err, domain.ErrEventCannotModify) {
			return response.Error(c, apperrors.BadRequest("event cannot be modified"))
		}
		if stderrors.Is(err, domain.ErrInvalidInput) {
			return response.Error(c, apperrors.BadRequest(err.Error()))
		}
		return response.Error(c, apperrors.WrapInternal(err, "failed to update event"))
	}

	return response.Success(c, event)
}

// Delete handles DELETE /api/v1/tenants/:id/events/:eventId - soft deletes an event.
func (h *EventHandler) Delete(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid tenant id"))
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid event id"))
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return response.Error(c, apperrors.Unauthorized("unauthenticated"))
	}

	if err := h.eventService.SoftDelete(c.Request().Context(), tenantID, userID, eventID); err != nil {
		if stderrors.Is(err, domain.ErrEventNotFound) {
			return response.Error(c, apperrors.NotFound("event"))
		}
		if stderrors.Is(err, domain.ErrEventCannotDelete) {
			return response.Error(c, apperrors.BadRequest("ongoing events cannot be deleted"))
		}
		return response.Error(c, apperrors.WrapInternal(err, "failed to delete event"))
	}

	return response.NoContent(c)
}

// Publish handles POST /api/v1/tenants/:id/events/:eventId/publish - publishes an event.
func (h *EventHandler) Publish(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid tenant id"))
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return response.Error(c, apperrors.BadRequest("invalid event id"))
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return response.Error(c, apperrors.Unauthorized("unauthenticated"))
	}

	event, err := h.eventService.Publish(c.Request().Context(), tenantID, userID, eventID)
	if err != nil {
		if stderrors.Is(err, domain.ErrEventNotFound) {
			return response.Error(c, apperrors.NotFound("event"))
		}
		if stderrors.Is(err, domain.ErrEventInvalidStatusTransition) {
			return response.Error(c, apperrors.BadRequest("only draft events can be published"))
		}
		if stderrors.Is(err, domain.ErrInvalidInput) {
			return response.Error(c, apperrors.BadRequest(err.Error()))
		}
		return response.Error(c, apperrors.WrapInternal(err, "failed to publish event"))
	}

	return response.Success(c, event)
}

// parseEventFilter extracts filter parameters from the query string.
func parseEventFilter(c echo.Context) domain.EventFilter {
	filter := domain.EventFilter{
		Status:  c.QueryParam("status"),
		Type:    c.QueryParam("type"),
		Page:    1,
		PerPage: 20,
	}

	if pageStr := c.QueryParam("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filter.Page = page
		}
	}

	if perPageStr := c.QueryParam("per_page"); perPageStr != "" {
		if perPage, err := strconv.Atoi(perPageStr); err == nil && perPage > 0 {
			filter.PerPage = perPage
		}
	}

	if startFromStr := c.QueryParam("start_from"); startFromStr != "" {
		if t, err := time.Parse(time.RFC3339, startFromStr); err == nil {
			filter.StartFrom = &t
		}
	}

	if startToStr := c.QueryParam("start_to"); startToStr != "" {
		if t, err := time.Parse(time.RFC3339, startToStr); err == nil {
			filter.StartTo = &t
		}
	}

	return filter
}
