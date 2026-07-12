// Package repository provides data access layer implementations for GuestFlow.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"guestflow/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HouseholdRepository provides data access operations for Household with tenant isolation.
type HouseholdRepository struct {
	db *sqlx.DB
}

// NewHouseholdRepository creates a new HouseholdRepository instance.
func NewHouseholdRepository(db *sqlx.DB) *HouseholdRepository {
	return &HouseholdRepository{db: db}
}

// Create inserts a new household into the database.
func (r *HouseholdRepository) Create(ctx context.Context, household *domain.Household) error {
	query := `
		INSERT INTO households (
			id, tenant_id, name, description, address, city, max_pax, notes,
			created_by, created_at, updated_at, deleted_at
		) VALUES (
			:id, :tenant_id, :name, :description, :address, :city, :max_pax, :notes,
			:created_by, :created_at, :updated_at, :deleted_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, household)
	if err != nil {
		return fmt.Errorf("create household: %w", err)
	}
	return nil
}

// GetByID retrieves a household by UUID with tenant isolation.
func (r *HouseholdRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Household, error) {
	var household domain.Household
	query := `
		SELECT * FROM households
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &household, query, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get household by id: %w", err)
	}
	return &household, nil
}

// Update modifies an existing household.
func (r *HouseholdRepository) Update(ctx context.Context, household *domain.Household) error {
	query := `
		UPDATE households SET
			name = :name,
			description = :description,
			address = :address,
			city = :city,
			max_pax = :max_pax,
			notes = :notes,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id AND deleted_at IS NULL
	`
	result, err := r.db.NamedExecContext(ctx, query, household)
	if err != nil {
		return fmt.Errorf("update household: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check update result: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// SoftDelete marks a household as deleted with tenant isolation.
func (r *HouseholdRepository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	now := time.Now().UTC()

	query := `
		UPDATE households
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, now, id, tenantID)
	if err != nil {
		return fmt.Errorf("soft delete household: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check delete result: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// ListByTenant lists households for a tenant with optional search and pagination.
func (r *HouseholdRepository) ListByTenant(ctx context.Context, params domain.HouseholdListParams) ([]*domain.Household, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 20
	}
	offset := (params.Page - 1) * params.PerPage

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, params.TenantID)
	argIdx++

	conditions = append(conditions, "deleted_at IS NULL")

	if strings.TrimSpace(params.Search) != "" {
		search := "%" + strings.TrimSpace(params.Search) + "%"
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argIdx))
		args = append(args, search)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT * FROM households
		%s
		ORDER BY created_at DESC
		LIMIT %d OFFSET %d
	`, where, params.PerPage, offset)

	var households []*domain.Household
	err := r.db.SelectContext(ctx, &households, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list households by tenant: %w", err)
	}
	return households, nil
}

// CountByTenant returns the total count of households for a tenant.
func (r *HouseholdRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID, search string) (int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	conditions = append(conditions, "deleted_at IS NULL")

	if strings.TrimSpace(search) != "" {
		s := "%" + strings.TrimSpace(search) + "%"
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argIdx))
		args = append(args, s)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`SELECT COUNT(*) FROM households %s`, where)

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("count households by tenant: %w", err)
	}
	return count, nil
}

// AddMember adds a guest to a household.
func (r *HouseholdRepository) AddMember(ctx context.Context, householdID, guestID uuid.UUID, isPrimary bool, role *string) error {
	query := `
		INSERT INTO household_members (household_id, guest_id, is_primary, role)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (household_id, guest_id) DO UPDATE SET
			is_primary = EXCLUDED.is_primary,
			role = EXCLUDED.role
	`
	_, err := r.db.ExecContext(ctx, query, householdID, guestID, isPrimary, role)
	if err != nil {
		return fmt.Errorf("add household member: %w", err)
	}
	return nil
}

// RemoveMember removes a guest from a household.
func (r *HouseholdRepository) RemoveMember(ctx context.Context, householdID, guestID uuid.UUID) error {
	query := `
		DELETE FROM household_members
		WHERE household_id = $1 AND guest_id = $2
	`
	result, err := r.db.ExecContext(ctx, query, householdID, guestID)
	if err != nil {
		return fmt.Errorf("remove household member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check remove result: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// ListMembers lists all guests in a household.
func (r *HouseholdRepository) ListMembers(ctx context.Context, householdID uuid.UUID) ([]*domain.Guest, error) {
	query := `
		SELECT g.* FROM guests g
		INNER JOIN household_members hm ON g.id = hm.guest_id
		WHERE hm.household_id = $1 AND g.deleted_at IS NULL
		ORDER BY g.full_name ASC
	`

	var guests []*domain.Guest
	err := r.db.SelectContext(ctx, &guests, query, householdID)
	if err != nil {
		return nil, fmt.Errorf("list household members: %w", err)
	}
	return guests, nil
}
