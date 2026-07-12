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

// GuestRepository provides data access operations for the Guest domain model
// with full tenant isolation and soft-delete support.
type GuestRepository struct {
	db *sqlx.DB
}

// NewGuestRepository creates a new GuestRepository instance.
func NewGuestRepository(db *sqlx.DB) *GuestRepository {
	return &GuestRepository{db: db}
}

// Create inserts a new guest into the database.
func (r *GuestRepository) Create(ctx context.Context, guest *domain.Guest) error {
	query := `
		INSERT INTO guests (
			id, tenant_id, full_name, nickname, phone, email, address, city, country,
			language, guest_type, segment, institution, title, relationship, pic,
			accessibility_needs, dietary_restrictions, allergies, notes,
			consent_communication, consent_version, source, is_active,
			created_by, updated_by, created_at, updated_at, deleted_at
		) VALUES (
			:id, :tenant_id, :full_name, :nickname, :phone, :email, :address, :city, :country,
			:language, :guest_type, :segment, :institution, :title, :relationship, :pic,
			:accessibility_needs, :dietary_restrictions, :allergies, :notes,
			:consent_communication, :consent_version, :source, :is_active,
			:created_by, :updated_by, :created_at, :updated_at, :deleted_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, guest)
	if err != nil {
		return fmt.Errorf("create guest: %w", err)
	}
	return nil
}

// GetByID retrieves a guest by UUID, ignoring tenant (for system operations).
func (r *GuestRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Guest, error) {
	var guest domain.Guest
	query := `
		SELECT * FROM guests
		WHERE id = $1 AND deleted_at IS NULL
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &guest, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get guest by id: %w", err)
	}
	return &guest, nil
}

// GetByIDForTenant retrieves a guest by UUID with tenant isolation.
func (r *GuestRepository) GetByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*domain.Guest, error) {
	var guest domain.Guest
	query := `
		SELECT * FROM guests
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &guest, query, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get guest by id for tenant: %w", err)
	}
	return &guest, nil
}

// Update modifies an existing guest's fields in the database.
func (r *GuestRepository) Update(ctx context.Context, guest *domain.Guest) error {
	query := `
		UPDATE guests SET
			full_name = :full_name,
			nickname = :nickname,
			phone = :phone,
			email = :email,
			address = :address,
			city = :city,
			country = :country,
			language = :language,
			guest_type = :guest_type,
			segment = :segment,
			institution = :institution,
			title = :title,
			relationship = :relationship,
			pic = :pic,
			accessibility_needs = :accessibility_needs,
			dietary_restrictions = :dietary_restrictions,
			allergies = :allergies,
			notes = :notes,
			consent_communication = :consent_communication,
			consent_version = :consent_version,
			source = :source,
			is_active = :is_active,
			updated_by = :updated_by,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id AND deleted_at IS NULL
	`
	result, err := r.db.NamedExecContext(ctx, query, guest)
	if err != nil {
		return fmt.Errorf("update guest: %w", err)
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

// SoftDelete marks a guest as deleted by setting the deleted_at timestamp.
func (r *GuestRepository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	now := time.Now().UTC()

	query := `
		UPDATE guests
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, now, id, tenantID)
	if err != nil {
		return fmt.Errorf("soft delete guest: %w", err)
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

// ListByTenant lists guests for a tenant with optional search, filters, and pagination.
func (r *GuestRepository) ListByTenant(ctx context.Context, params domain.GuestListParams) ([]*domain.Guest, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 20
	}
	offset := (params.Page - 1) * params.PerPage

	where, args := r.buildWhereClause(params)
	query := fmt.Sprintf(`
		SELECT * FROM guests
		%s
		ORDER BY created_at DESC
		LIMIT %d OFFSET %d
	`, where, params.PerPage, offset)

	var guests []*domain.Guest
	err := r.db.SelectContext(ctx, &guests, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list guests by tenant: %w", err)
	}
	return guests, nil
}

// CountByTenant returns the total count of guests matching the filter parameters.
func (r *GuestRepository) CountByTenant(ctx context.Context, params domain.GuestListParams) (int, error) {
	where, args := r.buildWhereClause(params)
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM guests
		%s
	`, where)

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("count guests by tenant: %w", err)
	}
	return count, nil
}

// buildWhereClause constructs the WHERE clause and arguments for guest list queries.
func (r *GuestRepository) buildWhereClause(params domain.GuestListParams) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	// Tenant isolation + soft delete
	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d AND deleted_at IS NULL", argIdx))
	args = append(args, params.TenantID)
	argIdx++

	// Search by name, phone, or email
	if strings.TrimSpace(params.Search) != "" {
		search := "%" + strings.TrimSpace(params.Search) + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(full_name ILIKE $%d OR phone ILIKE $%d OR email ILIKE $%d)",
			argIdx, argIdx, argIdx,
		))
		args = append(args, search)
		argIdx++
	}

	// Filter by guest type
	if params.GuestType != "" {
		conditions = append(conditions, fmt.Sprintf("guest_type = $%d", argIdx))
		args = append(args, params.GuestType)
		argIdx++
	}

	// Filter by segment
	if params.Segment != "" {
		conditions = append(conditions, fmt.Sprintf("segment = $%d", argIdx))
		args = append(args, params.Segment)
		argIdx++
	}

	// Filter by active status
	if params.Status != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *params.Status)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")
	return where, args
}

// BulkCreate inserts multiple guests in a single batch operation using a transaction.
// This is optimized for CSV import scenarios.
func (r *GuestRepository) BulkCreate(ctx context.Context, guests []*domain.Guest) error {
	if len(guests) == 0 {
		return nil
	}

	return RunInTransaction(ctx, r.db, func(tx *sqlx.Tx) error {
		query := `
			INSERT INTO guests (
				id, tenant_id, full_name, nickname, phone, email, address, city, country,
				language, guest_type, segment, institution, title, relationship, pic,
				accessibility_needs, dietary_restrictions, allergies, notes,
				consent_communication, consent_version, source, is_active,
				created_by, updated_by, created_at, updated_at, deleted_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9,
				$10, $11, $12, $13, $14, $15, $16,
				$17, $18, $19, $20,
				$21, $22, $23, $24,
				$25, $26, $27, $28, $29
			)
		`

		stmt, err := tx.PrepareContext(ctx, query)
		if err != nil {
			return fmt.Errorf("prepare bulk insert: %w", err)
		}
		defer stmt.Close()

		for _, g := range guests {
			_, err := stmt.ExecContext(ctx,
				g.ID, g.TenantID, g.FullName, g.Nickname, g.Phone, g.Email,
				g.Address, g.City, g.Country, g.Language, g.GuestType,
				g.Segment, g.Institution, g.Title, g.Relationship, g.PIC,
				g.AccessibilityNeeds, g.DietaryRestrictions, g.Allergies, g.Notes,
				g.ConsentCommunication, g.ConsentVersion, g.Source, g.IsActive,
				g.CreatedBy, g.UpdatedBy, g.CreatedAt, g.UpdatedAt, g.DeletedAt,
			)
			if err != nil {
				return fmt.Errorf("bulk insert guest %s: %w", g.ID, err)
			}
		}

		return nil
	})
}

// CheckDuplicates checks if any guests with the given phones or emails already exist
// within the tenant. Returns a map of field values to guest IDs.
func (r *GuestRepository) CheckDuplicates(ctx context.Context, tenantID uuid.UUID, phones, emails []string) (map[string]uuid.UUID, error) {
	duplicates := make(map[string]uuid.UUID)

	// Check phone duplicates
	if len(phones) > 0 {
		query, args, err := sqlx.In(`
			SELECT phone, id FROM guests
			WHERE tenant_id = ? AND deleted_at IS NULL AND phone IN (?)
		`, tenantID, phones)
		if err != nil {
			return nil, fmt.Errorf("check phone duplicates: %w", err)
		}
		query = r.db.Rebind(query)

		rows, err := r.db.QueryxContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("query phone duplicates: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var phone string
			var id uuid.UUID
			if err := rows.Scan(&phone, &id); err != nil {
				return nil, fmt.Errorf("scan phone duplicate: %w", err)
			}
			duplicates["phone:"+phone] = id
		}
	}

	// Check email duplicates
	if len(emails) > 0 {
		query, args, err := sqlx.In(`
			SELECT email, id FROM guests
			WHERE tenant_id = ? AND deleted_at IS NULL AND email IN (?)
		`, tenantID, emails)
		if err != nil {
			return nil, fmt.Errorf("check email duplicates: %w", err)
		}
		query = r.db.Rebind(query)

		rows, err := r.db.QueryxContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("query email duplicates: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var email string
			var id uuid.UUID
			if err := rows.Scan(&email, &id); err != nil {
				return nil, fmt.Errorf("scan email duplicate: %w", err)
			}
			duplicates["email:"+email] = id
		}
	}

	return duplicates, nil
}

// FindByPhoneOrEmail searches for a guest by phone or email within a tenant.
func (r *GuestRepository) FindByPhoneOrEmail(ctx context.Context, tenantID uuid.UUID, phone, email string) (*domain.Guest, error) {
	var guest domain.Guest
	var query string
	var args []interface{}

	if phone != "" && email != "" {
		query = `
			SELECT * FROM guests
			WHERE tenant_id = $1 AND deleted_at IS NULL
			  AND (phone = $2 OR email = $3)
			LIMIT 1
		`
		args = []interface{}{tenantID, phone, email}
	} else if phone != "" {
		query = `
			SELECT * FROM guests
			WHERE tenant_id = $1 AND deleted_at IS NULL AND phone = $2
			LIMIT 1
		`
		args = []interface{}{tenantID, phone}
	} else if email != "" {
		query = `
			SELECT * FROM guests
			WHERE tenant_id = $1 AND deleted_at IS NULL AND email = $2
			LIMIT 1
		`
		args = []interface{}{tenantID, email}
	} else {
		return nil, errors.New("phone or email required")
	}

	err := r.db.GetContext(ctx, &guest, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("find guest by phone or email: %w", err)
	}
	return &guest, nil
}
