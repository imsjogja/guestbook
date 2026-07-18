package service

import (
	"context"
	"errors"
	"fmt"

	"guestflow/internal/domain"
	"guestflow/internal/repository"

	"github.com/google/uuid"
)

// EventAccessService resolves tenant and event-scoped permissions.
type EventAccessService struct {
	eventRepo       *repository.EventRepository
	eventMemberRepo *repository.EventMemberRepository
	tenantUserRepo  *repository.TenantUserRepository
}

type EventAccess struct {
	Role        string   `json:"role"`
	Scope       string   `json:"scope"`
	Permissions []string `json:"permissions"`
}

func NewEventAccessService(
	eventRepo *repository.EventRepository,
	eventMemberRepo *repository.EventMemberRepository,
	tenantUserRepo *repository.TenantUserRepository,
) *EventAccessService {
	return &EventAccessService{
		eventRepo:       eventRepo,
		eventMemberRepo: eventMemberRepo,
		tenantUserRepo:  tenantUserRepo,
	}
}

// Resolve returns the effective role for a user within an event.
func (s *EventAccessService) Resolve(ctx context.Context, tenantID, eventID, userID uuid.UUID) (*EventAccess, error) {
	if _, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID); err != nil {
		return nil, err
	}

	tenantMember, err := s.tenantUserRepo.Get(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve event access: %w", domain.ErrForbidden)
	}
	if tenantMember.Status != domain.MembershipStatusActive {
		return nil, fmt.Errorf("resolve event access: %w", domain.ErrForbidden)
	}

	if tenantMember.Role == domain.RoleTenantOwner || tenantMember.Role == domain.RoleEventManager {
		return &EventAccess{
			Role:        tenantMember.Role,
			Scope:       "tenant",
			Permissions: domain.RolePermissions[tenantMember.Role],
		}, nil
	}

	eventMember, err := s.eventMemberRepo.Get(ctx, tenantID, eventID, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve event access: %w", domain.ErrForbidden)
	}
	permissions, ok := domain.RolePermissions[eventMember.Role]
	if !ok {
		return nil, fmt.Errorf("resolve event access: %w", domain.ErrForbidden)
	}

	return &EventAccess{
		Role:        eventMember.Role,
		Scope:       "event",
		Permissions: permissions,
	}, nil
}

func (s *EventAccessService) Authorize(ctx context.Context, tenantID, eventID, userID uuid.UUID, permission string) error {
	access, err := s.Resolve(ctx, tenantID, eventID, userID)
	if err != nil {
		return err
	}
	for _, allowed := range access.Permissions {
		if allowed == permission {
			return nil
		}
	}
	return domain.ErrForbidden
}

func (s *EventAccessService) GetAccess(ctx context.Context, tenantID, eventID, userID uuid.UUID) (*EventAccess, error) {
	return s.Resolve(ctx, tenantID, eventID, userID)
}

// ListAccessibleEvents limits event selectors to the user's effective scope.
func (s *EventAccessService) ListAccessibleEvents(ctx context.Context, tenantID, userID uuid.UUID, filter domain.EventFilter) ([]*domain.Event, int, error) {
	member, err := s.tenantUserRepo.Get(ctx, tenantID, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("list accessible events: %w", err)
	}
	if member.Role == domain.RoleTenantOwner || member.Role == domain.RoleEventManager {
		return s.listAllEvents(ctx, tenantID, filter)
	}
	return s.eventMemberRepo.ListEventsByUser(ctx, tenantID, userID, filter)
}

func (s *EventAccessService) listAllEvents(ctx context.Context, tenantID uuid.UUID, filter domain.EventFilter) ([]*domain.Event, int, error) {
	events, err := s.eventRepo.ListByTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.eventRepo.CountByTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

func IsEventAccessDenied(err error) bool {
	return errors.Is(err, domain.ErrForbidden) || errors.Is(err, domain.ErrMembershipNotFound)
}
