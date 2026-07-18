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

// EventMemberRepository manages staff assignments within an event.
type EventMemberRepository struct {
	db *sqlx.DB
}

func NewEventMemberRepository(db *sqlx.DB) *EventMemberRepository {
	return &EventMemberRepository{db: db}
}

func (r *EventMemberRepository) Create(ctx context.Context, member *domain.EventMember) error {
	query := `
		INSERT INTO event_members (
			id, tenant_id, event_id, user_id, role, status, invited_by,
			assigned_at, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :event_id, :user_id, :role, :status, :invited_by,
			:assigned_at, :created_at, :updated_at
		)
		ON CONFLICT (event_id, user_id) DO UPDATE SET
			role = EXCLUDED.role,
			status = 'active',
			invited_by = EXCLUDED.invited_by,
			assigned_at = NOW(),
			updated_at = NOW()`
	if _, err := r.db.NamedExecContext(ctx, query, member); err != nil {
		return fmt.Errorf("create event member: %w", err)
	}
	return nil
}

func (r *EventMemberRepository) Get(ctx context.Context, tenantID, eventID, userID uuid.UUID) (*domain.EventMember, error) {
	var member domain.EventMember
	query := `
		SELECT * FROM event_members
		WHERE tenant_id = $1 AND event_id = $2 AND user_id = $3 AND status = $4
		LIMIT 1
	`
	if err := r.db.GetContext(ctx, &member, query, tenantID, eventID, userID, domain.EventMemberStatusActive); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMembershipNotFound
		}
		return nil, fmt.Errorf("get event member: %w", err)
	}
	return &member, nil
}

func (r *EventMemberRepository) ListByEvent(ctx context.Context, tenantID, eventID uuid.UUID) ([]*domain.EventMember, error) {
	members := make([]*domain.EventMember, 0)
	query := `
		SELECT * FROM event_members
		WHERE tenant_id = $1 AND event_id = $2
		ORDER BY created_at ASC
	`
	if err := r.db.SelectContext(ctx, &members, query, tenantID, eventID); err != nil {
		return nil, fmt.Errorf("list event members: %w", err)
	}
	return members, nil
}

func (r *EventMemberRepository) UpdateRole(ctx context.Context, tenantID, eventID, userID uuid.UUID, role string) error {
	query := `
		UPDATE event_members
		SET role = $1, status = $2, updated_at = NOW()
		WHERE tenant_id = $3 AND event_id = $4 AND user_id = $5
	`
	result, err := r.db.ExecContext(ctx, query, role, domain.EventMemberStatusActive, tenantID, eventID, userID)
	if err != nil {
		return fmt.Errorf("update event member role: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check event member update: %w", err)
	}
	if rows == 0 {
		return domain.ErrMembershipNotFound
	}
	return nil
}

func (r *EventMemberRepository) Deactivate(ctx context.Context, tenantID, eventID, userID uuid.UUID) error {
	query := `
		UPDATE event_members
		SET status = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND event_id = $3 AND user_id = $4 AND status <> $1
	`
	result, err := r.db.ExecContext(ctx, query, domain.EventMemberStatusInactive, tenantID, eventID, userID)
	if err != nil {
		return fmt.Errorf("deactivate event member: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check event member deactivation: %w", err)
	}
	if rows == 0 {
		return domain.ErrMembershipNotFound
	}
	return nil
}

// ListEventsByUser returns events where a user has an active event assignment.
func (r *EventMemberRepository) ListEventsByUser(ctx context.Context, tenantID, userID uuid.UUID, filter domain.EventFilter) ([]*domain.Event, int, error) {
	conditions := []string{
		"e.tenant_id = $1",
		"em.tenant_id = $1",
		"em.user_id = $2",
		"em.status = 'active'",
		"e.deleted_at IS NULL",
	}
	args := []interface{}{tenantID, userID}
	argIndex := 2

	if filter.Status != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("e.status = $%d", argIndex))
		args = append(args, filter.Status)
	}
	if filter.Type != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("e.type = $%d", argIndex))
		args = append(args, filter.Type)
	}
	if filter.StartFrom != nil {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("e.start_date >= $%d", argIndex))
		args = append(args, *filter.StartFrom)
	}
	if filter.StartTo != nil {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("e.start_date <= $%d", argIndex))
		args = append(args, *filter.StartTo)
	}

	where := strings.Join(conditions, " AND ")
	var total int
	countQuery := "SELECT COUNT(DISTINCT e.id) FROM events e JOIN event_members em ON em.event_id = e.id WHERE " + where
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count accessible events: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 {
		perPage = 20
	}
	argIndex++
	limitArg := argIndex
	args = append(args, perPage)
	argIndex++
	offsetArg := argIndex
	args = append(args, (page-1)*perPage)
	query := fmt.Sprintf(
		"SELECT DISTINCT e.*, COALESCE((SELECT COUNT(*) FROM event_guests eg WHERE eg.event_id = e.id AND eg.tenant_id = e.tenant_id AND eg.status = 'active' AND eg.deleted_at IS NULL), 0) AS guest_count FROM events e JOIN event_members em ON em.event_id = e.id WHERE %s ORDER BY e.start_date DESC LIMIT $%d OFFSET $%d",
		where, limitArg, offsetArg,
	)

	events := make([]*domain.Event, 0)
	if err := r.db.SelectContext(ctx, &events, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list accessible events: %w", err)
	}
	return events, total, nil
}
