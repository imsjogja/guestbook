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

// TenantService encapsulates business logic for tenant operations.
type TenantService struct {
	tenantRepo     *repository.TenantRepository
	tenantUserRepo *repository.TenantUserRepository
	audit          *audit.Service
}

// NewTenantService creates a new TenantService.
func NewTenantService(
	tenantRepo *repository.TenantRepository,
	tenantUserRepo *repository.TenantUserRepository,
	audit *audit.Service,
) *TenantService {
	return &TenantService{
		tenantRepo:     tenantRepo,
		tenantUserRepo: tenantUserRepo,
		audit:          audit,
	}
}

// Create creates a new tenant and adds the creator as the tenant owner.
func (s *TenantService) Create(ctx context.Context, userID uuid.UUID, req domain.TenantCreateRequest) (*domain.Tenant, error) {
	// Normalize slug.
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))

	// Check slug uniqueness.
	exists, err := s.tenantRepo.SlugExists(ctx, req.Slug)
	if err != nil {
		return nil, fmt.Errorf("create tenant: check slug: %w", err)
	}
	if exists {
		return nil, domain.ErrDuplicateSlug
	}

	// Create tenant.
	tenant := domain.NewTenant(req.Name, req.Slug, userID)

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	// Add creator as owner.
	membership := domain.NewTenantMembership(tenant.ID, userID, domain.RoleTenantOwner, nil)
	membership.JoinedAt = &membership.CreatedAt

	if err := s.tenantUserRepo.Create(ctx, membership); err != nil {
		return nil, fmt.Errorf("create tenant: add owner membership: %w", err)
	}

	// Audit log.
	_ = s.audit.LogWithUser(ctx, userID, tenant.ID, domain.AuditActionCreate, domain.EntityTypeTenant, tenant.ID, nil, map[string]interface{}{
		"name": req.Name,
		"slug": req.Slug,
	})

	return tenant, nil
}

// Get retrieves a tenant by ID. Access check should be performed by the caller (middleware).
func (s *TenantService) Get(ctx context.Context, tenantID uuid.UUID) (*domain.Tenant, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}

	// Check if trial has expired and update status if needed.
	if tenant.Status == domain.TenantStatusTrial && tenant.IsTrialExpired() {
		tenant.Status = domain.TenantStatusSuspended
		if updateErr := s.tenantRepo.Update(ctx, tenant); updateErr != nil {
			// Non-critical: log but don't fail the request.
			_ = updateErr
		}
	}

	return tenant, nil
}

// Update updates tenant properties. The userID is recorded in the audit log.
func (s *TenantService) Update(ctx context.Context, tenantID, userID uuid.UUID, req domain.TenantUpdateRequest) (*domain.Tenant, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}

	oldValues := map[string]interface{}{
		"name":          tenant.Name,
		"description":   tenant.Description,
		"primary_color": tenant.PrimaryColor,
		"settings":      tenant.Settings,
	}

	// Apply updates.
	if req.Name != "" {
		tenant.Name = req.Name
	}
	if req.Description != "" {
		tenant.Description = &req.Description
	}
	if req.PrimaryColor != "" {
		tenant.PrimaryColor = req.PrimaryColor
	}
	if req.Settings != nil {
		tenant.Settings = req.Settings
	}

	tenant.Touch()

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}

	newValues := map[string]interface{}{
		"name":          tenant.Name,
		"description":   tenant.Description,
		"primary_color": tenant.PrimaryColor,
		"settings":      tenant.Settings,
	}

	// Audit log.
	_ = s.audit.LogWithUser(ctx, userID, tenantID, domain.AuditActionUpdate, domain.EntityTypeTenant, tenantID, oldValues, newValues)

	return tenant, nil
}

// ListMyTenants lists all tenants where the user is an active member.
func (s *TenantService) ListMyTenants(ctx context.Context, userID uuid.UUID) ([]*domain.Tenant, error) {
	tenants, err := s.tenantRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list my tenants: %w", err)
	}
	return tenants, nil
}

// InviteUser invites a user to a tenant by email address.
func (s *TenantService) InviteUser(ctx context.Context, tenantID, invitedBy uuid.UUID, email, role string) error {
	// Validate role.
	if !domain.IsValidRole(role) {
		return domain.ErrInvalidRole
	}

	// Prevent assigning tenant_owner via invite.
	if role == domain.RoleTenantOwner {
		return domain.ErrForbidden
	}

	// Verify tenant exists.
	if _, err := s.tenantRepo.GetByID(ctx, tenantID); err != nil {
		return fmt.Errorf("invite user: %w", err)
	}

	// TODO: Look up user by email in the user repository.
	// For now, we assume the user exists and we have their ID.
	// In a real implementation, this would:
	// 1. Find the user by email
	// 2. If not found, send an invitation email to register
	// 3. Create a pending membership

	// Check if membership already exists.
	existing, err := s.tenantUserRepo.Get(ctx, tenantID, uuid.Nil) // would look up by email-derived userID
	if err == nil && existing != nil {
		return domain.ErrAlreadyExists
	}

	// Audit log.
	_ = s.audit.LogWithUser(ctx, invitedBy, tenantID, domain.AuditActionInvite, domain.EntityTypeMembership, tenantID, nil, map[string]interface{}{
		"email": email,
		"role":  role,
	})

	// Placeholder: actual implementation would create membership and send email.
	_ = email

	return nil
}

// RemoveUser removes a user from a tenant.
func (s *TenantService) RemoveUser(ctx context.Context, tenantID, removedBy, targetUserID uuid.UUID) error {
	// Prevent self-removal of owner.
	membership, err := s.tenantUserRepo.Get(ctx, tenantID, targetUserID)
	if err != nil {
		return fmt.Errorf("remove user: %w", err)
	}

	if membership.Role == domain.RoleTenantOwner {
		return domain.ErrCannotRemoveOwner
	}

	if err := s.tenantUserRepo.SoftDelete(ctx, tenantID, targetUserID); err != nil {
		return fmt.Errorf("remove user: %w", err)
	}

	// Audit log.
	_ = s.audit.LogWithUser(ctx, removedBy, tenantID, domain.AuditActionRemove, domain.EntityTypeMembership, tenantID, map[string]interface{}{
		"user_id": targetUserID.String(),
		"role":    membership.Role,
	}, nil)

	return nil
}

// UpdateUserRole changes a member's role within a tenant.
func (s *TenantService) UpdateUserRole(ctx context.Context, tenantID, changedBy, targetUserID uuid.UUID, newRole string) error {
	// Validate role.
	if !domain.IsValidRole(newRole) {
		return domain.ErrInvalidRole
	}

	// Get current membership.
	membership, err := s.tenantUserRepo.Get(ctx, tenantID, targetUserID)
	if err != nil {
		return fmt.Errorf("update user role: %w", err)
	}

	// Prevent changing the owner's role.
	if membership.Role == domain.RoleTenantOwner {
		return domain.ErrOwnerRoleImmutable
	}

	// Prevent assigning owner role.
	if newRole == domain.RoleTenantOwner {
		return domain.ErrForbidden
	}

	oldRole := membership.Role

	if err := s.tenantUserRepo.UpdateRole(ctx, tenantID, targetUserID, newRole); err != nil {
		return fmt.Errorf("update user role: %w", err)
	}

	// Audit log.
	_ = s.audit.LogWithUser(ctx, changedBy, tenantID, domain.AuditActionUpdate, domain.EntityTypeMembership, tenantID, map[string]interface{}{
		"user_id":  targetUserID.String(),
		"old_role": oldRole,
	}, map[string]interface{}{
		"user_id":  targetUserID.String(),
		"new_role": newRole,
	})

	return nil
}
