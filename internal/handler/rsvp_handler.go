package handler

import (
	stderrors "errors"
	"strconv"

	"guestflow/internal/domain"
	"guestflow/internal/service"
	appresponse "guestflow/pkg/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// RSVPHandler handles HTTP requests for RSVP operations.
type RSVPHandler struct {
	rsvpService       *service.RSVPService
	invitationService *service.InvitationService
}

// NewRSVPHandler creates a new RSVPHandler.
func NewRSVPHandler(rsvpService *service.RSVPService, invitationService *service.InvitationService) *RSVPHandler {
	return &RSVPHandler{
		rsvpService:       rsvpService,
		invitationService: invitationService,
	}
}

// Submit handles POST /api/v1/rsvp - public RSVP submission (no auth, by token).
func (h *RSVPHandler) Submit(c echo.Context) error {
	var req domain.RSVPSubmitRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	if req.Token == "" {
		return appresponse.ValidationError(c, "token is required")
	}

	if req.Status == "" {
		return appresponse.ValidationError(c, "status is required")
	}

	// Get client IP address.
	ipAddress := c.RealIP()

	rsvp, err := h.rsvpService.Submit(c.Request().Context(), req, ipAddress)
	if err != nil {
		switch {
		case stderrors.Is(err, domain.ErrTokenInvalid):
			return appresponse.Unauthorized(c, "Invalid or expired invitation token")
		case stderrors.Is(err, domain.ErrInvitationRevoked):
			return appresponse.Forbidden(c, "Invitation has been revoked")
		case stderrors.Is(err, domain.ErrInvitationExpired):
			return appresponse.Forbidden(c, "Invitation has expired")
		case stderrors.Is(err, domain.ErrRSVPDeadlinePassed):
			return appresponse.Forbidden(c, "RSVP deadline has passed")
		case stderrors.Is(err, domain.ErrEventAtCapacity):
			return appresponse.Conflict(c, "Event has reached capacity")
		case stderrors.Is(err, domain.ErrInvalidInput):
			return appresponse.ValidationError(c, err.Error())
		case stderrors.Is(err, domain.ErrInvalidRSVPStatus):
			return appresponse.ValidationError(c, "Invalid RSVP status")
		default:
			return appresponse.InternalError(c, "Failed to submit RSVP")
		}
	}

	return appresponse.Success(c, rsvp)
}

// List handles GET /api/v1/tenants/:id/events/:eventId/rsvp - RSVP list (protected).
func (h *RSVPHandler) List(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	status := c.QueryParam("status")

	rsvps, total, err := h.rsvpService.GetByEvent(c.Request().Context(), tenantID, eventID, status, page, perPage)
	if err != nil {
		return appresponse.InternalError(c, "Failed to list RSVPs")
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	return appresponse.Paginated(c, rsvps, appresponse.Meta{
		CurrentPage: page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
	})
}

// Dashboard handles GET /api/v1/tenants/:id/events/:eventId/rsvp/dashboard - dashboard stats.
func (h *RSVPHandler) Dashboard(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	dashboard, err := h.rsvpService.GetDashboard(c.Request().Context(), tenantID, eventID)
	if err != nil {
		if stderrors.Is(err, domain.ErrEventNotFound) {
			return appresponse.NotFound(c, "Event")
		}
		return appresponse.InternalError(c, "Failed to get dashboard stats")
	}

	return appresponse.Success(c, dashboard)
}

// UpdateByOfficer handles PATCH /api/v1/tenants/:id/events/:eventId/rsvp/:rsvpId - officer manual update.
func (h *RSVPHandler) UpdateByOfficer(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	rsvpID, err := uuid.Parse(c.Param("rsvpId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid RSVP ID")
	}

	var req domain.RSVPUpdateRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	if req.Status == "" {
		return appresponse.ValidationError(c, "status is required")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	rsvp, err := h.rsvpService.UpdateByOfficer(c.Request().Context(), tenantID, eventID, rsvpID, userID, req)
	if err != nil {
		switch {
		case stderrors.Is(err, domain.ErrRSVPNotFound):
			return appresponse.NotFound(c, "RSVP")
		case stderrors.Is(err, domain.ErrEventNotFound):
			return appresponse.NotFound(c, "Event")
		case stderrors.Is(err, domain.ErrEventAtCapacity):
			return appresponse.Conflict(c, "Event has reached capacity")
		case stderrors.Is(err, domain.ErrInvalidRSVPStatus):
			return appresponse.ValidationError(c, "Invalid RSVP status")
		default:
			return appresponse.InternalError(c, "Failed to update RSVP")
		}
	}

	return appresponse.Success(c, rsvp)
}

// UpsertByGuest handles POST /api/v1/tenants/:id/events/:eventId/rsvp/by-guest/:guestId - creates or updates RSVP for a guest.
func (h *RSVPHandler) UpsertByGuest(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	guestID, err := uuid.Parse(c.Param("guestId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid guest ID")
	}

	var req domain.RSVPUpdateRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	if req.Status == "" {
		return appresponse.ValidationError(c, "status is required")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	rsvp, err := h.rsvpService.UpsertByOfficer(c.Request().Context(), tenantID, eventID, guestID, userID, req)
	if err != nil {
		switch {
		case stderrors.Is(err, domain.ErrInvitationNotFound):
			return appresponse.NotFound(c, "Invitation")
		case stderrors.Is(err, domain.ErrEventNotFound):
			return appresponse.NotFound(c, "Event")
		case stderrors.Is(err, domain.ErrEventAtCapacity):
			return appresponse.Conflict(c, "Event has reached capacity")
		case stderrors.Is(err, domain.ErrInvalidRSVPStatus):
			return appresponse.ValidationError(c, "Invalid RSVP status")
		default:
			return appresponse.InternalError(c, "Failed to save RSVP")
		}
	}

	return appresponse.Success(c, rsvp)
}
