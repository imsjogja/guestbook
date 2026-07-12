// Package repository provides data access layer implementations for GuestFlow.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"guestflow/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// CheckinRepository provides data access for check-in records with tenant isolation.
type CheckinRepository struct {
	db *sqlx.DB
}

// NewCheckinRepository creates a new CheckinRepository.
func NewCheckinRepository(db *sqlx.DB) *CheckinRepository {
	return &CheckinRepository{db: db}
}

// Create inserts a new check-in record into the database.
func (r *CheckinRepository) Create(ctx context.Context, checkin *domain.Checkin) error {
	query := `
		INSERT INTO checkins (
			id, tenant_id, event_id, session_id, guest_id, invitation_id, credential_id,
			method, status, device_id, gate_id, officer_id, actual_pax, adults, children,
			override_reason, approved_by, ip_address, latitude, longitude, notes, offline_synced,
			created_at, updated_at, deleted_at
		) VALUES (
			:id, :tenant_id, :event_id, :session_id, :guest_id, :invitation_id, :credential_id,
			:method, :status, :device_id, :gate_id, :officer_id, :actual_pax, :adults, :children,
			:override_reason, :approved_by, :ip_address, :latitude, :longitude, :notes, :offline_synced,
			:created_at, :updated_at, :deleted_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, checkin)
	if err != nil {
		return fmt.Errorf("create checkin: %w", err)
	}
	return nil
}

// GetByID retrieves a check-in record by its ID.
func (r *CheckinRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Checkin, error) {
	var checkin domain.Checkin
	query := `
		SELECT * FROM checkins
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &checkin, query, id, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get checkin by id: %w", err)
	}
	return &checkin, nil
}

// ListByEvent lists check-in records for an event with optional filters and pagination.
func (r *CheckinRepository) ListByEvent(ctx context.Context, params domain.CheckinListParams) ([]*domain.Checkin, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 20
	}
	offset := (params.Page - 1) * params.PerPage

	where, args := r.buildWhereClause(params)
	query := fmt.Sprintf(`
		SELECT * FROM checkins
		%s
		ORDER BY created_at DESC
		LIMIT %d OFFSET %d
	`, where, params.PerPage, offset)

	var checkins []*domain.Checkin
	err := r.db.SelectContext(ctx, &checkins, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list checkins by event: %w", err)
	}
	return checkins, nil
}

// CountByEvent returns the total count of check-ins for an event.
func (r *CheckinRepository) CountByEvent(ctx context.Context, tenantID, eventID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM checkins
		WHERE tenant_id = $1 AND event_id = $2 AND deleted_at IS NULL AND status = $3
	`
	var count int
	err := r.db.GetContext(ctx, &count, query, tenantID, eventID, domain.CheckinStatusSuccess)
	if err != nil {
		return 0, fmt.Errorf("count checkins by event: %w", err)
	}
	return count, nil
}

// CountPaxByEvent returns the total actual pax for an event.
func (r *CheckinRepository) CountPaxByEvent(ctx context.Context, tenantID, eventID uuid.UUID) (int, error) {
	query := `
		SELECT COALESCE(SUM(actual_pax), 0) FROM checkins
		WHERE tenant_id = $1 AND event_id = $2 AND deleted_at IS NULL AND status = $3
	`
	var total int
	err := r.db.GetContext(ctx, &total, query, tenantID, eventID, domain.CheckinStatusSuccess)
	if err != nil {
		return 0, fmt.Errorf("count pax by event: %w", err)
	}
	return total, nil
}

// CountWalkInsByEvent returns the total walk-in count for an event.
func (r *CheckinRepository) CountWalkInsByEvent(ctx context.Context, tenantID, eventID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM checkins
		WHERE tenant_id = $1 AND event_id = $2 AND deleted_at IS NULL
		  AND status = $3 AND method = $4
	`
	var count int
	err := r.db.GetContext(ctx, &count, query, tenantID, eventID, domain.CheckinStatusSuccess, domain.CheckinMethodWalkin)
	if err != nil {
		return 0, fmt.Errorf("count walk-ins by event: %w", err)
	}
	return count, nil
}

// CountByGate returns check-in counts per gate for an event.
func (r *CheckinRepository) CountByGate(ctx context.Context, tenantID, eventID uuid.UUID) ([]domain.GateStat, error) {
	query := `
		SELECT 
			g.id as gate_id,
			g.name as gate_name,
			COUNT(c.id) as count,
			COALESCE(SUM(c.actual_pax), 0) as pax
		FROM checkin_gates g
		LEFT JOIN checkins c ON c.gate_id = g.id 
			AND c.tenant_id = g.tenant_id 
			AND c.event_id = g.event_id
			AND c.deleted_at IS NULL
			AND c.status = $1
		WHERE g.tenant_id = $2 AND g.event_id = $3 AND g.deleted_at IS NULL
		GROUP BY g.id, g.name
		ORDER BY count DESC
	`

	var stats []domain.GateStat
	err := r.db.SelectContext(ctx, &stats, query, domain.CheckinStatusSuccess, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("count checkins by gate: %w", err)
	}
	return stats, nil
}

// CountByMethod returns check-in counts per method for an event.
func (r *CheckinRepository) CountByMethod(ctx context.Context, tenantID, eventID uuid.UUID) ([]domain.MethodStat, error) {
	query := `
		SELECT 
			method,
			COUNT(*) as count
		FROM checkins
		WHERE tenant_id = $1 AND event_id = $2 AND deleted_at IS NULL AND status = $3
		GROUP BY method
		ORDER BY count DESC
	`

	var stats []domain.MethodStat
	err := r.db.SelectContext(ctx, &stats, query, tenantID, eventID, domain.CheckinStatusSuccess)
	if err != nil {
		return nil, fmt.Errorf("count checkins by method: %w", err)
	}
	return stats, nil
}

// GetRecent returns recent check-ins for an event.
func (r *CheckinRepository) GetRecent(ctx context.Context, tenantID, eventID uuid.UUID, limit int) ([]domain.Checkin, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT * FROM checkins
		WHERE tenant_id = $1 AND event_id = $2 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $3
	`

	var checkins []domain.Checkin
	err := r.db.SelectContext(ctx, &checkins, query, tenantID, eventID, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent checkins: %w", err)
	}
	return checkins, nil
}

// IsCheckedIn checks if a guest has already checked in for an event.
func (r *CheckinRepository) IsCheckedIn(ctx context.Context, tenantID, eventID, guestID uuid.UUID) (bool, error) {
	query := `
		SELECT COUNT(*) FROM checkins
		WHERE tenant_id = $1 AND event_id = $2 AND guest_id = $3 
		  AND deleted_at IS NULL AND status = $4
	`
	var count int
	err := r.db.GetContext(ctx, &count, query, tenantID, eventID, guestID, domain.CheckinStatusSuccess)
	if err != nil {
		return false, fmt.Errorf("check if guest is checked in: %w", err)
	}
	return count > 0, nil
}

// Update updates a check-in record (e.g., for offline sync).
func (r *CheckinRepository) Update(ctx context.Context, checkin *domain.Checkin) error {
	query := `
		UPDATE checkins SET
			event_id = :event_id,
			session_id = :session_id,
			guest_id = :guest_id,
			invitation_id = :invitation_id,
			credential_id = :credential_id,
			method = :method,
			status = :status,
			device_id = :device_id,
			gate_id = :gate_id,
			officer_id = :officer_id,
			actual_pax = :actual_pax,
			adults = :adults,
			children = :children,
			override_reason = :override_reason,
			approved_by = :approved_by,
			ip_address = :ip_address,
			latitude = :latitude,
			longitude = :longitude,
			notes = :notes,
			offline_synced = :offline_synced,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id AND deleted_at IS NULL
	`
	result, err := r.db.NamedExecContext(ctx, query, checkin)
	if err != nil {
		return fmt.Errorf("update checkin: %w", err)
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

// FindDuplicateCheckin finds an existing successful checkin for the same guest and event.
func (r *CheckinRepository) FindDuplicateCheckin(ctx context.Context, tenantID, eventID, guestID uuid.UUID) (*domain.Checkin, error) {
	var checkin domain.Checkin
	query := `
		SELECT * FROM checkins
		WHERE tenant_id = $1 AND event_id = $2 AND guest_id = $3 
		  AND deleted_at IS NULL AND status = $4
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &checkin, query, tenantID, eventID, guestID, domain.CheckinStatusSuccess)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find duplicate checkin: %w", err)
	}
	return &checkin, nil
}

// GetPeakHour returns the hour with the most check-ins for an event.
func (r *CheckinRepository) GetPeakHour(ctx context.Context, tenantID, eventID uuid.UUID) (string, error) {
	query := `
		SELECT TO_CHAR(DATE_TRUNC('hour', created_at), 'HH24:MI') as peak_hour
		FROM checkins
		WHERE tenant_id = $1 AND event_id = $2 AND deleted_at IS NULL AND status = $3
		GROUP BY DATE_TRUNC('hour', created_at)
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`
	var peakHour string
	err := r.db.GetContext(ctx, &peakHour, query, tenantID, eventID, domain.CheckinStatusSuccess)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("get peak hour: %w", err)
	}
	return peakHour, nil
}

// SoftDelete marks a check-in record as deleted.
func (r *CheckinRepository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	now := time.Now().UTC()
	query := `
		UPDATE checkins
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, now, id, tenantID)
	if err != nil {
		return fmt.Errorf("soft delete checkin: %w", err)
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

// buildWhereClause constructs the WHERE clause for check-in list queries.
func (r *CheckinRepository) buildWhereClause(params domain.CheckinListParams) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	// Tenant isolation + event scope + soft delete
	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, params.TenantID)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("event_id = $%d", argIdx))
	args = append(args, params.EventID)
	argIdx++

	conditions = append(conditions, "deleted_at IS NULL")

	// Filter by method
	if strings.TrimSpace(params.Method) != "" {
		conditions = append(conditions, fmt.Sprintf("method = $%d", argIdx))
		args = append(args, params.Method)
		argIdx++
	}

	// Filter by status
	if strings.TrimSpace(params.Status) != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, params.Status)
		argIdx++
	}

	// Filter by gate
	if params.GateID != nil {
		conditions = append(conditions, fmt.Sprintf("gate_id = $%d", argIdx))
		args = append(args, *params.GateID)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")
	return where, args
}
