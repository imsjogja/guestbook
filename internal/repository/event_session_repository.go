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

// EventSessionRepository provides data access for event sessions.
type EventSessionRepository struct {
	db *sqlx.DB
}

// NewEventSessionRepository creates a new EventSessionRepository.
func NewEventSessionRepository(db *sqlx.DB) *EventSessionRepository {
	return &EventSessionRepository{db: db}
}

// Create inserts a new event session into the database.
func (r *EventSessionRepository) Create(ctx context.Context, session *domain.EventSession) error {
	query := `
		INSERT INTO event_sessions (
			id, event_id, name, description, start_time, end_time,
			location_id, capacity, sort_order,
			created_at, updated_at
		) VALUES (
			:id, :event_id, :name, :description, :start_time, :end_time,
			:location_id, :capacity, :sort_order,
			:created_at, :updated_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, session)
	if err != nil {
		return fmt.Errorf("failed to create event session: %w", err)
	}
	return nil
}

// GetByID retrieves an event session by its ID.
func (r *EventSessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.EventSession, error) {
	var session domain.EventSession
	query := `
		SELECT * FROM event_sessions
		WHERE id = $1
	`
	err := r.db.GetContext(ctx, &session, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrEventSessionNotFound
		}
		return nil, fmt.Errorf("failed to get event session by id: %w", err)
	}
	return &session, nil
}

// GetByIDForEvent retrieves a session by ID ensuring it belongs to the given event.
func (r *EventSessionRepository) GetByIDForEvent(ctx context.Context, id uuid.UUID, eventID uuid.UUID) (*domain.EventSession, error) {
	var session domain.EventSession
	query := `
		SELECT * FROM event_sessions
		WHERE id = $1 AND event_id = $2
	`
	err := r.db.GetContext(ctx, &session, query, id, eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrEventSessionNotFound
		}
		return nil, fmt.Errorf("failed to get event session by id for event: %w", err)
	}
	return &session, nil
}

// ListByEvent lists all sessions for a given event.
func (r *EventSessionRepository) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*domain.EventSession, error) {
	var sessions []*domain.EventSession
	query := `
		SELECT * FROM event_sessions
		WHERE event_id = $1
		ORDER BY sort_order ASC, start_time ASC
	`
	err := r.db.SelectContext(ctx, &sessions, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list event sessions by event: %w", err)
	}
	return sessions, nil
}

// Update updates an existing event session.
func (r *EventSessionRepository) Update(ctx context.Context, session *domain.EventSession) error {
	query := `
		UPDATE event_sessions SET
			name = COALESCE(NULLIF(:name, ''), name),
			description = COALESCE(:description, description),
			start_time = COALESCE(:start_time, start_time),
			end_time = COALESCE(:end_time, end_time),
			location_id = COALESCE(:location_id, location_id),
			capacity = COALESCE(:capacity, capacity),
			sort_order = COALESCE(:sort_order, sort_order),
			updated_at = :updated_at
		WHERE id = :id
	`
	result, err := r.db.NamedExecContext(ctx, query, session)
	if err != nil {
		return fmt.Errorf("failed to update event session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if rows == 0 {
		return domain.ErrEventSessionNotFound
	}

	return nil
}

// Delete permanently deletes an event session.
func (r *EventSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM event_sessions
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete event session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}
	if rows == 0 {
		return domain.ErrEventSessionNotFound
	}

	return nil
}

// CountByEvent returns the number of sessions for a given event.
func (r *EventSessionRepository) CountByEvent(ctx context.Context, eventID uuid.UUID) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM event_sessions
		WHERE event_id = $1
	`
	err := r.db.GetContext(ctx, &count, query, eventID)
	if err != nil {
		return 0, fmt.Errorf("failed to count event sessions by event: %w", err)
	}
	return count, nil
}
