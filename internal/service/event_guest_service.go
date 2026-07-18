package service

import (
	"context"
	"fmt"
	"strings"

	"guestflow/internal/audit"
	"guestflow/internal/domain"
	"guestflow/internal/repository"

	"github.com/google/uuid"
)

type EventGuestService struct {
	repo      *repository.EventGuestRepository
	eventRepo *repository.EventRepository
	guestRepo *repository.GuestRepository
	guestSvc  *GuestService
	audit     *audit.Service
}

func NewEventGuestService(repo *repository.EventGuestRepository, eventRepo *repository.EventRepository, guestRepo *repository.GuestRepository, guestSvc *GuestService, auditSvc *audit.Service) *EventGuestService {
	return &EventGuestService{repo: repo, eventRepo: eventRepo, guestRepo: guestRepo, guestSvc: guestSvc, audit: auditSvc}
}

func (s *EventGuestService) Create(ctx context.Context, tenantID, eventID, userID uuid.UUID, req domain.EventGuestCreateRequest) (*domain.EventGuest, error) {
	if _, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID); err != nil {
		return nil, fmt.Errorf("create event guest: %w", err)
	}
	if _, err := s.guestRepo.GetByIDForTenant(ctx, tenantID, req.GuestID); err != nil {
		return nil, fmt.Errorf("create event guest: %w", err)
	}
	if _, err := s.repo.GetByEventAndGuest(ctx, tenantID, eventID, req.GuestID); err == nil {
		return nil, domain.ErrAlreadyExists
	} else if err != domain.ErrNotFound {
		return nil, fmt.Errorf("check event guest: %w", err)
	}

	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = domain.EventGuestSourceManual
	}
	if !domain.IsValidEventGuestSource(source) {
		return nil, fmt.Errorf("create event guest: %w", domain.ErrInvalidInput)
	}
	if req.MaxPax < 1 {
		req.MaxPax = 1
	}
	if req.Adults == 0 && req.Children == 0 {
		req.Adults = 1
	}
	eventGuest := &domain.EventGuest{
		Base: domain.NewBase(), TenantID: tenantID, EventID: eventID, GuestID: req.GuestID,
		Status: domain.EventGuestStatusActive, Source: source, MaxPax: req.MaxPax,
		Adults: req.Adults, Children: req.Children, PlusOneAllowed: req.PlusOneAllowed,
		CreatedBy: userID,
	}
	if strings.TrimSpace(req.Notes) != "" {
		notes := strings.TrimSpace(req.Notes)
		eventGuest.Notes = &notes
	}
	if err := s.repo.Create(ctx, eventGuest); err != nil {
		return nil, err
	}
	_ = s.audit.LogWithUser(ctx, userID, tenantID, domain.AuditActionCreate, domain.EntityTypeEventGuest, eventGuest.ID, nil, map[string]interface{}{"event_id": eventID, "guest_id": req.GuestID})
	eventGuest.Guest, _ = s.guestRepo.GetByIDForTenant(ctx, tenantID, req.GuestID)
	return eventGuest, nil
}

func (s *EventGuestService) List(ctx context.Context, params domain.EventGuestListParams) ([]*domain.EventGuest, int, error) {
	items, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *EventGuestService) ImportCSV(ctx context.Context, tenantID, eventID, userID uuid.UUID, content []byte) (*domain.GuestImportResult, error) {
	if _, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID); err != nil {
		return nil, fmt.Errorf("import event guests: %w", err)
	}
	result, err := s.guestSvc.ImportCSV(ctx, tenantID, userID, content)
	if err != nil {
		return nil, err
	}
	for _, guestID := range result.ImportedGuestIDs {
		item := &domain.EventGuest{
			Base: domain.NewBase(), TenantID: tenantID, EventID: eventID, GuestID: guestID,
			Status: domain.EventGuestStatusActive, Source: domain.EventGuestSourceImport,
			MaxPax: 1, Adults: 1, CreatedBy: userID,
		}
		if err := s.repo.Create(ctx, item); err != nil {
			return nil, fmt.Errorf("add imported guest to event: %w", err)
		}
	}
	return result, nil
}

func (s *EventGuestService) Cancel(ctx context.Context, tenantID, eventID, eventGuestID, userID uuid.UUID) error {
	if err := s.repo.Cancel(ctx, tenantID, eventID, eventGuestID); err != nil {
		return err
	}
	_ = s.audit.LogWithUser(ctx, userID, tenantID, domain.AuditActionUpdate, domain.EntityTypeEventGuest, eventGuestID, nil, map[string]interface{}{"status": domain.EventGuestStatusCancelled})
	return nil
}
