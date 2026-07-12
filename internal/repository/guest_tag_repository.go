// Package repository provides data access layer implementations for GuestFlow.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"guestflow/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GuestTagRepository provides data access for guest tags with tenant isolation.
type GuestTagRepository struct {
	db *sqlx.DB
}

// NewGuestTagRepository creates a new GuestTagRepository instance.
func NewGuestTagRepository(db *sqlx.DB) *GuestTagRepository {
	return &GuestTagRepository{db: db}
}

// Create inserts a new guest tag.
func (r *GuestTagRepository) Create(ctx context.Context, tag *domain.GuestTag) error {
	query := `
		INSERT INTO guest_tags (id, tenant_id, name, color, description, created_at, updated_at)
		VALUES (:id, :tenant_id, :name, :color, :description, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, tag)
	if err != nil {
		return fmt.Errorf("create guest tag: %w", err)
	}
	return nil
}

// GetByID retrieves a tag by ID with tenant isolation.
func (r *GuestTagRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.GuestTag, error) {
	var tag domain.GuestTag
	query := `
		SELECT * FROM guest_tags
		WHERE id = $1 AND tenant_id = $2
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &tag, query, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get guest tag by id: %w", err)
	}
	return &tag, nil
}

// ListByTenant lists all tags for a tenant.
func (r *GuestTagRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.GuestTag, error) {
	var tags []*domain.GuestTag
	query := `
		SELECT * FROM guest_tags
		WHERE tenant_id = $1
		ORDER BY name ASC
	`
	err := r.db.SelectContext(ctx, &tags, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list guest tags by tenant: %w", err)
	}
	return tags, nil
}

// Update modifies an existing tag.
func (r *GuestTagRepository) Update(ctx context.Context, tag *domain.GuestTag) error {
	query := `
		UPDATE guest_tags SET
			name = COALESCE(NULLIF(:name, ''), name),
			color = COALESCE(:color, color),
			description = COALESCE(:description, description),
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id
	`
	result, err := r.db.NamedExecContext(ctx, query, tag)
	if err != nil {
		return fmt.Errorf("update guest tag: %w", err)
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

// Delete removes a tag permanently (tags have no soft-delete).
func (r *GuestTagRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	// First remove all assignments
	_, _ = r.db.ExecContext(ctx,
		`DELETE FROM guest_tag_assignments WHERE tag_id = $1`, id)

	query := `DELETE FROM guest_tags WHERE id = $1 AND tenant_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete guest tag: %w", err)
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

// AssignTag assigns a tag to a guest.
func (r *GuestTagRepository) AssignTag(ctx context.Context, guestID, tagID uuid.UUID) error {
	query := `
		INSERT INTO guest_tag_assignments (guest_id, tag_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, guestID, tagID)
	if err != nil {
		return fmt.Errorf("assign tag to guest: %w", err)
	}
	return nil
}

// RemoveTag removes a tag from a guest.
func (r *GuestTagRepository) RemoveTag(ctx context.Context, guestID, tagID uuid.UUID) error {
	query := `
		DELETE FROM guest_tag_assignments
		WHERE guest_id = $1 AND tag_id = $2
	`
	_, err := r.db.ExecContext(ctx, query, guestID, tagID)
	if err != nil {
		return fmt.Errorf("remove tag from guest: %w", err)
	}
	return nil
}

// ListTagsByGuest lists all tags assigned to a guest.
func (r *GuestTagRepository) ListTagsByGuest(ctx context.Context, guestID uuid.UUID) ([]*domain.GuestTag, error) {
	var tags []*domain.GuestTag
	query := `
		SELECT t.* FROM guest_tags t
		INNER JOIN guest_tag_assignments ta ON t.id = ta.tag_id
		WHERE ta.guest_id = $1
		ORDER BY t.name ASC
	`
	err := r.db.SelectContext(ctx, &tags, query, guestID)
	if err != nil {
		return nil, fmt.Errorf("list tags by guest: %w", err)
	}
	return tags, nil
}

// NameExists checks if a tag name already exists for the tenant.
func (r *GuestTagRepository) NameExists(ctx context.Context, tenantID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	args := []interface{}{tenantID, strings.ToLower(name)}
	query := `
		SELECT EXISTS(
			SELECT 1 FROM guest_tags
			WHERE tenant_id = $1 AND LOWER(name) = $2
	`
	if excludeID != nil {
		query += ` AND id != $3`
		args = append(args, *excludeID)
	}
	query += ` LIMIT 1)`

	err := r.db.GetContext(ctx, &exists, query, args...)
	if err != nil {
		return false, fmt.Errorf("check tag name exists: %w", err)
	}
	return exists, nil
}
