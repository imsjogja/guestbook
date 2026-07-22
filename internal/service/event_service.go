package service

import (
	"context"
	"fmt"

	"guestflow/internal/audit"
	"guestflow/internal/domain"
	"guestflow/internal/repository"

	"github.com/google/uuid"
)

// EventService encapsulates business logic for event operations.
type EventService struct {
	eventRepo         *repository.EventRepository
	eventSessionRepo  *repository.EventSessionRepository
	eventLocationRepo *repository.EventLocationRepository
	auditSvc          *audit.Service
	billingSvc        *BillingService
}

// NewEventService creates a new EventService.
func NewEventService(
	eventRepo *repository.EventRepository,
	eventSessionRepo *repository.EventSessionRepository,
	eventLocationRepo *repository.EventLocationRepository,
	auditSvc *audit.Service,
	billingSvc *BillingService,
) *EventService {
	return &EventService{
		eventRepo:         eventRepo,
		eventSessionRepo:  eventSessionRepo,
		eventLocationRepo: eventLocationRepo,
		auditSvc:          auditSvc,
	}
}

// Create creates a new event for a tenant. The event is created with status "draft".
func (s *EventService) Create(ctx context.Context, tenantID, userID uuid.UUID, req domain.EventCreateRequest) (*domain.Event, error) {
	// Check subscription quota
	if s.billingSvc != nil {
		subStatus, err := s.billingSvc.GetSubscriptionStatus(ctx, tenantID)
		if err == nil && subStatus.MaxEvents != nil {
			count, err := s.eventRepo.CountByTenant(ctx, tenantID, domain.EventFilter{})
			if err == nil && count >= *subStatus.MaxEvents {
				return nil, fmt.Errorf("quota exceeded: maximum number of events reached for your current plan")
			}
		}
	}

	// Validate event type.
	if !domain.IsValidEventType(req.Type) {
		return nil, fmt.Errorf("create event: %w", domain.ErrInvalidInput)
	}

	// Validate date logic.
	if req.EndDate != nil && req.EndDate.Before(req.StartDate) {
		return nil, fmt.Errorf("create event: %w", domain.ErrInvalidInput)
	}

	// Validate RSVP deadline is before start date.
	if req.RSVPDeadline != nil && req.RSVPDeadline.After(req.StartDate) {
		return nil, fmt.Errorf("create event: %w", domain.ErrInvalidInput)
	}

	event := &domain.Event{
		TenantBase: domain.TenantBase{
			Base:     domain.NewBase(),
			TenantID: tenantID,
		},
		Name:             req.Name,
		Type:             req.Type,
		Status:           domain.EventStatusDraft,
		StartDate:        req.StartDate,
		EndDate:          req.EndDate,
		RSVPDeadline:     req.RSVPDeadline,
		Capacity:         req.Capacity,
		TargetInvites:    req.TargetInvites,
		TargetAttendance: req.TargetAttendance,
		Settings:         make(domain.JSONMap),
		CreatedBy:        userID,
	}

	if req.Description != "" {
		event.Description = &req.Description
	}
	if req.DressCode != "" {
		event.DressCode = &req.DressCode
	}
	if req.PrivacyNotice != "" {
		event.PrivacyNotice = &req.PrivacyNotice
	}
	if req.GuestPolicy != "" {
		event.GuestPolicy = &req.GuestPolicy
	}

	if err := s.eventRepo.Create(ctx, event); err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}

	// Audit log.
	_ = s.auditSvc.LogWithUser(ctx, userID, tenantID, domain.AuditActionCreate, domain.EntityTypeEvent, event.ID, nil, map[string]interface{}{
		"name":       req.Name,
		"type":       req.Type,
		"start_date": req.StartDate,
	})

	return event, nil
}

// Get retrieves an event by ID within a tenant.
func (s *EventService) Get(ctx context.Context, tenantID, eventID uuid.UUID) (*domain.Event, error) {
	event, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	return event, nil
}

// GetBySelfCheckinToken resolves the event QR token without requiring a tenant header.
func (s *EventService) GetBySelfCheckinToken(ctx context.Context, token string) (*domain.Event, error) {
	event, err := s.eventRepo.GetBySelfCheckinToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("get event by self check-in token: %w", err)
	}
	return event, nil
}

// List lists events for a tenant with optional filtering and pagination.
func (s *EventService) List(ctx context.Context, tenantID uuid.UUID, filter domain.EventFilter) ([]*domain.Event, int, error) {
	// Normalize pagination defaults.
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.PerPage > 100 {
		filter.PerPage = 100
	}

	events, err := s.eventRepo.ListByTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("list events: %w", err)
	}

	total, err := s.eventRepo.CountByTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("list events: count: %w", err)
	}

	return events, total, nil
}

// Update updates an existing event. Prevents updates to completed or cancelled events.
func (s *EventService) Update(ctx context.Context, tenantID, userID, eventID uuid.UUID, req domain.EventUpdateRequest) (*domain.Event, error) {
	event, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}

	// Prevent updates to completed or cancelled events.
	if event.Status == domain.EventStatusCompleted || event.Status == domain.EventStatusCancelled {
		return nil, fmt.Errorf("update event: %w", domain.ErrEventCannotModify)
	}

	// Apply updates.
	if req.Name != "" {
		event.Name = req.Name
	}
	if req.Type != "" {
		if !domain.IsValidEventType(req.Type) {
			return nil, fmt.Errorf("update event: invalid type: %w", domain.ErrInvalidInput)
		}
		event.Type = req.Type
	}
	if req.Description != "" {
		event.Description = &req.Description
	}
	if req.Status != "" {
		if !domain.IsValidEventStatus(req.Status) {
			return nil, fmt.Errorf("update event: invalid status: %w", domain.ErrInvalidInput)
		}
		event.Status = req.Status
	}
	if req.StartDate != nil {
		event.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		event.EndDate = req.EndDate
	}
	if req.RSVPDeadline != nil {
		event.RSVPDeadline = req.RSVPDeadline
	}
	if req.Capacity != nil {
		event.Capacity = req.Capacity
	}
	if req.TargetInvites != nil {
		event.TargetInvites = req.TargetInvites
	}
	if req.TargetAttendance != nil {
		event.TargetAttendance = req.TargetAttendance
	}
	if req.DressCode != "" {
		event.DressCode = &req.DressCode
	}
	if req.PrivacyNotice != "" {
		event.PrivacyNotice = &req.PrivacyNotice
	}
	if req.GuestPolicy != "" {
		event.GuestPolicy = &req.GuestPolicy
	}
	if req.Settings != nil {
		event.Settings = req.Settings
	}

	// Validate date logic after updates.
	if event.EndDate != nil && event.EndDate.Before(event.StartDate) {
		return nil, fmt.Errorf("update event: end date before start date: %w", domain.ErrInvalidInput)
	}
	if event.RSVPDeadline != nil && event.RSVPDeadline.After(event.StartDate) {
		return nil, fmt.Errorf("update event: rsvp deadline after start date: %w", domain.ErrInvalidInput)
	}

	event.Touch()

	if err := s.eventRepo.Update(ctx, event); err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}

	// Audit log.
	_ = s.auditSvc.LogWithUser(ctx, userID, tenantID, domain.AuditActionUpdate, domain.EntityTypeEvent, eventID, nil, map[string]interface{}{
		"name":   event.Name,
		"status": event.Status,
		"type":   event.Type,
	})

	return event, nil
}

// Publish validates required fields and publishes an event.
func (s *EventService) Publish(ctx context.Context, tenantID, userID, eventID uuid.UUID) (*domain.Event, error) {
	event, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("publish event: %w", err)
	}

	// Only draft events can be published.
	if event.Status != domain.EventStatusDraft {
		return nil, fmt.Errorf("publish event: %w", domain.ErrEventInvalidStatusTransition)
	}

	// Validate required fields for publishing.
	if event.Name == "" {
		return nil, fmt.Errorf("publish event: name is required: %w", domain.ErrInvalidInput)
	}
	if event.Type == "" {
		return nil, fmt.Errorf("publish event: type is required: %w", domain.ErrInvalidInput)
	}
	if event.StartDate.IsZero() {
		return nil, fmt.Errorf("publish event: start date is required: %w", domain.ErrInvalidInput)
	}

	// Update status to published.
	if err := s.eventRepo.UpdateStatus(ctx, eventID, tenantID, domain.EventStatusPublished); err != nil {
		return nil, fmt.Errorf("publish event: %w", err)
	}

	// Re-fetch to get updated state.
	event, err = s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("publish event: refetch: %w", err)
	}

	// Audit log.
	_ = s.auditSvc.LogWithUser(ctx, userID, tenantID, domain.AuditActionUpdate, domain.EntityTypeEvent, eventID, map[string]interface{}{
		"status": domain.EventStatusDraft,
	}, map[string]interface{}{
		"status": domain.EventStatusPublished,
	})

	return event, nil
}

// SoftDelete soft-deletes an event. Prevents deletion of ongoing events.
func (s *EventService) SoftDelete(ctx context.Context, tenantID, userID, eventID uuid.UUID) error {
	event, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID)
	if err != nil {
		return fmt.Errorf("soft-delete event: %w", err)
	}

	// Prevent deletion of ongoing events.
	if event.Status == domain.EventStatusOngoing {
		return fmt.Errorf("soft-delete event: %w", domain.ErrEventCannotDelete)
	}

	if err := s.eventRepo.SoftDelete(ctx, eventID, tenantID); err != nil {
		return fmt.Errorf("soft-delete event: %w", err)
	}

	// Audit log.
	_ = s.auditSvc.LogWithUser(ctx, userID, tenantID, domain.AuditActionDelete, domain.EntityTypeEvent, eventID, map[string]interface{}{
		"name":   event.Name,
		"status": event.Status,
	}, nil)

	return nil
}

// GetSessions retrieves all sessions for an event.
func (s *EventService) GetSessions(ctx context.Context, tenantID, eventID uuid.UUID) ([]*domain.EventSession, error) {
	// Verify the event exists and belongs to the tenant.
	if _, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID); err != nil {
		return nil, fmt.Errorf("get event sessions: %w", err)
	}

	sessions, err := s.eventSessionRepo.ListByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("get event sessions: %w", err)
	}
	return sessions, nil
}

// GetLocations retrieves all locations for a tenant.
func (s *EventService) GetLocations(ctx context.Context, tenantID uuid.UUID) ([]*domain.EventLocation, error) {
	locations, err := s.eventLocationRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get event locations: %w", err)
	}
	return locations, nil
}
