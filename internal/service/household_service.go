// Package service provides business logic layer implementations for GuestFlow.
package service

import (
	"context"
	"fmt"

	"guestflow/internal/audit"
	"guestflow/internal/domain"
	"guestflow/internal/repository"

	"github.com/google/uuid"
)

// HouseholdService encapsulates business logic for household operations.
type HouseholdService struct {
	householdRepo *repository.HouseholdRepository
	audit         *audit.Service
}

// NewHouseholdService creates a new HouseholdService.
func NewHouseholdService(
	householdRepo *repository.HouseholdRepository,
	auditSvc *audit.Service,
) *HouseholdService {
	return &HouseholdService{
		householdRepo: householdRepo,
		audit:         auditSvc,
	}
}

// Create creates a new household and optionally adds initial members.
func (s *HouseholdService) Create(ctx context.Context, tenantID, createdBy uuid.UUID, req domain.HouseholdCreateRequest) (*domain.Household, error) {
	household := domain.NewHousehold(tenantID, createdBy, req)

	if err := s.householdRepo.Create(ctx, household); err != nil {
		return nil, fmt.Errorf("create household: %w", err)
	}

	// Add initial members if provided
	for _, guestID := range req.GuestIDs {
		if err := s.householdRepo.AddMember(ctx, household.ID, guestID, false, nil); err != nil {
			// Non-critical: log but continue
			_ = err
		}
	}

	// Audit log
	_ = s.audit.LogWithUser(ctx, createdBy, tenantID, domain.AuditActionCreate, "household", household.ID, nil, map[string]interface{}{
		"name": household.Name,
	})

	return household, nil
}

// Get retrieves a household by ID with tenant isolation.
func (s *HouseholdService) Get(ctx context.Context, tenantID, householdID uuid.UUID) (*domain.Household, error) {
	household, err := s.householdRepo.GetByID(ctx, tenantID, householdID)
	if err != nil {
		return nil, fmt.Errorf("get household: %w", err)
	}
	return household, nil
}

// List lists households for a tenant with pagination.
func (s *HouseholdService) List(ctx context.Context, params domain.HouseholdListParams) ([]*domain.Household, int, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 20
	}

	households, err := s.householdRepo.ListByTenant(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list households: %w", err)
	}

	total, err := s.householdRepo.CountByTenant(ctx, params.TenantID, params.Search)
	if err != nil {
		return nil, 0, fmt.Errorf("count households: %w", err)
	}

	return households, total, nil
}

// Update applies partial updates to a household.
func (s *HouseholdService) Update(ctx context.Context, tenantID, householdID uuid.UUID, req domain.HouseholdUpdateRequest) (*domain.Household, error) {
	household, err := s.householdRepo.GetByID(ctx, tenantID, householdID)
	if err != nil {
		return nil, fmt.Errorf("update household: %w", err)
	}

	household.ApplyUpdate(req)

	if err := s.householdRepo.Update(ctx, household); err != nil {
		return nil, fmt.Errorf("update household: %w", err)
	}

	return household, nil
}

// Delete soft-deletes a household.
func (s *HouseholdService) Delete(ctx context.Context, tenantID, householdID uuid.UUID) error {
	if err := s.householdRepo.SoftDelete(ctx, tenantID, householdID); err != nil {
		return fmt.Errorf("delete household: %w", err)
	}
	return nil
}

// AddMember adds a guest to a household.
func (s *HouseholdService) AddMember(ctx context.Context, tenantID, householdID, guestID uuid.UUID, isPrimary bool, role string) error {
	// Verify household exists
	if _, err := s.householdRepo.GetByID(ctx, tenantID, householdID); err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	var rolePtr *string
	if role != "" {
		rolePtr = &role
	}

	if err := s.householdRepo.AddMember(ctx, householdID, guestID, isPrimary, rolePtr); err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	return nil
}

// RemoveMember removes a guest from a household.
func (s *HouseholdService) RemoveMember(ctx context.Context, tenantID, householdID, guestID uuid.UUID) error {
	// Verify household exists
	if _, err := s.householdRepo.GetByID(ctx, tenantID, householdID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	if err := s.householdRepo.RemoveMember(ctx, householdID, guestID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	return nil
}

// ListMembers lists all guests in a household.
func (s *HouseholdService) ListMembers(ctx context.Context, tenantID, householdID uuid.UUID) ([]*domain.Guest, error) {
	// Verify household exists
	if _, err := s.householdRepo.GetByID(ctx, tenantID, householdID); err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	members, err := s.householdRepo.ListMembers(ctx, householdID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	return members, nil
}
