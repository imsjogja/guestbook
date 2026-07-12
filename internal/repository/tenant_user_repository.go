package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"guestflow/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// TenantUserRepository manages tenant membership data access.
type TenantUserRepository struct {
	db *sqlx.DB
}

// NewTenantUserRepository creates a new TenantUserRepository.
func NewTenantUserRepository(db *sqlx.DB) *TenantUserRepository {
	return &TenantUserRepository{db: db}
}

// Create inserts a new tenant membership record.
func (r *TenantUserRepository) Create(ctx context.Context, membership *domain.TenantMembership) error {
	query := `
		INSERT INTO tenant_memberships (
			id, tenant_id, user_id, role, invited_by, invited_at, joined_at, status,
			created_at, updated_at
		) VALUES (
			:id, :tenant_id, :user_id, :role, :invited_by, :invited_at, :joined_at, :status,
			:created_at, :updated_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, membership)
	if err != nil {
		return fmt.Errorf("failed to create tenant membership: %w", err)
	}
	return nil
}

// Get retrieves a specific membership by tenant ID and user ID.
func (r *TenantUserRepository) Get(ctx context.Context, tenantID, userID uuid.UUID) (*domain.TenantMembership, error) {
	var membership domain.TenantMembership
	query := `
		SELECT * FROM tenant_memberships
		WHERE tenant_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &membership, query, tenantID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMembershipNotFound
		}
		return nil, fmt.Errorf("failed to get tenant membership: %w", err)
	}
	return &membership, nil
}

// UpdateRole changes the role of an existing membership.
func (r *TenantUserRepository) UpdateRole(ctx context.Context, tenantID, userID uuid.UUID, role string) error {
	query := `
		UPDATE tenant_memberships
		SET role = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND user_id = $3 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, role, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to update membership role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if rows == 0 {
		return domain.ErrMembershipNotFound
	}

	return nil
}

// ListByTenant lists all active memberships for a given tenant.
func (r *TenantUserRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.TenantMembership, error) {
	var memberships []*domain.TenantMembership
	query := `
		SELECT * FROM tenant_memberships
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	err := r.db.SelectContext(ctx, &memberships, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list memberships by tenant: %w", err)
	}
	return memberships, nil
}

// ListByUser lists all memberships for a given user across all tenants.
func (r *TenantUserRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.TenantMembership, error) {
	var memberships []*domain.TenantMembership
	query := `
		SELECT * FROM tenant_memberships
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	err := r.db.SelectContext(ctx, &memberships, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list memberships by user: %w", err)
	}
	return memberships, nil
}

// SoftDelete removes a membership (soft-delete).
func (r *TenantUserRepository) SoftDelete(ctx context.Context, tenantID, userID uuid.UUID) error {
	query := `
		UPDATE tenant_memberships
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE tenant_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to soft-delete membership: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}
	if rows == 0 {
		return domain.ErrMembershipNotFound
	}

	return nil
}

// HasAccess checks whether a user has an active membership in a tenant.
func (r *TenantUserRepository) HasAccess(ctx context.Context, tenantID, userID uuid.UUID) (bool, error) {
	var hasAccess bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM tenant_memberships
			WHERE tenant_id = $1 AND user_id = $2
			  AND status = 'active'
			  AND deleted_at IS NULL
		)
	`
	err := r.db.GetContext(ctx, &hasAccess, query, tenantID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check tenant access: %w", err)
	}
	return hasAccess, nil
}

// GetRole retrieves the role for a user within a tenant.
func (r *TenantUserRepository) GetRole(ctx context.Context, tenantID, userID uuid.UUID) (string, error) {
	var role string
	query := `
		SELECT role FROM tenant_memberships
		WHERE tenant_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &role, query, tenantID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", domain.ErrMembershipNotFound
		}
		return "", fmt.Errorf("failed to get role: %w", err)
	}
	return role, nil
}
