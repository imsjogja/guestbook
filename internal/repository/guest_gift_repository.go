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

type GuestGiftRepository struct {
	db *sqlx.DB
}

func NewGuestGiftRepository(db *sqlx.DB) *GuestGiftRepository {
	return &GuestGiftRepository{db: db}
}

func (r *GuestGiftRepository) ListByEvent(ctx context.Context, tenantID, eventID uuid.UUID) ([]*domain.GuestGift, error) {
	items := make([]*domain.GuestGift, 0)
	if err := r.db.SelectContext(ctx, &items, `
		SELECT id, tenant_id, event_id, guest_id, event_guest_id, amount,
		       gift_type, notes, received_at, recorded_by, created_at, updated_at, deleted_at
		FROM guest_gifts
		WHERE tenant_id = $1 AND event_id = $2 AND deleted_at IS NULL
		ORDER BY received_at DESC, created_at DESC`, tenantID, eventID); err != nil {
		return nil, fmt.Errorf("list guest gifts: %w", err)
	}
	return items, nil
}

func (r *GuestGiftRepository) GetByEventAndGuest(ctx context.Context, tenantID, eventID, guestID uuid.UUID) (*domain.GuestGift, error) {
	var item domain.GuestGift
	err := r.db.GetContext(ctx, &item, `
		SELECT id, tenant_id, event_id, guest_id, event_guest_id, amount,
		       gift_type, notes, received_at, recorded_by, created_at, updated_at, deleted_at
		FROM guest_gifts
		WHERE tenant_id = $1 AND event_id = $2 AND guest_id = $3 AND deleted_at IS NULL`,
		tenantID, eventID, guestID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get guest gift: %w", err)
	}
	return &item, nil
}

func (r *GuestGiftRepository) Upsert(ctx context.Context, item *domain.GuestGift) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO guest_gifts (
			id, tenant_id, event_id, guest_id, event_guest_id, amount,
			gift_type, notes, received_at, recorded_by, created_at, updated_at, deleted_at
		) VALUES (
			:id, :tenant_id, :event_id, :guest_id, :event_guest_id, :amount,
			:gift_type, :notes, :received_at, :recorded_by, :created_at, :updated_at, NULL
		)
		ON CONFLICT (tenant_id, event_id, guest_id) WHERE deleted_at IS NULL DO UPDATE SET
			event_guest_id = EXCLUDED.event_guest_id,
			amount = EXCLUDED.amount,
			gift_type = EXCLUDED.gift_type,
			notes = EXCLUDED.notes,
			received_at = EXCLUDED.received_at,
			recorded_by = EXCLUDED.recorded_by,
			updated_at = EXCLUDED.updated_at,
			deleted_at = NULL`, item)
	if err != nil {
		return fmt.Errorf("upsert guest gift: %w", err)
	}
	return nil
}

func (r *GuestGiftRepository) Delete(ctx context.Context, tenantID, eventID, guestID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE guest_gifts
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE tenant_id = $1 AND event_id = $2 AND guest_id = $3 AND deleted_at IS NULL`,
		tenantID, eventID, guestID)
	if err != nil {
		return fmt.Errorf("delete guest gift: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("check deleted guest gift: %w", err)
	} else if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
