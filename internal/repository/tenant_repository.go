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

// TenantRepository provides data access for tenants with tenant isolation and soft-delete support.
type TenantRepository struct {
	db *sqlx.DB
}

// NewTenantRepository creates a new TenantRepository.
func NewTenantRepository(db *sqlx.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

// Create inserts a new tenant into the database.
func (r *TenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		INSERT INTO tenants (
			id, name, slug, description, logo_url, primary_color, settings, status, trial_ends_at,
			created_at, updated_at
		) VALUES (
			:id, :name, :slug, :description, :logo_url, :primary_color, :settings, :status, :trial_ends_at,
			:created_at, :updated_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, tenant)
	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}
	return nil
}

// GetByID retrieves a tenant by its ID, respecting soft-delete.
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	var tenant domain.Tenant
	query := `
		SELECT * FROM tenants
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &tenant, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, fmt.Errorf("failed to get tenant by id: %w", err)
	}
	return &tenant, nil
}

// GetBySlug retrieves a tenant by its slug, respecting soft-delete.
func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	var tenant domain.Tenant
	query := `
		SELECT * FROM tenants
		WHERE slug = $1 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &tenant, query, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, fmt.Errorf("failed to get tenant by slug: %w", err)
	}
	return &tenant, nil
}

// Update updates an existing tenant. Only non-zero fields are updated.
func (r *TenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		UPDATE tenants SET
			name = COALESCE(NULLIF(:name, ''), name),
			description = COALESCE(:description, description),
			logo_url = COALESCE(:logo_url, logo_url),
			primary_color = COALESCE(NULLIF(:primary_color, ''), primary_color),
			settings = COALESCE(:settings, settings),
			status = COALESCE(NULLIF(:status, ''), status),
			trial_ends_at = COALESCE(:trial_ends_at, trial_ends_at),
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`
	result, err := r.db.NamedExecContext(ctx, query, tenant)
	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if rows == 0 {
		return domain.ErrTenantNotFound
	}

	return nil
}

// ListByUser lists all tenants where the specified user is an active member.
func (r *TenantRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Tenant, error) {
	var tenants []*domain.Tenant
	query := `
		SELECT t.* FROM tenants t
		INNER JOIN tenant_users tm ON t.id = tm.tenant_id
		WHERE tm.user_id = $1
		  AND tm.status = 'active'
		  AND t.deleted_at IS NULL
		ORDER BY t.created_at DESC
	`
	err := r.db.SelectContext(ctx, &tenants, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants by user: %w", err)
	}
	return tenants, nil
}

// ListActive lists all tenants that are in trial or active state.
func (r *TenantRepository) ListActive(ctx context.Context) ([]*domain.Tenant, error) {
	var tenants []*domain.Tenant
	query := `
		SELECT * FROM tenants
		WHERE status IN ('trial', 'active')
		  AND deleted_at IS NULL
	`
	err := r.db.SelectContext(ctx, &tenants, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list active tenants: %w", err)
	}
	return tenants, nil
}

// SoftDelete marks a tenant as deleted (soft-delete pattern).
func (r *TenantRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE tenants
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to soft-delete tenant: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}
	if rows == 0 {
		return domain.ErrTenantNotFound
	}

	return nil
}

// SlugExists checks whether a slug is already in use by a non-deleted tenant.
func (r *TenantRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM tenants
			WHERE slug = $1 AND deleted_at IS NULL
		)
	`
	err := r.db.GetContext(ctx, &exists, query, slug)
	if err != nil {
		return false, fmt.Errorf("failed to check slug existence: %w", err)
	}
	return exists, nil
}
