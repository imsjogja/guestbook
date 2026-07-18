package service

import (
	"context"
	"errors"
	"fmt"

	"guestflow/internal/audit"
	"guestflow/internal/domain"
	"guestflow/internal/repository"

	"github.com/google/uuid"
)

// EventMemberRecord combines an event assignment with the user profile.
type EventMemberRecord struct {
	Membership *domain.EventMember
	User       *domain.User
}

type EventMemberService struct {
	eventMemberRepo *repository.EventMemberRepository
	eventRepo       *repository.EventRepository
	tenantUserRepo  *repository.TenantUserRepository
	userRepo        *repository.UserRepository
	audit           *audit.Service
}

func NewEventMemberService(
	eventMemberRepo *repository.EventMemberRepository,
	eventRepo *repository.EventRepository,
	tenantUserRepo *repository.TenantUserRepository,
	userRepo *repository.UserRepository,
	auditSvc *audit.Service,
) *EventMemberService {
	return &EventMemberService{
		eventMemberRepo: eventMemberRepo,
		eventRepo:       eventRepo,
		tenantUserRepo:  tenantUserRepo,
		userRepo:        userRepo,
		audit:           auditSvc,
	}
}

func (s *EventMemberService) List(ctx context.Context, tenantID, eventID uuid.UUID) ([]EventMemberRecord, error) {
	if _, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID); err != nil {
		return nil, fmt.Errorf("list event members: %w", err)
	}
	members, err := s.eventMemberRepo.ListByEvent(ctx, tenantID, eventID)
	if err != nil {
		return nil, err
	}
	records := make([]EventMemberRecord, 0, len(members))
	for _, member := range members {
		user, err := s.userRepo.GetByID(ctx, member.UserID)
		if err != nil {
			if errors.Is(err, domain.ErrUserNotFound) {
				continue
			}
			return nil, fmt.Errorf("list event members: get user: %w", err)
		}
		records = append(records, EventMemberRecord{Membership: member, User: user})
	}
	return records, nil
}

func (s *EventMemberService) Create(ctx context.Context, tenantID, eventID, assignedBy uuid.UUID, req domain.EventMemberCreateRequest) (*domain.EventMember, error) {
	if !domain.IsValidEventMemberRole(req.Role) {
		return nil, domain.ErrInvalidRole
	}
	if _, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID); err != nil {
		return nil, fmt.Errorf("create event member: %w", err)
	}
	tenantMember, err := s.tenantUserRepo.Get(ctx, tenantID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("create event member: %w", err)
	}
	if tenantMember.Status != domain.MembershipStatusActive {
		return nil, domain.ErrForbidden
	}

	base := domain.NewBase()
	member := &domain.EventMember{
		Base:       base,
		TenantID:   tenantID,
		EventID:    eventID,
		UserID:     req.UserID,
		Role:       req.Role,
		Status:     domain.EventMemberStatusActive,
		InvitedBy:  &assignedBy,
		AssignedAt: base.CreatedAt,
	}
	if err := s.eventMemberRepo.Create(ctx, member); err != nil {
		return nil, fmt.Errorf("create event member: %w", err)
	}
	member, err = s.eventMemberRepo.Get(ctx, tenantID, eventID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("create event member: retrieve assignment: %w", err)
	}
	_ = s.audit.LogWithUser(ctx, assignedBy, tenantID, domain.AuditActionCreate, domain.EntityTypeMembership, member.ID, nil, map[string]interface{}{
		"event_id": member.EventID.String(),
		"user_id":  member.UserID.String(),
		"role":     member.Role,
	})
	return member, nil
}

func (s *EventMemberService) UpdateRole(ctx context.Context, tenantID, eventID, changedBy, userID uuid.UUID, role string) error {
	if !domain.IsValidEventMemberRole(role) {
		return domain.ErrInvalidRole
	}
	if err := s.eventMemberRepo.UpdateRole(ctx, tenantID, eventID, userID, role); err != nil {
		return err
	}
	_ = s.audit.LogWithUser(ctx, changedBy, tenantID, domain.AuditActionUpdate, domain.EntityTypeMembership, userID, nil, map[string]interface{}{
		"event_id": eventID.String(),
		"user_id":  userID.String(),
		"role":     role,
	})
	return nil
}

func (s *EventMemberService) Remove(ctx context.Context, tenantID, eventID, removedBy, userID uuid.UUID) error {
	if err := s.eventMemberRepo.Deactivate(ctx, tenantID, eventID, userID); err != nil {
		return err
	}
	_ = s.audit.LogWithUser(ctx, removedBy, tenantID, domain.AuditActionRemove, domain.EntityTypeMembership, userID, nil, map[string]interface{}{
		"event_id": eventID.String(),
		"user_id":  userID.String(),
	})
	return nil
}
