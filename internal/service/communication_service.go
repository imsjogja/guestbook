// Package service provides business logic for communication operations.
package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"guestflow/internal/domain"
	"guestflow/internal/repository"

	"github.com/google/uuid"
)

// CommunicationService handles business logic for templates, campaigns, and messages.
type CommunicationService struct {
	commRepo  *repository.CommunicationRepository
	guestRepo *repository.GuestRepository
	eventRepo *repository.EventRepository
}

// NewCommunicationService creates a new CommunicationService.
func NewCommunicationService(
	commRepo *repository.CommunicationRepository,
	guestRepo *repository.GuestRepository,
	eventRepo *repository.EventRepository,
) *CommunicationService {
	return &CommunicationService{
		commRepo:  commRepo,
		guestRepo: guestRepo,
		eventRepo: eventRepo,
	}
}

// ---------------------------------------------------------------------------
// Templates
// ---------------------------------------------------------------------------

// CreateTemplate creates a new communication template.
func (s *CommunicationService) CreateTemplate(ctx context.Context, tenantID uuid.UUID, req domain.CommunicationTemplateCreateRequest) (*domain.CommunicationTemplate, error) {
	// Validate channel
	if req.Channel != domain.ChannelWhatsApp && req.Channel != domain.ChannelEmail && req.Channel != domain.ChannelSMS {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidChannel, req.Channel)
	}

	lang := req.Language
	if lang == "" {
		lang = "id"
	}

	now := time.Now().UTC()
	template := &domain.CommunicationTemplate{
		TenantBase: domain.TenantBase{
			Base: domain.Base{
				ID:        uuid.New(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			TenantID: tenantID,
		},
		Name:      req.Name,
		Channel:   req.Channel,
		Type:      req.Type,
		Body:      req.Body,
		Variables: req.Variables,
		IsActive:  true,
		IsSystem:  false,
		Language:  lang,
	}

	if req.Subject != "" {
		template.Subject = &req.Subject
	}
	if req.Description != "" {
		template.Description = &req.Description
	}

	if err := s.commRepo.CreateTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}

	return template, nil
}

// GetTemplate retrieves a template by ID.
func (s *CommunicationService) GetTemplate(ctx context.Context, tenantID, templateID uuid.UUID) (*domain.CommunicationTemplate, error) {
	template, err := s.commRepo.GetTemplate(ctx, tenantID, templateID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}
	return template, nil
}

// UpdateTemplate updates an existing template.
func (s *CommunicationService) UpdateTemplate(ctx context.Context, tenantID, templateID uuid.UUID, req domain.CommunicationTemplateUpdateRequest) (*domain.CommunicationTemplate, error) {
	template, err := s.commRepo.GetTemplate(ctx, tenantID, templateID)
	if err != nil {
		return nil, fmt.Errorf("get template for update: %w", err)
	}

	// Apply updates
	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Channel != "" {
		template.Channel = req.Channel
	}
	if req.Type != "" {
		template.Type = req.Type
	}
	if req.Subject != "" {
		template.Subject = &req.Subject
	}
	if req.Body != "" {
		template.Body = req.Body
	}
	if req.Variables != nil {
		template.Variables = req.Variables
	}
	if req.Description != "" {
		template.Description = &req.Description
	}
	if req.Language != "" {
		template.Language = req.Language
	}
	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}

	template.Touch()

	if err := s.commRepo.UpdateTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("update template: %w", err)
	}

	return template, nil
}

// DeleteTemplate soft-deletes a template.
func (s *CommunicationService) DeleteTemplate(ctx context.Context, tenantID, templateID uuid.UUID) error {
	if err := s.commRepo.SoftDeleteTemplate(ctx, tenantID, templateID); err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	return nil
}

// ListTemplates lists templates for a tenant with filters.
func (s *CommunicationService) ListTemplates(ctx context.Context, tenantID uuid.UUID, channel, msgType string, isActive *bool, page, perPage int) ([]*domain.CommunicationTemplate, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	params := domain.TemplateListParams{
		TenantID: tenantID,
		Channel:  channel,
		Type:     msgType,
		IsActive: isActive,
		Page:     page,
		PerPage:  perPage,
	}

	templates, err := s.commRepo.ListTemplatesByTenant(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list templates: %w", err)
	}

	total, err := s.commRepo.CountTemplatesByTenant(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("count templates: %w", err)
	}

	return templates, total, nil
}

// GetTemplatesForEvent returns applicable templates for an event type and channel.
func (s *CommunicationService) GetTemplatesForEvent(ctx context.Context, tenantID uuid.UUID, channel, msgType string) ([]*domain.CommunicationTemplate, error) {
	params := domain.TemplateListParams{
		TenantID: tenantID,
		Channel:  channel,
		Type:     msgType,
		IsActive: boolPtr(true),
		Page:     1,
		PerPage:  100,
	}

	templates, err := s.commRepo.ListTemplatesByTenant(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("get templates for event: %w", err)
	}

	return templates, nil
}

// ---------------------------------------------------------------------------
// Template Rendering
// ---------------------------------------------------------------------------

// varPattern matches {{variable_name}} in templates
var varPattern = regexp.MustCompile(`\{\{(\w+)\}\}`)

// RenderTemplate replaces {{variables}} in the template body with actual values.
// Supported variables: guest_name, event_name, event_date, event_time, rsvp_link,
// guest_title, guest_type, table_name, seat_number, venue_name, venue_address.
func RenderTemplate(body string, variables map[string]string) string {
	return varPattern.ReplaceAllStringFunc(body, func(match string) string {
		// Extract variable name from {{name}}
		name := match[2 : len(match)-2]
		if val, ok := variables[name]; ok {
			return val
		}
		return match // keep original if not found
	})
}

// BuildRenderVariables constructs the variable map from guest, event, and invitation data.
func BuildRenderVariables(guest *domain.Guest, event *domain.Event, invitation *domain.Invitation, baseURL string) map[string]string {
	vars := make(map[string]string)

	// Guest variables
	vars["guest_name"] = guest.FullName
	if guest.Nickname != nil && *guest.Nickname != "" {
		vars["guest_name"] = *guest.Nickname
	}
	if guest.Title != nil {
		vars["guest_title"] = *guest.Title
	}
	vars["guest_type"] = guest.GuestType
	if guest.Phone != nil {
		vars["guest_phone"] = *guest.Phone
	}
	if guest.Email != nil {
		vars["guest_email"] = *guest.Email
	}

	// Event variables
	vars["event_name"] = event.Name
	vars["event_date"] = event.StartDate.Format("Monday, January 2, 2006")
	vars["event_time"] = event.StartDate.Format("15:04")
	if event.EndDate != nil {
		vars["event_end_time"] = event.EndDate.Format("15:04")
	}
	if event.DressCode != nil {
		vars["dress_code"] = *event.DressCode
	}

	// RSVP link
	if invitation != nil && invitation.Token != "" {
		vars["rsvp_link"] = fmt.Sprintf("%s/rsvp/%s", strings.TrimSuffix(baseURL, "/"), invitation.Token)
		vars["invitation_url"] = invitation.URL
	} else {
		vars["rsvp_link"] = baseURL
		vars["invitation_url"] = baseURL
	}

	return vars
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// SendMessage sends a manual message to specified guests using a template.
func (s *CommunicationService) SendMessage(ctx context.Context, tenantID, eventID, userID uuid.UUID, req domain.SendMessageRequest) ([]*domain.CommunicationMessage, error) {
	// Get template
	template, err := s.commRepo.GetTemplate(ctx, tenantID, req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	if !template.IsActive {
		return nil, domain.ErrTemplateInactive
	}

	// Get event
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	var messages []*domain.CommunicationMessage

	for _, guestID := range req.GuestIDs {
		// Get guest
		guest, err := s.guestRepo.GetByIDForTenant(ctx, tenantID, guestID)
		if err != nil {
			return nil, fmt.Errorf("get guest %s: %w", guestID, err)
		}

		// Build render variables
		vars := BuildRenderVariables(guest, event, nil, "")
		// Override with user-provided variables
		for k, v := range req.Variables {
			if sv, ok := v.(string); ok {
				vars[k] = sv
			}
		}

		// Render body
		renderedBody := RenderTemplate(template.Body, vars)

		// Create message record
		msg := &domain.CommunicationMessage{
			Base:     domain.NewBase(),
			TenantID: tenantID,
			EventID:  eventID,
			GuestID:  guestID,
			Channel:  template.Channel,
			Type:     template.Type,
			Body:     renderedBody,
			Status:   domain.MessageStatusQueued,
		}

		if template.Subject != nil {
			subject := RenderTemplate(*template.Subject, vars)
			msg.Subject = &subject
		}

		if err := s.commRepo.CreateMessage(ctx, msg); err != nil {
			return nil, fmt.Errorf("create message for guest %s: %w", guestID, err)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// ListMessages lists messages for an event with optional filters.
func (s *CommunicationService) ListMessages(ctx context.Context, tenantID, eventID uuid.UUID, campaignID, guestID *uuid.UUID, status string, page, perPage int) ([]*domain.CommunicationMessage, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	params := domain.MessageListParams{
		TenantID:   tenantID,
		EventID:    eventID,
		CampaignID: campaignID,
		GuestID:    guestID,
		Status:     status,
		Page:       page,
		PerPage:    perPage,
	}

	messages, err := s.commRepo.ListMessagesByCampaign(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list messages: %w", err)
	}

	total, err := s.commRepo.CountMessagesByCampaign(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("count messages: %w", err)
	}

	return messages, total, nil
}

// GetMessageStats returns aggregated delivery statistics for an event.
func (s *CommunicationService) GetMessageStats(ctx context.Context, tenantID, eventID uuid.UUID) (map[string]int, error) {
	counts, err := s.commRepo.CountByStatus(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get message stats: %w", err)
	}
	return counts, nil
}

// ---------------------------------------------------------------------------
// Campaigns
// ---------------------------------------------------------------------------

// CreateCampaign creates a new communication campaign.
func (s *CommunicationService) CreateCampaign(ctx context.Context, tenantID, eventID, userID uuid.UUID, req domain.CommunicationCampaignCreateRequest) (*domain.CommunicationCampaign, error) {
	// Validate template exists
	template, err := s.commRepo.GetTemplate(ctx, tenantID, req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}
	if !template.IsActive {
		return nil, domain.ErrTemplateInactive
	}

	// Validate channel
	if req.Channel != domain.ChannelWhatsApp && req.Channel != domain.ChannelEmail && req.Channel != domain.ChannelSMS {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidChannel, req.Channel)
	}

	now := time.Now().UTC()
	status := domain.CampaignStatusDraft
	if req.ScheduledAt != nil && req.ScheduledAt.After(now) {
		status = domain.CampaignStatusScheduled
	}

	campaign := &domain.CommunicationCampaign{
		TenantBase: domain.TenantBase{
			Base: domain.Base{
				ID:        uuid.New(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			TenantID: tenantID,
		},
		EventID:         eventID,
		Name:            req.Name,
		TemplateID:      req.TemplateID,
		Channel:         req.Channel,
		Type:            req.Type,
		Status:          status,
		RecipientFilter: req.RecipientFilter,
		ScheduledAt:     req.ScheduledAt,
		CreatedBy:       userID,
	}

	if err := s.commRepo.CreateCampaign(ctx, campaign); err != nil {
		return nil, fmt.Errorf("create campaign: %w", err)
	}

	return campaign, nil
}

// GetCampaign retrieves a campaign by ID.
func (s *CommunicationService) GetCampaign(ctx context.Context, tenantID, campaignID uuid.UUID) (*domain.CommunicationCampaign, error) {
	campaign, err := s.commRepo.GetCampaign(ctx, tenantID, campaignID)
	if err != nil {
		return nil, fmt.Errorf("get campaign: %w", err)
	}
	return campaign, nil
}

// ListCampaigns lists campaigns for an event.
func (s *CommunicationService) ListCampaigns(ctx context.Context, tenantID, eventID uuid.UUID, status string, page, perPage int) ([]*domain.CommunicationCampaign, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	params := domain.CampaignListParams{
		TenantID: tenantID,
		EventID:  eventID,
		Status:   status,
		Page:     page,
		PerPage:  perPage,
	}

	campaigns, err := s.commRepo.ListCampaignsByEvent(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list campaigns: %w", err)
	}

	total, err := s.commRepo.CountCampaignsByEvent(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("count campaigns: %w", err)
	}

	return campaigns, total, nil
}

// SendCampaign processes a campaign: resolves recipients, renders messages, queues for sending.
func (s *CommunicationService) SendCampaign(ctx context.Context, tenantID, eventID, campaignID uuid.UUID) error {
	// Get campaign
	campaign, err := s.commRepo.GetCampaign(ctx, tenantID, campaignID)
	if err != nil {
		return fmt.Errorf("get campaign: %w", err)
	}

	if campaign.Status == domain.CampaignStatusSending || campaign.Status == domain.CampaignStatusCompleted {
		return domain.ErrCampaignStarted
	}

	// Get template
	template, err := s.commRepo.GetTemplate(ctx, tenantID, campaign.TemplateID)
	if err != nil {
		return fmt.Errorf("get campaign template: %w", err)
	}
	if !template.IsActive {
		return domain.ErrTemplateInactive
	}

	// Get event
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("get event: %w", err)
	}

	// Build recipient filter from campaign
	var guestIDs []uuid.UUID
	if campaign.RecipientFilter != nil {
		// Parse recipient filter to determine target guests
		// For now, fetch all active guests for the event and apply filter
		params := domain.GuestListParams{
			TenantID: tenantID,
			Page:     1,
			PerPage:  10000,
		}

		// Apply simple filter if present
		if gf, ok := campaign.RecipientFilter["guest_type"]; ok {
			if gs, ok := gf.(string); ok {
				params.GuestType = gs
			}
		}
		if sf, ok := campaign.RecipientFilter["segment"]; ok {
			if ss, ok := sf.(string); ok {
				params.Segment = ss
			}
		}

		guests, err := s.guestRepo.ListByTenant(ctx, params)
		if err != nil {
			return fmt.Errorf("list recipients: %w", err)
		}

		for _, g := range guests {
			guestIDs = append(guestIDs, g.ID)
		}
	}

	if len(guestIDs) == 0 {
		return domain.ErrEmptyRecipientList
	}

	// Update campaign to sending status
	now := time.Now().UTC()
	if err := s.commRepo.UpdateCampaignStatus(ctx, tenantID, campaignID, domain.CampaignStatusSending, &now, nil, 0, 0); err != nil {
		return fmt.Errorf("update campaign status: %w", err)
	}

	// Create messages for each recipient
	sentCount := 0
	failedCount := 0

	for _, guestID := range guestIDs {
		guest, err := s.guestRepo.GetByIDForTenant(ctx, tenantID, guestID)
		if err != nil {
			failedCount++
			continue
		}

		// Build render variables
		vars := BuildRenderVariables(guest, event, nil, "")
		renderedBody := RenderTemplate(template.Body, vars)

		msg := &domain.CommunicationMessage{
			Base:       domain.NewBase(),
			TenantID:   tenantID,
			CampaignID: &campaignID,
			EventID:    eventID,
			GuestID:    guestID,
			Channel:    campaign.Channel,
			Type:       campaign.Type,
			Body:       renderedBody,
			Status:     domain.MessageStatusQueued,
		}

		if template.Subject != nil {
			subject := RenderTemplate(*template.Subject, vars)
			msg.Subject = &subject
		}

		if err := s.commRepo.CreateMessage(ctx, msg); err != nil {
			failedCount++
			continue
		}

		sentCount++
	}

	// Update campaign to completed
	completedAt := time.Now().UTC()
	if err := s.commRepo.UpdateCampaignStatus(ctx, tenantID, campaignID, domain.CampaignStatusCompleted, &now, &completedAt, sentCount, failedCount); err != nil {
		return fmt.Errorf("finalize campaign: %w", err)
	}

	return nil
}

// CancelCampaign cancels a scheduled or draft campaign.
func (s *CommunicationService) CancelCampaign(ctx context.Context, tenantID, campaignID uuid.UUID) error {
	campaign, err := s.commRepo.GetCampaign(ctx, tenantID, campaignID)
	if err != nil {
		return fmt.Errorf("get campaign: %w", err)
	}

	if campaign.Status == domain.CampaignStatusSending || campaign.Status == domain.CampaignStatusCompleted {
		return domain.ErrCampaignStarted
	}

	now := time.Now().UTC()
	if err := s.commRepo.UpdateCampaignStatus(ctx, tenantID, campaignID, domain.CampaignStatusCancelled, nil, &now, 0, 0); err != nil {
		return fmt.Errorf("cancel campaign: %w", err)
	}

	return nil
}

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}
