package handler

import (
	stderrors "errors"
	"strconv"
	"strings"

	"guestflow/internal/domain"
	"guestflow/internal/service"
	appresponse "guestflow/pkg/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// CommunicationHandler handles HTTP requests for communication operations.
type CommunicationHandler struct {
	commService *service.CommunicationService
}

// NewCommunicationHandler creates a new CommunicationHandler.
func NewCommunicationHandler(commService *service.CommunicationService) *CommunicationHandler {
	return &CommunicationHandler{
		commService: commService,
	}
}

// ---------------------------------------------------------------------------
// Templates
// ---------------------------------------------------------------------------

// CreateTemplate handles POST /api/v1/tenants/:id/templates - creates a new template.
func (h *CommunicationHandler) CreateTemplate(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	var req domain.CommunicationTemplateCreateRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	if req.Name == "" || req.Channel == "" || req.Body == "" {
		return appresponse.ValidationError(c, "name, channel, and body are required")
	}

	template, err := h.commService.CreateTemplate(c.Request().Context(), tenantID, req)
	if err != nil {
		if stderrors.Is(err, domain.ErrInvalidChannel) {
			return appresponse.ValidationError(c, err.Error())
		}
		return appresponse.InternalError(c, "Failed to create template")
	}

	return appresponse.Created(c, template)
}

// ListTemplates handles GET /api/v1/tenants/:id/templates - lists templates.
func (h *CommunicationHandler) ListTemplates(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
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

	channel := strings.TrimSpace(c.QueryParam("channel"))
	msgType := strings.TrimSpace(c.QueryParam("type"))

	var isActive *bool
	if activeParam := c.QueryParam("is_active"); activeParam != "" {
		a, err := strconv.ParseBool(activeParam)
		if err == nil {
			isActive = &a
		}
	}

	templates, total, err := h.commService.ListTemplates(c.Request().Context(), tenantID, channel, msgType, isActive, page, perPage)
	if err != nil {
		return appresponse.InternalError(c, "Failed to list templates")
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	return appresponse.Paginated(c, templates, appresponse.Meta{
		CurrentPage: page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
	})
}

// GetTemplate handles GET /api/v1/tenants/:id/templates/:templateId - gets a template.
func (h *CommunicationHandler) GetTemplate(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid template ID")
	}

	template, err := h.commService.GetTemplate(c.Request().Context(), tenantID, templateID)
	if err != nil {
		if stderrors.Is(err, domain.ErrTemplateNotFound) {
			return appresponse.NotFound(c, "Template")
		}
		return appresponse.InternalError(c, "Failed to retrieve template")
	}

	return appresponse.Success(c, template)
}

// UpdateTemplate handles PATCH /api/v1/tenants/:id/templates/:templateId - updates a template.
func (h *CommunicationHandler) UpdateTemplate(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid template ID")
	}

	var req domain.CommunicationTemplateUpdateRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	template, err := h.commService.UpdateTemplate(c.Request().Context(), tenantID, templateID, req)
	if err != nil {
		if stderrors.Is(err, domain.ErrTemplateNotFound) {
			return appresponse.NotFound(c, "Template")
		}
		return appresponse.InternalError(c, "Failed to update template")
	}

	return appresponse.Success(c, template)
}

// DeleteTemplate handles DELETE /api/v1/tenants/:id/templates/:templateId - deletes a template.
func (h *CommunicationHandler) DeleteTemplate(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid template ID")
	}

	if err := h.commService.DeleteTemplate(c.Request().Context(), tenantID, templateID); err != nil {
		if stderrors.Is(err, domain.ErrTemplateNotFound) {
			return appresponse.NotFound(c, "Template")
		}
		return appresponse.InternalError(c, "Failed to delete template")
	}

	return appresponse.NoContent(c)
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// SendMessage handles POST /api/v1/tenants/:id/events/:eventId/messages/send - sends manual message.
func (h *CommunicationHandler) SendMessage(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	var req domain.SendMessageRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	if len(req.GuestIDs) == 0 {
		return appresponse.ValidationError(c, "guest_ids is required with at least one guest")
	}

	if req.TemplateID == uuid.Nil {
		return appresponse.ValidationError(c, "template_id is required")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	messages, err := h.commService.SendMessage(c.Request().Context(), tenantID, eventID, userID, req)
	if err != nil {
		switch {
		case stderrors.Is(err, domain.ErrTemplateNotFound):
			return appresponse.NotFound(c, "Template")
		case stderrors.Is(err, domain.ErrTemplateInactive):
			return appresponse.Conflict(c, "Template is inactive")
		case stderrors.Is(err, domain.ErrNotFound):
			return appresponse.NotFound(c, "Guest or event")
		default:
			return appresponse.InternalError(c, "Failed to send messages")
		}
	}

	return appresponse.Created(c, messages)
}

// ListMessages handles GET /api/v1/tenants/:id/events/:eventId/messages - lists messages.
func (h *CommunicationHandler) ListMessages(c echo.Context) error {
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

	status := strings.TrimSpace(c.QueryParam("status"))

	var campaignID, guestID *uuid.UUID
	if cid := c.QueryParam("campaign_id"); cid != "" {
		if id, err := uuid.Parse(cid); err == nil {
			campaignID = &id
		}
	}
	if gid := c.QueryParam("guest_id"); gid != "" {
		if id, err := uuid.Parse(gid); err == nil {
			guestID = &id
		}
	}

	messages, total, err := h.commService.ListMessages(c.Request().Context(), tenantID, eventID, campaignID, guestID, status, page, perPage)
	if err != nil {
		return appresponse.InternalError(c, "Failed to list messages")
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	return appresponse.Paginated(c, messages, appresponse.Meta{
		CurrentPage: page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
	})
}

// ---------------------------------------------------------------------------
// Campaigns
// ---------------------------------------------------------------------------

// CreateCampaign handles POST /api/v1/tenants/:id/events/:eventId/campaigns - creates a campaign.
func (h *CommunicationHandler) CreateCampaign(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	var req domain.CommunicationCampaignCreateRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	if req.Name == "" || req.TemplateID == uuid.Nil || req.Channel == "" || req.Type == "" {
		return appresponse.ValidationError(c, "name, template_id, channel, and type are required")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	campaign, err := h.commService.CreateCampaign(c.Request().Context(), tenantID, eventID, userID, req)
	if err != nil {
		switch {
		case stderrors.Is(err, domain.ErrTemplateNotFound):
			return appresponse.NotFound(c, "Template")
		case stderrors.Is(err, domain.ErrTemplateInactive):
			return appresponse.Conflict(c, "Template is inactive")
		case stderrors.Is(err, domain.ErrInvalidChannel):
			return appresponse.ValidationError(c, err.Error())
		default:
			return appresponse.InternalError(c, "Failed to create campaign")
		}
	}

	return appresponse.Created(c, campaign)
}

// ListCampaigns handles GET /api/v1/tenants/:id/events/:eventId/campaigns - lists campaigns.
func (h *CommunicationHandler) ListCampaigns(c echo.Context) error {
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

	status := strings.TrimSpace(c.QueryParam("status"))

	campaigns, total, err := h.commService.ListCampaigns(c.Request().Context(), tenantID, eventID, status, page, perPage)
	if err != nil {
		return appresponse.InternalError(c, "Failed to list campaigns")
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	return appresponse.Paginated(c, campaigns, appresponse.Meta{
		CurrentPage: page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
	})
}

// LaunchCampaign handles POST /api/v1/tenants/:id/events/:eventId/campaigns/:campaignId/launch - launches a campaign.
func (h *CommunicationHandler) LaunchCampaign(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	campaignID, err := uuid.Parse(c.Param("campaignId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid campaign ID")
	}

	if err := h.commService.SendCampaign(c.Request().Context(), tenantID, eventID, campaignID); err != nil {
		switch {
		case stderrors.Is(err, domain.ErrCampaignNotFound):
			return appresponse.NotFound(c, "Campaign")
		case stderrors.Is(err, domain.ErrCampaignStarted):
			return appresponse.Conflict(c, "Campaign has already started or completed")
		case stderrors.Is(err, domain.ErrTemplateInactive):
			return appresponse.Conflict(c, "Template is inactive")
		case stderrors.Is(err, domain.ErrEmptyRecipientList):
			return appresponse.ValidationError(c, "No recipients match the filter criteria")
		default:
			return appresponse.InternalError(c, "Failed to launch campaign")
		}
	}

	return appresponse.Success(c, map[string]string{"status": "launched"})
}

// CancelCampaign handles POST /api/v1/tenants/:id/events/:eventId/campaigns/:campaignId/cancel - cancels a campaign.
func (h *CommunicationHandler) CancelCampaign(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	campaignID, err := uuid.Parse(c.Param("campaignId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid campaign ID")
	}

	if err := h.commService.CancelCampaign(c.Request().Context(), tenantID, campaignID); err != nil {
		switch {
		case stderrors.Is(err, domain.ErrCampaignNotFound):
			return appresponse.NotFound(c, "Campaign")
		case stderrors.Is(err, domain.ErrCampaignStarted):
			return appresponse.Conflict(c, "Cannot cancel a campaign that has already started")
		default:
			return appresponse.InternalError(c, "Failed to cancel campaign")
		}
	}

	return appresponse.Success(c, map[string]string{"status": "cancelled"})
}
