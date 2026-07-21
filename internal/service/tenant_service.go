package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"guestflow/internal/audit"
	"guestflow/internal/auth"
	"guestflow/internal/domain"
	"guestflow/internal/repository"

	"github.com/google/uuid"
)

// TenantService encapsulates business logic for tenant operations.
type TenantService struct {
	tenantRepo     *repository.TenantRepository
	tenantUserRepo *repository.TenantUserRepository
	userRepo       *repository.UserRepository
	audit          *audit.Service
	billingSvc     *BillingService
}

// NewTenantService creates a new TenantService.
func NewTenantService(
	tenantRepo *repository.TenantRepository,
	tenantUserRepo *repository.TenantUserRepository,
	userRepo *repository.UserRepository,
	audit *audit.Service,
	billingSvc *BillingService,
) *TenantService {
	return &TenantService{
		tenantRepo:     tenantRepo,
		tenantUserRepo: tenantUserRepo,
		userRepo:       userRepo,
		audit:          audit,
		billingSvc:     billingSvc,
	}
}

// TenantMemberRecord pairs a tenant membership with its user profile.
type TenantMemberRecord struct {
	Membership *domain.TenantMembership
	User       *domain.User
}

// TenantAccess describes the authenticated user's effective tenant permissions.
type TenantAccess struct {
	Role        string   `json:"role"`
	Scope       string   `json:"scope"`
	Permissions []string `json:"permissions"`
}

// GetAccess returns the effective role and permissions for a tenant member.
func (s *TenantService) GetAccess(ctx context.Context, tenantID, userID uuid.UUID) (*TenantAccess, error) {
	membership, err := s.tenantUserRepo.Get(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("get tenant access: %w", err)
	}

	return &TenantAccess{
		Role:        membership.Role,
		Scope:       "tenant",
		Permissions: append([]string(nil), domain.RolePermissions[membership.Role]...),
	}, nil
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

// ListMembers returns the active members for a tenant, joined with user profiles.
func (s *TenantService) ListMembers(ctx context.Context, tenantID uuid.UUID) ([]TenantMemberRecord, error) {
	memberships, err := s.tenantUserRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list tenant members: %w", err)
	}

	records := make([]TenantMemberRecord, 0, len(memberships))
	for _, membership := range memberships {
		user, userErr := s.userRepo.GetByID(ctx, membership.UserID)
		if userErr != nil {
			if errors.Is(userErr, domain.ErrUserNotFound) {
				continue
			}
			return nil, fmt.Errorf("list tenant members: get user %s: %w", membership.UserID, userErr)
		}
		records = append(records, TenantMemberRecord{
			Membership: membership,
			User:       user,
		})
	}

	return records, nil
}

// AddUser creates or reactivates a tenant member directly as an active,
// email-verified account. This replaces the previous invitation flow.
func (s *TenantService) AddUser(ctx context.Context, tenantID, addedBy uuid.UUID, req domain.TenantMemberCreateRequest) error {
	// Validate role.
	if !domain.IsValidRole(req.Role) {
		return domain.ErrInvalidRole
	}

	// Check subscription quota
	if s.billingSvc != nil {
		subStatus, err := s.billingSvc.GetSubscriptionStatus(ctx, tenantID)
		if err == nil && subStatus.MaxTeamMembers != nil {
			memberships, err := s.tenantUserRepo.ListByTenant(ctx, tenantID)
			if err == nil && len(memberships) >= *subStatus.MaxTeamMembers {
				return fmt.Errorf("quota exceeded: maximum number of team members reached for your current plan")
			}
		}
	}

	// Prevent assigning tenant_owner to a new member.
	if req.Role == domain.RoleTenantOwner {
		return domain.ErrForbidden
	}

	// Verify tenant exists.
	if _, err := s.tenantRepo.GetByID(ctx, tenantID); err != nil {
		return fmt.Errorf("add user: %w", err)
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)
	req.Phone = strings.TrimSpace(req.Phone)
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	now := time.Now().UTC()
	if errors.Is(err, domain.ErrUserNotFound) {
		passwordHash, hashErr := auth.HashPassword(req.Password)
		if hashErr != nil {
			return fmt.Errorf("add user: hash password: %w", hashErr)
		}
		user = &domain.User{
			Base:            domain.NewBase(),
			Email:           req.Email,
			PasswordHash:    passwordHash,
			FullName:        req.FullName,
			Phone:           optionalString(req.Phone),
			EmailVerifiedAt: &now,
			Status:          domain.UserStatusActive,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return fmt.Errorf("add user: create account: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("add user: lookup account: %w", err)
	} else {
		// Existing accounts keep their password, but become verified and active
		// when an owner explicitly adds them to this tenant.
		user.EmailVerifiedAt = &now
		if user.Status != domain.UserStatusActive {
			user.Status = domain.UserStatusActive
		}
		if err := s.userRepo.Update(ctx, user); err != nil {
			return fmt.Errorf("add user: activate account: %w", err)
		}
	}

	// Check active memberships before upserting an inactive one.
	existing, err := s.tenantUserRepo.Get(ctx, tenantID, user.ID)
	if err == nil && existing != nil {
		return domain.ErrAlreadyExists
	}
	if err != nil && !errors.Is(err, domain.ErrMembershipNotFound) {
		return fmt.Errorf("add user: check membership: %w", err)
	}

	membership := domain.NewTenantMembership(tenantID, user.ID, req.Role, &addedBy)
	membership.Status = domain.MembershipStatusActive
	membership.JoinedAt = &now

	if err := s.tenantUserRepo.UpsertActive(ctx, membership); err != nil {
		return fmt.Errorf("add user: create membership: %w", err)
	}

	// Audit log.
	_ = s.audit.LogWithUser(ctx, addedBy, tenantID, domain.AuditActionCreate, domain.EntityTypeMembership, tenantID, nil, map[string]interface{}{
		"email":  req.Email,
		"role":   req.Role,
		"status": domain.MembershipStatusActive,
	})

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
