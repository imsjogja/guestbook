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

// EventRepository provides data access for events with tenant isolation and soft-delete support.
type EventRepository struct {
	db *sqlx.DB
}

// NewEventRepository creates a new EventRepository.
func NewEventRepository(db *sqlx.DB) *EventRepository {
	return &EventRepository{db: db}
}

// Create inserts a new event into the database.
func (r *EventRepository) Create(ctx context.Context, event *domain.Event) error {
	query := `
		INSERT INTO events (
			id, tenant_id, name, type, description, cover_url, status,
			start_date, end_date, rsvp_deadline, capacity, target_invites,
			target_attendance, primary_location_id, dress_code, privacy_notice,
			guest_policy, settings, created_by,
			created_at, updated_at
		) VALUES (
			:id, :tenant_id, :name, :type, :description, :cover_url, :status,
			:start_date, :end_date, :rsvp_deadline, :capacity, :target_invites,
			:target_attendance, :primary_location_id, :dress_code, :privacy_notice,
			:guest_policy, :settings, :created_by,
			:created_at, :updated_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, event)
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}
	return nil
}

// GetByID retrieves an event by its ID, respecting soft-delete.
func (r *EventRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Event, error) {
	var event domain.Event
	query := `
		SELECT * FROM events
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &event, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrEventNotFound
		}
		return nil, fmt.Errorf("failed to get event by id: %w", err)
	}
	return &event, nil
}

// GetByIDForTenant retrieves an event by ID ensuring it belongs to the given tenant.
func (r *EventRepository) GetByIDForTenant(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*domain.Event, error) {
	var event domain.Event
	query := `
		SELECT * FROM events
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &event, query, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrEventNotFound
		}
		return nil, fmt.Errorf("failed to get event by id for tenant: %w", err)
	}
	return &event, nil
}

// ListByTenant lists all events for a tenant with optional filtering and pagination.
func (r *EventRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, filter domain.EventFilter) ([]*domain.Event, error) {
	var events []*domain.Event

	query, args := r.buildListQuery(tenantID, filter)

	err := r.db.SelectContext(ctx, &events, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list events by tenant: %w", err)
	}
	return events, nil
}

// CountByTenant returns the total count of events matching the filter for a tenant.
func (r *EventRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID, filter domain.EventFilter) (int, error) {
	var count int

	query, args := r.buildCountQuery(tenantID, filter)

	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to count events by tenant: %w", err)
	}
	return count, nil
}

// Update updates an existing event. Only non-zero fields are updated.
func (r *EventRepository) Update(ctx context.Context, event *domain.Event) error {
	query := `
		UPDATE events SET
			name = COALESCE(NULLIF(:name, ''), name),
			type = COALESCE(NULLIF(:type, ''), type),
			description = COALESCE(:description, description),
			cover_url = COALESCE(:cover_url, cover_url),
			status = COALESCE(NULLIF(:status, ''), status),
			start_date = COALESCE(:start_date, start_date),
			end_date = COALESCE(:end_date, end_date),
			rsvp_deadline = COALESCE(:rsvp_deadline, rsvp_deadline),
			capacity = COALESCE(:capacity, capacity),
			target_invites = COALESCE(:target_invites, target_invites),
			target_attendance = COALESCE(:target_attendance, target_attendance),
			primary_location_id = COALESCE(:primary_location_id, primary_location_id),
			dress_code = COALESCE(:dress_code, dress_code),
			privacy_notice = COALESCE(:privacy_notice, privacy_notice),
			guest_policy = COALESCE(:guest_policy, guest_policy),
			settings = COALESCE(:settings, settings),
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id AND deleted_at IS NULL
	`
	result, err := r.db.NamedExecContext(ctx, query, event)
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if rows == 0 {
		return domain.ErrEventNotFound
	}

	return nil
}

// SoftDelete marks an event as deleted (soft-delete pattern).
func (r *EventRepository) SoftDelete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	query := `
		UPDATE events
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to soft-delete event: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}
	if rows == 0 {
		return domain.ErrEventNotFound
	}

	return nil
}

// UpdateStatus updates only the status field of an event.
func (r *EventRepository) UpdateStatus(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, status string) error {
	query := `
		UPDATE events
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, status, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update event status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check status update result: %w", err)
	}
	if rows == 0 {
		return domain.ErrEventNotFound
	}

	return nil
}

// buildListQuery constructs the SELECT query with filters.
func (r *EventRepository) buildListQuery(tenantID uuid.UUID, filter domain.EventFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "tenant_id = $1 AND deleted_at IS NULL")
	args = append(args, tenantID)

	argIdx := 1

	if filter.Status != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
	}

	if filter.Type != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, filter.Type)
	}

	if filter.StartFrom != nil {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("start_date >= $%d", argIdx))
		args = append(args, *filter.StartFrom)
	}

	if filter.StartTo != nil {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("start_date <= $%d", argIdx))
		args = append(args, *filter.StartTo)
	}

	query := "SELECT * FROM events WHERE " + strings.Join(conditions, " AND ") + " ORDER BY start_date DESC"

	if filter.PerPage > 0 {
		argIdx++
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.PerPage)

		if filter.Page > 0 {
			offset := (filter.Page - 1) * filter.PerPage
			argIdx++
			query += fmt.Sprintf(" OFFSET $%d", argIdx)
			args = append(args, offset)
		}
	}

	return query, args
}

// buildCountQuery constructs the COUNT query with filters.
func (r *EventRepository) buildCountQuery(tenantID uuid.UUID, filter domain.EventFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "tenant_id = $1 AND deleted_at IS NULL")
	args = append(args, tenantID)

	argIdx := 1

	if filter.Status != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
	}

	if filter.Type != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, filter.Type)
	}

	if filter.StartFrom != nil {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("start_date >= $%d", argIdx))
		args = append(args, *filter.StartFrom)
	}

	if filter.StartTo != nil {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("start_date <= $%d", argIdx))
		args = append(args, *filter.StartTo)
	}

	query := "SELECT COUNT(*) FROM events WHERE " + strings.Join(conditions, " AND ")

	return query, args
}
