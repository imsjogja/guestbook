// Package service provides business logic layer implementations for GuestFlow.
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

// GuestService encapsulates business logic for guest operations.
type GuestService struct {
	guestRepo     *repository.GuestRepository
	checkinRepo   *repository.CheckinRepository
	householdRepo *repository.HouseholdRepository
	tagRepo       *repository.GuestTagRepository
	audit         *audit.Service
	importService *ImportService
}

// GuestDetail is the tenant guest profile enriched with check-in history.
type GuestDetail struct {
	*domain.Guest
	Checkins []domain.Checkin `json:"checkins"`
}

// NewGuestService creates a new GuestService.
func NewGuestService(
	guestRepo *repository.GuestRepository,
	checkinRepo *repository.CheckinRepository,
	householdRepo *repository.HouseholdRepository,
	tagRepo *repository.GuestTagRepository,
	auditSvc *audit.Service,
) *GuestService {
	svc := &GuestService{
		guestRepo:     guestRepo,
		checkinRepo:   checkinRepo,
		householdRepo: householdRepo,
		tagRepo:       tagRepo,
		audit:         auditSvc,
	}
	// Import service is initialized with a reference back to the guest repo
	svc.importService = NewImportService(guestRepo)
	return svc
}

// Create creates a new guest after validating and checking for duplicates.
func (s *GuestService) Create(ctx context.Context, tenantID, createdBy uuid.UUID, req domain.GuestCreateRequest) (*domain.Guest, error) {
	// Normalize inputs
	req.FullName = strings.TrimSpace(req.FullName)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = strings.TrimSpace(req.Phone)

	// Check for duplicates by email
	if req.Email != "" {
		existing, err := s.guestRepo.FindByPhoneOrEmail(ctx, tenantID, "", req.Email)
		if err == nil && existing != nil {
			return nil, domain.ErrAlreadyExists
		}
	}

	// Check for duplicates by phone
	if req.Phone != "" {
		existing, err := s.guestRepo.FindByPhoneOrEmail(ctx, tenantID, req.Phone, "")
		if err == nil && existing != nil {
			return nil, domain.ErrAlreadyExists
		}
	}

	guest := domain.NewGuest(tenantID, createdBy, req)

	if err := s.guestRepo.Create(ctx, guest); err != nil {
		return nil, fmt.Errorf("create guest: %w", err)
	}

	// Audit log
	_ = s.audit.LogWithUser(ctx, createdBy, tenantID, domain.AuditActionCreate, domain.EntityTypeGuest, guest.ID, nil, map[string]interface{}{
		"full_name":  guest.FullName,
		"guest_type": guest.GuestType,
	})

	return guest, nil
}

// Get retrieves a guest by ID with tenant isolation.
func (s *GuestService) Get(ctx context.Context, tenantID, guestID uuid.UUID) (*GuestDetail, error) {
	guest, err := s.guestRepo.GetByIDForTenant(ctx, tenantID, guestID)
	if err != nil {
		return nil, fmt.Errorf("get guest: %w", err)
	}

	checkins, err := s.checkinRepo.ListByGuest(ctx, tenantID, guestID)
	if err != nil {
		return nil, fmt.Errorf("get guest check-ins: %w", err)
	}

	return &GuestDetail{Guest: guest, Checkins: checkins}, nil
}

// List lists guests for a tenant with filtering and pagination.
func (s *GuestService) List(ctx context.Context, params domain.GuestListParams) ([]*domain.Guest, int, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 20
	}

	guests, err := s.guestRepo.ListByTenant(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list guests: %w", err)
	}

	total, err := s.guestRepo.CountByTenant(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("count guests: %w", err)
	}

	return guests, total, nil
}

// Update applies partial updates to a guest.
func (s *GuestService) Update(ctx context.Context, tenantID, guestID, updatedBy uuid.UUID, req domain.GuestUpdateRequest) (*domain.Guest, error) {
	guest, err := s.guestRepo.GetByIDForTenant(ctx, tenantID, guestID)
	if err != nil {
		return nil, fmt.Errorf("update guest: %w", err)
	}

	oldValues := map[string]interface{}{
		"full_name":  guest.FullName,
		"guest_type": guest.GuestType,
		"phone":      guest.Phone,
		"email":      guest.Email,
		"is_active":  guest.IsActive,
	}

	// Check for email conflicts if changing email
	if req.Email != "" && (guest.Email == nil || *guest.Email != req.Email) {
		existing, err := s.guestRepo.FindByPhoneOrEmail(ctx, tenantID, "", req.Email)
		if err == nil && existing != nil && existing.ID != guestID {
			return nil, domain.ErrAlreadyExists
		}
	}

	// Check for phone conflicts if changing phone
	if req.Phone != "" && (guest.Phone == nil || *guest.Phone != req.Phone) {
		existing, err := s.guestRepo.FindByPhoneOrEmail(ctx, tenantID, req.Phone, "")
		if err == nil && existing != nil && existing.ID != guestID {
			return nil, domain.ErrAlreadyExists
		}
	}

	guest.ApplyUpdate(req, updatedBy)

	if err := s.guestRepo.Update(ctx, guest); err != nil {
		return nil, fmt.Errorf("update guest: %w", err)
	}

	newValues := map[string]interface{}{
		"full_name":  guest.FullName,
		"guest_type": guest.GuestType,
		"phone":      guest.Phone,
		"email":      guest.Email,
		"is_active":  guest.IsActive,
	}

	// Audit log
	_ = s.audit.LogWithUser(ctx, updatedBy, tenantID, domain.AuditActionUpdate, domain.EntityTypeGuest, guestID, oldValues, newValues)

	return guest, nil
}

// Delete soft-deletes a guest.
func (s *GuestService) Delete(ctx context.Context, tenantID, guestID, deletedBy uuid.UUID) error {
	if err := s.guestRepo.SoftDelete(ctx, tenantID, guestID); err != nil {
		return fmt.Errorf("delete guest: %w", err)
	}

	// Audit log
	_ = s.audit.LogWithUser(ctx, deletedBy, tenantID, domain.AuditActionDelete, domain.EntityTypeGuest, guestID, map[string]interface{}{
		"deleted": true,
	}, nil)

	return nil
}

// ImportCSV handles CSV import delegating to ImportService.
func (s *GuestService) ImportCSV(ctx context.Context, tenantID, createdBy uuid.UUID, content []byte) (*domain.GuestImportResult, error) {
	return s.importService.ImportCSV(ctx, tenantID, createdBy, content)
}

// ImportCSVForEvent reuses existing tenant guests and imports their event
// roster associations without weakening tenant-level duplicate protection.
func (s *GuestService) ImportCSVForEvent(ctx context.Context, tenantID, createdBy uuid.UUID, content []byte) (*domain.GuestImportResult, error) {
	return s.importService.ImportCSVForEvent(ctx, tenantID, createdBy, content)
}

// Search searches guests by name, phone, or email.
func (s *GuestService) Search(ctx context.Context, tenantID uuid.UUID, query string) ([]*domain.Guest, error) {
	params := domain.GuestListParams{
		TenantID: tenantID,
		Search:   query,
		Page:     1,
		PerPage:  50,
	}

	guests, err := s.guestRepo.ListByTenant(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search guests: %w", err)
	}
	return guests, nil
}

// CheckDuplicates checks for duplicate guests by phone or email.
func (s *GuestService) CheckDuplicates(ctx context.Context, tenantID uuid.UUID, phones, emails []string) (map[string]uuid.UUID, error) {
	return s.guestRepo.CheckDuplicates(ctx, tenantID, phones, emails)
}
