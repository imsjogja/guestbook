package handler

import (
	"errors"
	"strings"

	"guestflow/internal/domain"
	"guestflow/internal/service"
	appresponse "guestflow/pkg/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// SeatingHandler handles HTTP requests for seating management operations.
type SeatingHandler struct {
	seatingService *service.SeatingService
}

// NewSeatingHandler creates a new SeatingHandler.
func NewSeatingHandler(seatingService *service.SeatingService) *SeatingHandler {
	return &SeatingHandler{
		seatingService: seatingService,
	}
}

// ─── Table Handlers ───────────────────────────────────────────────────────────

// CreateTable handles POST /api/v1/tenants/:id/events/:eventId/tables - Create table.
func (h *SeatingHandler) CreateTable(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	var req domain.TableCreateRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	if req.Name == "" {
		return appresponse.BadRequest(c, "Name is required")
	}
	if req.Capacity < 1 {
		return appresponse.BadRequest(c, "Capacity must be at least 1")
	}

	table, err := h.seatingService.CreateTable(c.Request().Context(), tenantID, eventID, req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return appresponse.BadRequest(c, err.Error())
		}
		return appresponse.InternalError(c, "Failed to create table")
	}

	return appresponse.Created(c, table)
}

// ListTables handles GET /api/v1/tenants/:id/events/:eventId/tables - List tables.
func (h *SeatingHandler) ListTables(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	tables, err := h.seatingService.ListTables(c.Request().Context(), tenantID, eventID)
	if err != nil {
		return appresponse.InternalError(c, "Failed to list tables")
	}

	return appresponse.Success(c, tables)
}

// GetTable handles GET /api/v1/tenants/:id/events/:eventId/tables/:tableId - Get table detail.
func (h *SeatingHandler) GetTable(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	tableID, err := uuid.Parse(c.Param("tableId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid table ID")
	}

	table, err := h.seatingService.GetTable(c.Request().Context(), tenantID, eventID, tableID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return appresponse.NotFound(c, "Table")
		}
		return appresponse.InternalError(c, "Failed to retrieve table")
	}

	return appresponse.Success(c, table)
}

// UpdateTable handles PATCH /api/v1/tenants/:id/events/:eventId/tables/:tableId - Update table.
func (h *SeatingHandler) UpdateTable(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	tableID, err := uuid.Parse(c.Param("tableId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid table ID")
	}

	var req domain.TableUpdateRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	table, err := h.seatingService.UpdateTable(c.Request().Context(), tenantID, eventID, tableID, req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			return appresponse.NotFound(c, "Table")
		case errors.Is(err, domain.ErrInvalidInput):
			return appresponse.BadRequest(c, err.Error())
		case errors.Is(err, domain.ErrForbidden):
			return appresponse.Forbidden(c, err.Error())
		default:
			return appresponse.InternalError(c, "Failed to update table")
		}
	}

	return appresponse.Success(c, table)
}

// DeleteTable handles DELETE /api/v1/tenants/:id/events/:eventId/tables/:tableId - Delete table.
func (h *SeatingHandler) DeleteTable(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	tableID, err := uuid.Parse(c.Param("tableId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid table ID")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	if err := h.seatingService.DeleteTable(c.Request().Context(), tenantID, eventID, tableID, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return appresponse.NotFound(c, "Table")
		}
		return appresponse.InternalError(c, "Failed to delete table")
	}

	return appresponse.NoContent(c)
}

// ─── Seat Assignment Handlers ─────────────────────────────────────────────────

// AssignGuest handles POST /api/v1/tenants/:id/events/:eventId/tables/:tableId/assign - Assign guest.
func (h *SeatingHandler) AssignGuest(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	tableID, err := uuid.Parse(c.Param("tableId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid table ID")
	}

	var req domain.AssignGuestRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	if req.GuestID == uuid.Nil {
		return appresponse.BadRequest(c, "guest_id is required")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	if err := h.seatingService.AssignGuest(c.Request().Context(), tenantID, eventID, tableID, req, userID); err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			return appresponse.NotFound(c, "Table or guest")
		case errors.Is(err, domain.ErrInvalidInput):
			return appresponse.BadRequest(c, err.Error())
		case errors.Is(err, domain.ErrForbidden):
			return appresponse.Forbidden(c, err.Error())
		default:
			return appresponse.InternalError(c, "Failed to assign guest")
		}
	}

	return appresponse.Success(c, map[string]string{"message": "Guest assigned successfully"})
}

// UnassignGuest handles DELETE /api/v1/tenants/:id/events/:eventId/tables/:tableId/assign/:guestId - Unassign guest.
func (h *SeatingHandler) UnassignGuest(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	tableID, err := uuid.Parse(c.Param("tableId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid table ID")
	}

	guestID, err := uuid.Parse(c.Param("guestId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid guest ID")
	}

	if err := h.seatingService.UnassignGuest(c.Request().Context(), tenantID, eventID, tableID, guestID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return appresponse.NotFound(c, "Assignment")
		}
		return appresponse.InternalError(c, "Failed to unassign guest")
	}

	return appresponse.NoContent(c)
}

// ─── Layout Handler ───────────────────────────────────────────────────────────

// GetLayout handles GET /api/v1/tenants/:id/events/:eventId/seating/layout - Full layout.
func (h *SeatingHandler) GetLayout(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	layout, err := h.seatingService.GetLayout(c.Request().Context(), tenantID, eventID)
	if err != nil {
		return appresponse.InternalError(c, "Failed to retrieve seating layout")
	}

	return appresponse.Success(c, layout)
}

// ─── Zone Handlers ────────────────────────────────────────────────────────────

// CreateZone handles POST /api/v1/tenants/:id/events/:eventId/zones - Create zone.
func (h *SeatingHandler) CreateZone(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		SortOrder   int    `json:"sort_order,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return appresponse.BadRequest(c, "Name is required")
	}

	zoneReq := struct {
		EventID     uuid.UUID `json:"event_id" validate:"required"`
		Name        string    `json:"name" validate:"required,min=1,max=100"`
		Description string    `json:"description,omitempty"`
		SortOrder   int       `json:"sort_order,omitempty"`
	}{
		EventID:     eventID,
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
	}

	zone, err := h.seatingService.CreateZone(c.Request().Context(), tenantID, zoneReq)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return appresponse.BadRequest(c, err.Error())
		}
		return appresponse.InternalError(c, "Failed to create zone")
	}

	return appresponse.Created(c, zone)
}

// ListZones handles GET /api/v1/tenants/:id/events/:eventId/zones - List zones.
func (h *SeatingHandler) ListZones(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	zones, err := h.seatingService.ListZones(c.Request().Context(), tenantID, eventID)
	if err != nil {
		return appresponse.InternalError(c, "Failed to list zones")
	}

	return appresponse.Success(c, zones)
}

// AutoAssign handles POST /api/v1/tenants/:id/events/:eventId/seating/auto-assign - Auto assign guests.
func (h *SeatingHandler) AutoAssign(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	result, err := h.seatingService.AutoAssign(c.Request().Context(), tenantID, eventID, userID)
	if err != nil {
		return appresponse.InternalError(c, "Failed to auto-assign guests")
	}

	return appresponse.Success(c, result)
}
