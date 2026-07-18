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

type EventGuestRepository struct {
	db *sqlx.DB
}

func NewEventGuestRepository(db *sqlx.DB) *EventGuestRepository {
	return &EventGuestRepository{db: db}
}

func (r *EventGuestRepository) Create(ctx context.Context, eventGuest *domain.EventGuest) error {
	const query = `
		INSERT INTO event_guests (
			id, tenant_id, event_id, guest_id, status, source, max_pax,
			adults, children, plus_one_allowed, notes, created_by, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :event_id, :guest_id, :status, :source, :max_pax,
			:adults, :children, :plus_one_allowed, :notes, :created_by, :created_at, :updated_at
		)`
	_, err := r.db.NamedExecContext(ctx, query, eventGuest)
	if err != nil {
		return fmt.Errorf("create event guest: %w", err)
	}
	return nil
}

func (r *EventGuestRepository) GetByID(ctx context.Context, tenantID, eventID, id uuid.UUID) (*domain.EventGuest, error) {
	row := eventGuestRow{}
	err := r.db.GetContext(ctx, &row, eventGuestSelect+`
		WHERE eg.id = $1 AND eg.tenant_id = $2 AND eg.event_id = $3 AND eg.deleted_at IS NULL`, id, tenantID, eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get event guest: %w", err)
	}
	return row.toDomain(), nil
}

func (r *EventGuestRepository) GetByEventAndGuest(ctx context.Context, tenantID, eventID, guestID uuid.UUID) (*domain.EventGuest, error) {
	row := eventGuestRow{}
	err := r.db.GetContext(ctx, &row, eventGuestSelect+`
		WHERE eg.tenant_id = $1 AND eg.event_id = $2 AND eg.guest_id = $3 AND eg.deleted_at IS NULL`, tenantID, eventID, guestID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get event guest by guest: %w", err)
	}
	return row.toDomain(), nil
}

func (r *EventGuestRepository) List(ctx context.Context, params domain.EventGuestListParams) ([]*domain.EventGuest, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 20
	}
	if params.PerPage > 100 {
		params.PerPage = 100
	}

	where, args := eventGuestWhere(params)
	query := eventGuestSelect + "\n" + where + fmt.Sprintf("\nORDER BY eg.created_at DESC LIMIT %d OFFSET %d", params.PerPage, (params.Page-1)*params.PerPage)
	var rows []eventGuestRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list event guests: %w", err)
	}
	result := make([]*domain.EventGuest, 0, len(rows))
	for i := range rows {
		result = append(result, rows[i].toDomain())
	}
	return result, nil
}

func (r *EventGuestRepository) Count(ctx context.Context, params domain.EventGuestListParams) (int, error) {
	where, args := eventGuestWhere(params)
	var count int
	if err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM event_guests eg JOIN guests g ON g.id = eg.guest_id AND g.deleted_at IS NULL\n"+where, args...); err != nil {
		return 0, fmt.Errorf("count event guests: %w", err)
	}
	return count, nil
}

func (r *EventGuestRepository) Cancel(ctx context.Context, tenantID, eventID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE event_guests SET status = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND event_id = $4 AND deleted_at IS NULL`,
		domain.EventGuestStatusCancelled, id, tenantID, eventID)
	if err != nil {
		return fmt.Errorf("cancel event guest: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("check cancelled event guest: %w", err)
	} else if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const eventGuestSelect = `SELECT
	eg.id, eg.tenant_id, eg.event_id, eg.guest_id, eg.status, eg.source,
	eg.max_pax, eg.adults, eg.children, eg.plus_one_allowed, eg.notes,
	eg.created_by, eg.created_at, eg.updated_at, eg.deleted_at,
	g.full_name AS guest_full_name, g.nickname AS guest_nickname, g.phone AS guest_phone,
	g.email AS guest_email, g.guest_type AS guest_guest_type, g.segment AS guest_segment,
	g.is_active AS guest_is_active
FROM event_guests eg
JOIN guests g ON g.id = eg.guest_id AND g.tenant_id = eg.tenant_id`

type eventGuestRow struct {
	domain.EventGuest
	GuestFullName string  `db:"guest_full_name"`
	GuestNickname *string `db:"guest_nickname"`
	GuestPhone    *string `db:"guest_phone"`
	GuestEmail    *string `db:"guest_email"`
	GuestType     string  `db:"guest_guest_type"`
	GuestSegment  *string `db:"guest_segment"`
	GuestIsActive bool    `db:"guest_is_active"`
}

func (r eventGuestRow) toDomain() *domain.EventGuest {
	result := r.EventGuest
	result.Guest = &domain.Guest{
		TenantBase: domain.TenantBase{Base: domain.Base{ID: result.GuestID}, TenantID: result.TenantID},
		FullName:   r.GuestFullName, Nickname: r.GuestNickname, Phone: r.GuestPhone,
		Email: r.GuestEmail, GuestType: r.GuestType, Segment: r.GuestSegment, IsActive: r.GuestIsActive,
	}
	return &result
}

func eventGuestWhere(params domain.EventGuestListParams) (string, []interface{}) {
	conditions := []string{"eg.tenant_id = $1", "eg.event_id = $2", "eg.deleted_at IS NULL"}
	args := []interface{}{params.TenantID, params.EventID}
	next := 3
	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("eg.status = $%d", next))
		args = append(args, params.Status)
		next++
	}
	if search := strings.TrimSpace(params.Search); search != "" {
		conditions = append(conditions, fmt.Sprintf("(g.full_name ILIKE $%d OR g.phone ILIKE $%d OR g.email ILIKE $%d)", next, next, next))
		args = append(args, "%"+search+"%")
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}
