package rbac

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"guestflow/internal/domain"
	"guestflow/internal/repository"

	"github.com/google/uuid"
)

// permissionCacheEntry stores cached permissions with an expiry time.
type permissionCacheEntry struct {
	role      string
	perms     []string
	expiresAt time.Time
}

// Service handles authorization checks based on tenant membership roles.
type Service struct {
	membershipRepo *repository.TenantUserRepository
	cache          map[string]permissionCacheEntry
	mu             sync.RWMutex
	ttl            time.Duration
}

// NewService creates a new RBAC Service.
func NewService(membershipRepo *repository.TenantUserRepository) *Service {
	return &Service{
		membershipRepo: membershipRepo,
		cache:          make(map[string]permissionCacheEntry),
		ttl:            5 * time.Minute,
	}
}

// cacheKey generates a cache key for tenant+user lookups.
func cacheKey(tenantID, userID uuid.UUID) string {
	return tenantID.String() + ":" + userID.String()
}

// getCachedPermissions returns cached permissions if present and not expired.
func (s *Service) getCachedPermissions(tenantID, userID uuid.UUID) (string, []string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.cache[cacheKey(tenantID, userID)]
	if !ok || time.Now().After(entry.expiresAt) {
		return "", nil, false
	}
	return entry.role, entry.perms, true
}

// setCachedPermissions stores permissions in the cache.
func (s *Service) setCachedPermissions(tenantID, userID uuid.UUID, role string, perms []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache[cacheKey(tenantID, userID)] = permissionCacheEntry{
		role:      role,
		perms:     perms,
		expiresAt: time.Now().Add(s.ttl),
	}
}

// invalidateCache removes cached permissions for a tenant+user.
func (s *Service) invalidateCache(tenantID, userID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.cache, cacheKey(tenantID, userID))
}

// HasPermission checks if the user has the given permission within the tenant.
func (s *Service) HasPermission(ctx context.Context, tenantID, userID uuid.UUID, permission string) (bool, error) {
	_, perms, ok := s.getCachedPermissions(tenantID, userID)
	if !ok {
		role, err := s.membershipRepo.GetRole(ctx, tenantID, userID)
		if err != nil {
			if errors.Is(err, domain.ErrMembershipNotFound) {
				return false, nil
			}
			return false, fmt.Errorf("has permission: %w", err)
		}
		perms = GetPermissionsForRole(role)
		s.setCachedPermissions(tenantID, userID, role, perms)
	}

	return hasPermission(perms, permission), nil
}

// GetRole returns the user's role in the specified tenant.
func (s *Service) GetRole(ctx context.Context, tenantID, userID uuid.UUID) (string, error) {
	role, _, ok := s.getCachedPermissions(tenantID, userID)
	if ok {
		return role, nil
	}

	role, err := s.membershipRepo.GetRole(ctx, tenantID, userID)
	if err != nil {
		return "", fmt.Errorf("get role: %w", err)
	}

	perms := GetPermissionsForRole(role)
	s.setCachedPermissions(tenantID, userID, role, perms)

	return role, nil
}

// GetPermissions returns all permissions the user has in the specified tenant.
func (s *Service) GetPermissions(ctx context.Context, tenantID, userID uuid.UUID) ([]string, error) {
	role, perms, ok := s.getCachedPermissions(tenantID, userID)
	if !ok {
		var err error
		role, err = s.membershipRepo.GetRole(ctx, tenantID, userID)
		if err != nil {
			if errors.Is(err, domain.ErrMembershipNotFound) {
				return []string{}, nil
			}
			return nil, fmt.Errorf("get permissions: %w", err)
		}
		perms = GetPermissionsForRole(role)
		s.setCachedPermissions(tenantID, userID, role, perms)
	}

	return perms, nil
}

// EnforcePermission returns an error if the user does not have the required permission.
func (s *Service) EnforcePermission(ctx context.Context, tenantID, userID uuid.UUID, permission string) error {
	ok, err := s.HasPermission(ctx, tenantID, userID, permission)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrForbidden
	}
	return nil
}

// EnforceAnyPermission returns an error if the user has none of the specified permissions.
func (s *Service) EnforceAnyPermission(ctx context.Context, tenantID, userID uuid.UUID, permissions ...string) error {
	if len(permissions) == 0 {
		return nil
	}

	_, perms, ok := s.getCachedPermissions(tenantID, userID)
	if !ok {
		role, err := s.membershipRepo.GetRole(ctx, tenantID, userID)
		if err != nil {
			if errors.Is(err, domain.ErrMembershipNotFound) {
				return domain.ErrForbidden
			}
			return fmt.Errorf("enforce any permission: %w", err)
		}
		perms = GetPermissionsForRole(role)
		s.setCachedPermissions(tenantID, userID, role, perms)
	}

	if !hasAnyPermission(perms, permissions...) {
		return domain.ErrForbidden
	}
	return nil
}

// EnforceAllPermissions returns an error if the user does not have all specified permissions.
func (s *Service) EnforceAllPermissions(ctx context.Context, tenantID, userID uuid.UUID, permissions ...string) error {
	if len(permissions) == 0 {
		return nil
	}

	_, perms, ok := s.getCachedPermissions(tenantID, userID)
	if !ok {
		role, err := s.membershipRepo.GetRole(ctx, tenantID, userID)
		if err != nil {
			if errors.Is(err, domain.ErrMembershipNotFound) {
				return domain.ErrForbidden
			}
			return fmt.Errorf("enforce all permissions: %w", err)
		}
		perms = GetPermissionsForRole(role)
		s.setCachedPermissions(tenantID, userID, role, perms)
	}

	if !hasAllPermissions(perms, permissions...) {
		return domain.ErrForbidden
	}
	return nil
}

// ServiceInterface defines the methods needed by middleware
// This avoids circular imports between rbac and middleware packages
type ServiceInterface interface {
	HasPermission(ctx context.Context, tenantID, userID uuid.UUID, permission string) (bool, error)
	GetRole(ctx context.Context, tenantID, userID uuid.UUID) (string, error)
	GetPermissions(ctx context.Context, tenantID, userID uuid.UUID) ([]string, error)
	EnforcePermission(ctx context.Context, tenantID, userID uuid.UUID, permission string) error
	EnforceAnyPermission(ctx context.Context, tenantID, userID uuid.UUID, permissions ...string) error
	EnforceAllPermissions(ctx context.Context, tenantID, userID uuid.UUID, permissions ...string) error
}

// Compile-time check that Service implements ServiceInterface
var _ ServiceInterface = (*Service)(nil)
