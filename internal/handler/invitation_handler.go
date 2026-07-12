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

// InvitationHandler handles HTTP requests for invitation operations.
type InvitationHandler struct {
	invitationService *service.InvitationService
}

// NewInvitationHandler creates a new InvitationHandler.
func NewInvitationHandler(invitationService *service.InvitationService) *InvitationHandler {
	return &InvitationHandler{
		invitationService: invitationService,
	}
}

// Create handles POST /api/v1/tenants/:id/events/:eventId/invitations - creates invitations for guests.
func (h *InvitationHandler) Create(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	var req domain.InvitationCreateRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	if len(req.GuestIDs) == 0 {
		return appresponse.ValidationError(c, "guest_ids is required with at least one guest")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	// Create invitation for the first guest in the list (single create).
	invitation, err := h.invitationService.Create(c.Request().Context(), tenantID, eventID, userID, req, req.GuestIDs[0])
	if err != nil {
		switch {
		case stderrors.Is(err, domain.ErrNotFound):
			return appresponse.NotFound(c, "Event or guest")
		case stderrors.Is(err, domain.ErrAlreadyExists):
			return appresponse.Conflict(c, "Invitation already exists for this guest")
		case stderrors.Is(err, domain.ErrInvalidInput):
			return appresponse.ValidationError(c, err.Error())
		default:
			return appresponse.InternalError(c, "Failed to create invitation")
		}
	}

	return appresponse.Created(c, invitation)
}

// List handles GET /api/v1/tenants/:id/events/:eventId/invitations - lists invitations for an event.
func (h *InvitationHandler) List(c echo.Context) error {
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

	invitations, total, err := h.invitationService.List(c.Request().Context(), tenantID, eventID, status, page, perPage)
	if err != nil {
		return appresponse.InternalError(c, "Failed to list invitations")
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	return appresponse.Paginated(c, invitations, appresponse.Meta{
		CurrentPage: page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
	})
}

// Get handles GET /api/v1/tenants/:id/events/:eventId/invitations/:invitationId - gets an invitation.
func (h *InvitationHandler) Get(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	invitationID, err := uuid.Parse(c.Param("invitationId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid invitation ID")
	}

	invitation, err := h.invitationService.Get(c.Request().Context(), tenantID, invitationID)
	if err != nil {
		if stderrors.Is(err, domain.ErrInvitationNotFound) {
			return appresponse.NotFound(c, "Invitation")
		}
		return appresponse.InternalError(c, "Failed to retrieve invitation")
	}

	return appresponse.Success(c, invitation)
}

// Delete handles DELETE /api/v1/tenants/:id/events/:eventId/invitations/:invitationId - revokes/deletes an invitation.
func (h *InvitationHandler) Delete(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	invitationID, err := uuid.Parse(c.Param("invitationId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid invitation ID")
	}

	// Check for revoke reason in query param.
	reason := c.QueryParam("reason")

	if reason != "" {
		// Revoke with reason.
		userID, err := getUserIDFromEchoContext(c)
		if err != nil {
			return appresponse.Unauthorized(c, "Authentication required")
		}

		if err := h.invitationService.Revoke(c.Request().Context(), tenantID, invitationID, userID, reason); err != nil {
			if stderrors.Is(err, domain.ErrInvitationNotFound) {
				return appresponse.NotFound(c, "Invitation")
			}
			if stderrors.Is(err, domain.ErrInvitationRevoked) {
				return appresponse.Conflict(c, "Invitation is already revoked")
			}
			return appresponse.InternalError(c, "Failed to revoke invitation")
		}
	} else {
		// Soft delete.
		if err := h.invitationService.SoftDelete(c.Request().Context(), tenantID, invitationID); err != nil {
			if stderrors.Is(err, domain.ErrInvitationNotFound) {
				return appresponse.NotFound(c, "Invitation")
			}
			return appresponse.InternalError(c, "Failed to delete invitation")
		}
	}

	return appresponse.NoContent(c)
}

// GetQRData handles GET /api/v1/tenants/:id/events/:eventId/invitations/:invitationId/qr - gets QR code data.
func (h *InvitationHandler) GetQRData(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	invitationID, err := uuid.Parse(c.Param("invitationId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid invitation ID")
	}

	qrData, err := h.invitationService.GetQRData(c.Request().Context(), tenantID, invitationID)
	if err != nil {
		if stderrors.Is(err, domain.ErrInvitationNotFound) {
			return appresponse.NotFound(c, "Invitation")
		}
		return appresponse.InternalError(c, "Failed to get QR data")
	}

	return appresponse.Success(c, qrData)
}

// BatchCreate handles POST /api/v1/tenants/:id/events/:eventId/invitations/batch - batch create invitations.
func (h *InvitationHandler) BatchCreate(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	var req domain.InvitationCreateRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	if len(req.GuestIDs) == 0 {
		return appresponse.ValidationError(c, "guest_ids is required with at least one guest")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	invitations, err := h.invitationService.GenerateBatch(c.Request().Context(), tenantID, eventID, userID, req)
	if err != nil {
		switch {
		case stderrors.Is(err, domain.ErrNotFound):
			return appresponse.NotFound(c, "Event")
		case stderrors.Is(err, domain.ErrInvalidInput):
			return appresponse.ValidationError(c, err.Error())
		default:
			return appresponse.InternalError(c, "Failed to create invitations")
		}
	}

	return appresponse.Created(c, invitations)
}
