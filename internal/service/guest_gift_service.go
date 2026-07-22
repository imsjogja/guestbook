package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"guestflow/internal/audit"
	"guestflow/internal/domain"
	"guestflow/internal/repository"

	"github.com/google/uuid"
)

type GuestGiftService struct {
	repo           *repository.GuestGiftRepository
	eventGuestRepo *repository.EventGuestRepository
	audit          *audit.Service
}

func NewGuestGiftService(repo *repository.GuestGiftRepository, eventGuestRepo *repository.EventGuestRepository, auditSvc *audit.Service) *GuestGiftService {
	return &GuestGiftService{repo: repo, eventGuestRepo: eventGuestRepo, audit: auditSvc}
}

func (s *GuestGiftService) List(ctx context.Context, tenantID, eventID uuid.UUID) ([]*domain.GuestGift, error) {
	return s.repo.ListByEvent(ctx, tenantID, eventID)
}

func (s *GuestGiftService) Upsert(ctx context.Context, tenantID, eventID, guestID, userID uuid.UUID, req domain.GuestGiftUpsertRequest) (*domain.GuestGift, error) {
	eventGuest, err := s.eventGuestRepo.GetByEventAndGuest(ctx, tenantID, eventID, guestID)
	if err != nil {
		return nil, fmt.Errorf("get event guest for gift: %w", err)
	}
	if eventGuest.Status != domain.EventGuestStatusActive {
		return nil, fmt.Errorf("event guest is not active: %w", domain.ErrInvalidInput)
	}

	giftType := strings.TrimSpace(req.GiftType)
	if giftType == "" {
		giftType = domain.GuestGiftTypeCash
	}
	if !domain.IsValidGuestGiftType(giftType) {
		return nil, fmt.Errorf("invalid gift type: %w", domain.ErrInvalidInput)
	}
	if req.Amount != nil && *req.Amount < 1 {
		return nil, fmt.Errorf("amount must be greater than zero: %w", domain.ErrInvalidInput)
	}
	if (giftType == domain.GuestGiftTypeCash || giftType == domain.GuestGiftTypeTransfer) && req.Amount == nil {
		return nil, fmt.Errorf("amount is required for cash or transfer gifts: %w", domain.ErrInvalidInput)
	}

	notes := strings.TrimSpace(req.Notes)
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}

	item, err := s.repo.GetByEventAndGuest(ctx, tenantID, eventID, guestID)
	action := domain.AuditActionCreate
	if errorsIsNotFound(err) {
		item = &domain.GuestGift{
			Base:         domain.NewBase(),
			TenantID:     tenantID,
			EventID:      eventID,
			GuestID:      guestID,
			EventGuestID: eventGuest.ID,
			ReceivedAt:   domain.NewBase().CreatedAt,
		}
	} else if err != nil {
		return nil, fmt.Errorf("find guest gift: %w", err)
	} else {
		action = domain.AuditActionUpdate
		item.Touch()
	}

	item.EventGuestID = eventGuest.ID
	item.Amount = req.Amount
	item.GiftType = giftType
	item.Notes = notesPtr
	item.RecordedBy = &userID
	if item.ReceivedAt.IsZero() {
		item.ReceivedAt = item.CreatedAt
	}
	if err := s.repo.Upsert(ctx, item); err != nil {
		return nil, err
	}
	if s.audit != nil {
		_ = s.audit.LogWithUser(ctx, userID, tenantID, action, domain.EntityTypeGuestGift, item.ID, nil, map[string]interface{}{
			"event_id": eventID, "guest_id": guestID, "amount": req.Amount, "gift_type": giftType,
		})
	}
	return s.repo.GetByEventAndGuest(ctx, tenantID, eventID, guestID)
}

func (s *GuestGiftService) Delete(ctx context.Context, tenantID, eventID, guestID, userID uuid.UUID) error {
	if err := s.repo.Delete(ctx, tenantID, eventID, guestID); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.LogWithUser(ctx, userID, tenantID, domain.AuditActionDelete, domain.EntityTypeGuestGift, guestID, nil, map[string]interface{}{
			"event_id": eventID, "guest_id": guestID,
		})
	}
	return nil
}

// Keep the not-found check local to avoid exposing repository implementation details.
func errorsIsNotFound(err error) bool {
	return errors.Is(err, domain.ErrNotFound)
}
