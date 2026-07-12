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

// EventLocationRepository provides data access for event locations with tenant isolation and soft-delete support.
type EventLocationRepository struct {
	db *sqlx.DB
}

// NewEventLocationRepository creates a new EventLocationRepository.
func NewEventLocationRepository(db *sqlx.DB) *EventLocationRepository {
	return &EventLocationRepository{db: db}
}

// Create inserts a new event location into the database.
func (r *EventLocationRepository) Create(ctx context.Context, location *domain.EventLocation) error {
	query := `
		INSERT INTO event_locations (
			id, tenant_id, name, address, city, maps_url,
			latitude, longitude, instructions,
			created_at, updated_at
		) VALUES (
			:id, :tenant_id, :name, :address, :city, :maps_url,
			:latitude, :longitude, :instructions,
			:created_at, :updated_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, location)
	if err != nil {
		return fmt.Errorf("failed to create event location: %w", err)
	}
	return nil
}

// GetByID retrieves an event location by its ID.
func (r *EventLocationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.EventLocation, error) {
	var location domain.EventLocation
	query := `
		SELECT * FROM event_locations
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &location, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrEventLocationNotFound
		}
		return nil, fmt.Errorf("failed to get event location by id: %w", err)
	}
	return &location, nil
}

// GetByIDForTenant retrieves an event location by ID ensuring it belongs to the given tenant.
func (r *EventLocationRepository) GetByIDForTenant(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*domain.EventLocation, error) {
	var location domain.EventLocation
	query := `
		SELECT * FROM event_locations
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &location, query, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrEventLocationNotFound
		}
		return nil, fmt.Errorf("failed to get event location by id for tenant: %w", err)
	}
	return &location, nil
}

// ListByTenant lists all event locations for a tenant.
func (r *EventLocationRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.EventLocation, error) {
	var locations []*domain.EventLocation
	query := `
		SELECT * FROM event_locations
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY name ASC
	`
	err := r.db.SelectContext(ctx, &locations, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list event locations by tenant: %w", err)
	}
	return locations, nil
}

// Update updates an existing event location.
func (r *EventLocationRepository) Update(ctx context.Context, location *domain.EventLocation) error {
	query := `
		UPDATE event_locations SET
			name = COALESCE(NULLIF(:name, ''), name),
			address = COALESCE(:address, address),
			city = COALESCE(:city, city),
			maps_url = COALESCE(:maps_url, maps_url),
			latitude = COALESCE(:latitude, latitude),
			longitude = COALESCE(:longitude, longitude),
			instructions = COALESCE(:instructions, instructions),
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id AND deleted_at IS NULL
	`
	result, err := r.db.NamedExecContext(ctx, query, location)
	if err != nil {
		return fmt.Errorf("failed to update event location: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if rows == 0 {
		return domain.ErrEventLocationNotFound
	}

	return nil
}

// SoftDelete marks an event location as deleted (soft-delete pattern).
func (r *EventLocationRepository) SoftDelete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	query := `
		UPDATE event_locations
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to soft-delete event location: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}
	if rows == 0 {
		return domain.ErrEventLocationNotFound
	}

	return nil
}
