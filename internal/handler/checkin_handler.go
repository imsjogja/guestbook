package handler

import (
	"errors"
	"strconv"
	"strings"

	"guestflow/internal/domain"
	"guestflow/internal/service"
	appresponse "guestflow/pkg/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// CheckinHandler handles HTTP requests for check-in operations.
type CheckinHandler struct {
	checkinService *service.CheckinService
}

// NewCheckinHandler creates a new CheckinHandler.
func NewCheckinHandler(checkinService *service.CheckinService) *CheckinHandler {
	return &CheckinHandler{
		checkinService: checkinService,
	}
}

// Checkin handles POST /api/v1/tenants/:id/events/:eventId/checkin - Process check-in.
func (h *CheckinHandler) Checkin(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	var req domain.CheckinRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	// Validate method
	req.Method = strings.TrimSpace(req.Method)
	if req.Method == "" {
		return appresponse.BadRequest(c, "method is required")
	}

	// Get optional officer ID from authenticated user
	var officerID *uuid.UUID
	if userID, err := getUserIDFromEchoContext(c); err == nil && userID != uuid.Nil {
		officerID = &userID
	}

	result, err := h.checkinService.ProcessCheckin(c.Request().Context(), tenantID, eventID, officerID, req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAlreadyExists):
			return appresponse.Conflict(c, "Guest already checked in")
		case errors.Is(err, domain.ErrInvalidInput):
			return appresponse.BadRequest(c, err.Error())
		case errors.Is(err, domain.ErrNotFound):
			return appresponse.NotFound(c, "Guest or credential")
		default:
			return appresponse.InternalError(c, "Failed to process check-in")
		}
	}

	return appresponse.Success(c, result)
}

// GetStats handles GET /api/v1/tenants/:id/events/:eventId/checkin/stats - Real-time stats.
func (h *CheckinHandler) GetStats(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	stats, err := h.checkinService.GetStats(c.Request().Context(), tenantID, eventID)
	if err != nil {
		return appresponse.InternalError(c, "Failed to retrieve check-in stats")
	}

	return appresponse.Success(c, stats)
}

// SearchGuests handles GET /api/v1/tenants/:id/events/:eventId/checkin/search - Search guests for manual check-in.
func (h *CheckinHandler) SearchGuests(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	query := strings.TrimSpace(c.QueryParam("q"))
	if query == "" {
		return appresponse.BadRequest(c, "Search query 'q' is required")
	}

	// Default to masking sensitive data for security
	maskSensitive := true
	if maskParam := c.QueryParam("mask"); maskParam != "" {
		if parsed, err := strconv.ParseBool(maskParam); err == nil {
			maskSensitive = parsed
		}
	}

	results, err := h.checkinService.SearchGuests(c.Request().Context(), tenantID, eventID, query, maskSensitive)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return appresponse.BadRequest(c, err.Error())
		}
		return appresponse.InternalError(c, "Failed to search guests")
	}

	return appresponse.Success(c, results)
}

// GetRecent handles GET /api/v1/tenants/:id/events/:eventId/checkin/recent - Recent check-ins.
func (h *CheckinHandler) GetRecent(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	recent, err := h.checkinService.GetRecent(c.Request().Context(), tenantID, eventID, limit)
	if err != nil {
		return appresponse.InternalError(c, "Failed to retrieve recent check-ins")
	}

	return appresponse.Success(c, recent)
}

// Walkin handles POST /api/v1/tenants/:id/events/:eventId/checkin/walkin - Walk-in registration.
func (h *CheckinHandler) Walkin(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	var req domain.WalkinRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	// Validate required fields
	req.FullName = strings.TrimSpace(req.FullName)
	if req.FullName == "" {
		return appresponse.BadRequest(c, "full_name is required")
	}
	if req.GuestType == "" {
		return appresponse.BadRequest(c, "guest_type is required")
	}
	if req.ActualPax < 1 {
		return appresponse.BadRequest(c, "actual_pax must be at least 1")
	}

	// Get optional officer ID from authenticated user
	var officerID *uuid.UUID
	if userID, err := getUserIDFromEchoContext(c); err == nil && userID != uuid.Nil {
		officerID = &userID
	}

	result, err := h.checkinService.ProcessWalkin(c.Request().Context(), tenantID, eventID, officerID, req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAlreadyExists):
			return appresponse.Conflict(c, "Guest already exists")
		case errors.Is(err, domain.ErrInvalidInput):
			return appresponse.BadRequest(c, err.Error())
		default:
			return appresponse.InternalError(c, "Failed to process walk-in registration")
		}
	}

	return appresponse.Created(c, result)
}
