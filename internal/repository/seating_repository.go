// Package repository provides data access layer implementations for GuestFlow.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"guestflow/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SeatingRepository provides data access for seating management with tenant isolation.
type SeatingRepository struct {
	db *sqlx.DB
}

// NewSeatingRepository creates a new SeatingRepository.
func NewSeatingRepository(db *sqlx.DB) *SeatingRepository {
	return &SeatingRepository{db: db}
}

// ─── Venue Zone Operations ────────────────────────────────────────────────────

// CreateZone inserts a new venue zone.
func (r *SeatingRepository) CreateZone(ctx context.Context, zone *domain.VenueZone) error {
	query := `
		INSERT INTO venue_zones (
			id, tenant_id, event_id, name, description, sort_order,
			created_at, updated_at, deleted_at
		) VALUES (
			:id, :tenant_id, :event_id, :name, :description, :sort_order,
			:created_at, :updated_at, :deleted_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, zone)
	if err != nil {
		return fmt.Errorf("create venue zone: %w", err)
	}
	return nil
}

// GetZone retrieves a venue zone by ID.
func (r *SeatingRepository) GetZone(ctx context.Context, tenantID, id uuid.UUID) (*domain.VenueZone, error) {
	var zone domain.VenueZone
	query := `
		SELECT * FROM venue_zones
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &zone, query, id, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get venue zone: %w", err)
	}
	return &zone, nil
}

// ListZonesByEvent lists all venue zones for an event.
func (r *SeatingRepository) ListZonesByEvent(ctx context.Context, tenantID, eventID uuid.UUID) ([]domain.VenueZone, error) {
	query := `
		SELECT * FROM venue_zones
		WHERE tenant_id = $1 AND event_id = $2 AND deleted_at IS NULL
		ORDER BY sort_order, name
	`
	var zones []domain.VenueZone
	err := r.db.SelectContext(ctx, &zones, query, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("list venue zones by event: %w", err)
	}
	return zones, nil
}

// SoftDeleteZone marks a venue zone as deleted.
func (r *SeatingRepository) SoftDeleteZone(ctx context.Context, tenantID, id uuid.UUID) error {
	now := time.Now().UTC()
	query := `
		UPDATE venue_zones
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, now, id, tenantID)
	if err != nil {
		return fmt.Errorf("soft delete venue zone: %w", err)
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

// ─── Table Operations ─────────────────────────────────────────────────────────

// CreateTable inserts a new table.
func (r *SeatingRepository) CreateTable(ctx context.Context, table *domain.Table) error {
	query := `
		INSERT INTO tables (
			id, tenant_id, event_id, zone_id, name, capacity, shape,
			position_x, position_y, is_locked, accessibility, vip_only, notes,
			created_at, updated_at, deleted_at
		) VALUES (
			:id, :tenant_id, :event_id, :zone_id, :name, :capacity, :shape,
			:position_x, :position_y, :is_locked, :accessibility, :vip_only, :notes,
			:created_at, :updated_at, :deleted_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, table)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	return nil
}

// GetTable retrieves a table by ID with tenant isolation.
func (r *SeatingRepository) GetTable(ctx context.Context, tenantID, eventID, id uuid.UUID) (*domain.Table, error) {
	var table domain.Table
	query := `
		SELECT * FROM tables
		WHERE id = $1 AND tenant_id = $2 AND event_id = $3 AND deleted_at IS NULL
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &table, query, id, tenantID, eventID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get table: %w", err)
	}
	return &table, nil
}

// UpdateTable updates an existing table.
func (r *SeatingRepository) UpdateTable(ctx context.Context, table *domain.Table) error {
	query := `
		UPDATE tables SET
			zone_id = COALESCE(:zone_id, zone_id),
			name = COALESCE(NULLIF(:name, ''), name),
			capacity = COALESCE(NULLIF(:capacity, 0), capacity),
			shape = COALESCE(NULLIF(:shape, ''), shape),
			position_x = COALESCE(:position_x, position_x),
			position_y = COALESCE(:position_y, position_y),
			is_locked = :is_locked,
			accessibility = :accessibility,
			vip_only = :vip_only,
			notes = COALESCE(:notes, notes),
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id AND deleted_at IS NULL
	`
	result, err := r.db.NamedExecContext(ctx, query, table)
	if err != nil {
		return fmt.Errorf("update table: %w", err)
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

// SoftDeleteTable marks a table as deleted.
func (r *SeatingRepository) SoftDeleteTable(ctx context.Context, tenantID, eventID, id uuid.UUID) error {
	return RunInTransaction(ctx, r.db, func(tx *sqlx.Tx) error {
		now := time.Now().UTC()

		// Delete all seat assignments for this table first
		_, err := tx.ExecContext(ctx, `
			DELETE FROM seat_assignments
			WHERE table_id = $1
		`, id)
		if err != nil {
			return fmt.Errorf("delete seat assignments for table: %w", err)
		}

		// Soft delete the table
		query := `
			UPDATE tables
			SET deleted_at = $1, updated_at = $1
			WHERE id = $2 AND tenant_id = $3 AND event_id = $4 AND deleted_at IS NULL
		`
		result, err := tx.ExecContext(ctx, query, now, id, tenantID, eventID)
		if err != nil {
			return fmt.Errorf("soft delete table: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("check delete result: %w", err)
		}
		if rowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// ListTablesByEvent lists all tables for an event with optional zone info.
func (r *SeatingRepository) ListTablesByEvent(ctx context.Context, tenantID, eventID uuid.UUID) ([]domain.TableWithOccupancy, error) {
	query := `
		SELECT 
			t.*,
			COALESCE(sa.occupancy, 0) as occupancy
		FROM tables t
		LEFT JOIN (
			SELECT table_id, COUNT(*) as occupancy
			FROM seat_assignments
			GROUP BY table_id
		) sa ON sa.table_id = t.id
		WHERE t.tenant_id = $1 AND t.event_id = $2 AND t.deleted_at IS NULL
		ORDER BY t.name
	`

	var tables []domain.TableWithOccupancy
	err := r.db.SelectContext(ctx, &tables, query, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("list tables by event: %w", err)
	}
	return tables, nil
}

// GetTableWithOccupancy retrieves a single table with its occupancy count.
func (r *SeatingRepository) GetTableWithOccupancy(ctx context.Context, tenantID, eventID, id uuid.UUID) (*domain.TableWithOccupancy, error) {
	query := `
		SELECT 
			t.*,
			COALESCE(sa.occupancy, 0) as occupancy
		FROM tables t
		LEFT JOIN (
			SELECT table_id, COUNT(*) as occupancy
			FROM seat_assignments
			GROUP BY table_id
		) sa ON sa.table_id = t.id
		WHERE t.id = $1 AND t.tenant_id = $2 AND t.event_id = $3 AND t.deleted_at IS NULL
		LIMIT 1
	`

	var table domain.TableWithOccupancy
	err := r.db.GetContext(ctx, &table, query, id, tenantID, eventID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get table with occupancy: %w", err)
	}
	return &table, nil
}

// ─── Seat Assignment Operations ───────────────────────────────────────────────

// AssignGuest assigns a guest to a table.
func (r *SeatingRepository) AssignGuest(ctx context.Context, assignment *domain.SeatAssignment) error {
	query := `
		INSERT INTO seat_assignments (
			table_id, guest_id, seat_number, assigned_by, assigned_at
		) VALUES (
			:table_id, :guest_id, :seat_number, :assigned_by, :assigned_at
		)
		ON CONFLICT (table_id, guest_id) DO UPDATE SET
			seat_number = EXCLUDED.seat_number,
			assigned_by = EXCLUDED.assigned_by,
			assigned_at = EXCLUDED.assigned_at
	`
	_, err := r.db.NamedExecContext(ctx, query, assignment)
	if err != nil {
		return fmt.Errorf("assign guest to table: %w", err)
	}
	return nil
}

// UnassignGuest removes a guest's seat assignment from a table.
func (r *SeatingRepository) UnassignGuest(ctx context.Context, tableID, guestID uuid.UUID) error {
	query := `
		DELETE FROM seat_assignments
		WHERE table_id = $1 AND guest_id = $2
	`
	result, err := r.db.ExecContext(ctx, query, tableID, guestID)
	if err != nil {
		return fmt.Errorf("unassign guest from table: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check unassign result: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListAssignmentsByTable lists all seat assignments for a table.
func (r *SeatingRepository) ListAssignmentsByTable(ctx context.Context, tableID uuid.UUID) ([]domain.AssignedGuest, error) {
	query := `
		SELECT 
			sa.guest_id,
			g.full_name,
			g.guest_type,
			sa.seat_number,
			sa.assigned_at
		FROM seat_assignments sa
		JOIN guests g ON g.id = sa.guest_id AND g.deleted_at IS NULL
		WHERE sa.table_id = $1
		ORDER BY sa.seat_number, g.full_name
	`

	var guests []domain.AssignedGuest
	err := r.db.SelectContext(ctx, &guests, query, tableID)
	if err != nil {
		return nil, fmt.Errorf("list assignments by table: %w", err)
	}
	return guests, nil
}

// ListAssignmentsByEvent lists all seat assignments for an event.
func (r *SeatingRepository) ListAssignmentsByEvent(ctx context.Context, tenantID, eventID uuid.UUID) ([]domain.SeatAssignment, error) {
	query := `
		SELECT sa.*
		FROM seat_assignments sa
		JOIN tables t ON t.id = sa.table_id AND t.deleted_at IS NULL
		WHERE t.tenant_id = $1 AND t.event_id = $2
		ORDER BY sa.table_id, sa.seat_number
	`

	var assignments []domain.SeatAssignment
	err := r.db.SelectContext(ctx, &assignments, query, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("list assignments by event: %w", err)
	}
	return assignments, nil
}

// CountAssignmentsByTable returns the number of guests assigned to a table.
func (r *SeatingRepository) CountAssignmentsByTable(ctx context.Context, tableID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM seat_assignments
		WHERE table_id = $1
	`
	var count int
	err := r.db.GetContext(ctx, &count, query, tableID)
	if err != nil {
		return 0, fmt.Errorf("count assignments by table: %w", err)
	}
	return count, nil
}

// GetGuestAssignment retrieves a guest's current table assignment.
func (r *SeatingRepository) GetGuestAssignment(ctx context.Context, guestID uuid.UUID) (*domain.SeatAssignment, error) {
	var assignment domain.SeatAssignment
	query := `
		SELECT * FROM seat_assignments
		WHERE guest_id = $1
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &assignment, query, guestID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get guest assignment: %w", err)
	}
	return &assignment, nil
}

// IsGuestAssigned checks if a guest is already assigned to any table.
func (r *SeatingRepository) IsGuestAssigned(ctx context.Context, guestID uuid.UUID) (bool, error) {
	query := `
		SELECT COUNT(*) FROM seat_assignments
		WHERE guest_id = $1
	`
	var count int
	err := r.db.GetContext(ctx, &count, query, guestID)
	if err != nil {
		return false, fmt.Errorf("check if guest is assigned: %w", err)
	}
	return count > 0, nil
}

// CountUnassignedGuests returns the number of checked-in guests not assigned to any table.
func (r *SeatingRepository) CountUnassignedGuests(ctx context.Context, tenantID, eventID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM checkins c
		LEFT JOIN seat_assignments sa ON sa.guest_id = c.guest_id
		WHERE c.tenant_id = $1 AND c.event_id = $2 
		  AND c.deleted_at IS NULL AND c.status = $3
		  AND sa.guest_id IS NULL
	`
	var count int
	err := r.db.GetContext(ctx, &count, query, tenantID, eventID, domain.CheckinStatusSuccess)
	if err != nil {
		return 0, fmt.Errorf("count unassigned guests: %w", err)
	}
	return count, nil
}
