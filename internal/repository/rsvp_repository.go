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

// RSVPRepository provides data access for RSVP responses with tenant isolation.
type RSVPRepository struct {
	db *sqlx.DB
}

// NewRSVPRepository creates a new RSVPRepository.
func NewRSVPRepository(db *sqlx.DB) *RSVPRepository {
	return &RSVPRepository{db: db}
}

// Create inserts a new RSVP response into the database.
func (r *RSVPRepository) Create(ctx context.Context, rsvp *domain.RSVPResponse) error {
	query := `
		INSERT INTO rsvp_responses (
			id, tenant_id, event_id, invitation_id, guest_id, event_guest_id,
			status, attending_pax, adults, children,
			menu_choice, allergies, accessibility_needs,
			transportation, notes,
			responded_at, edited_at, edited_by, ip_address,
			created_at, updated_at
		) VALUES (
			:id, :tenant_id, :event_id, :invitation_id, :guest_id, :event_guest_id,
			:status, :attending_pax, :adults, :children,
			:menu_choice, :allergies, :accessibility_needs,
			:transportation, :notes,
			:responded_at, :edited_at, :edited_by, :ip_address,
			:created_at, :updated_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, rsvp)
	if err != nil {
		return fmt.Errorf("create rsvp: %w", err)
	}
	return nil
}

// GetByID retrieves an RSVP by its ID.
func (r *RSVPRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.RSVPResponse, error) {
	var rsvp domain.RSVPResponse
	query := `
		SELECT * FROM rsvp_responses
		WHERE id = $1
	`
	err := r.db.GetContext(ctx, &rsvp, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRSVPNotFound
		}
		return nil, fmt.Errorf("get rsvp by id: %w", err)
	}
	return &rsvp, nil
}

// GetByIDForTenant retrieves an RSVP by ID with tenant isolation.
func (r *RSVPRepository) GetByIDForTenant(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*domain.RSVPResponse, error) {
	var rsvp domain.RSVPResponse
	query := `
		SELECT * FROM rsvp_responses
		WHERE id = $1 AND tenant_id = $2
	`
	err := r.db.GetContext(ctx, &rsvp, query, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRSVPNotFound
		}
		return nil, fmt.Errorf("get rsvp by id for tenant: %w", err)
	}
	return &rsvp, nil
}

// GetByInvitation retrieves an RSVP for a specific invitation.
func (r *RSVPRepository) GetByInvitation(ctx context.Context, invitationID uuid.UUID) (*domain.RSVPResponse, error) {
	var rsvp domain.RSVPResponse
	query := `
		SELECT * FROM rsvp_responses
		WHERE invitation_id = $1
	`
	err := r.db.GetContext(ctx, &rsvp, query, invitationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRSVPNotFound
		}
		return nil, fmt.Errorf("get rsvp by invitation: %w", err)
	}
	return &rsvp, nil
}

// GetByEvent lists RSVPs for an event with pagination and optional status filter.
func (r *RSVPRepository) GetByEvent(ctx context.Context, tenantID, eventID uuid.UUID, status string, page, perPage int) ([]*domain.RSVPResponseWithGuest, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("r.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("r.event_id = $%d", argIdx))
	args = append(args, eventID)
	argIdx++

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("r.status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	query := `
		SELECT
			r.*,
			g.full_name as guest_full_name,
			g.email as guest_email,
			g.phone as guest_phone,
			g.guest_type
		FROM rsvp_responses r
		JOIN guests g ON r.guest_id = g.id
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY r.responded_at DESC NULLS LAST, r.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1) + `
	`
	args = append(args, perPage, offset)

	var rsvps []*domain.RSVPResponseWithGuest
	err := r.db.SelectContext(ctx, &rsvps, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list rsvps by event: %w", err)
	}
	return rsvps, nil
}

// CountByEvent counts RSVPs for an event matching the status filter.
func (r *RSVPRepository) CountByEvent(ctx context.Context, tenantID, eventID uuid.UUID, status string) (int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("event_id = $%d", argIdx))
	args = append(args, eventID)
	argIdx++

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	query := `
		SELECT COUNT(*) FROM rsvp_responses
		WHERE ` + strings.Join(conditions, " AND ")

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("count rsvps by event: %w", err)
	}
	return count, nil
}

// Update updates an existing RSVP response.
func (r *RSVPRepository) Update(ctx context.Context, rsvp *domain.RSVPResponse) error {
	query := `
		UPDATE rsvp_responses SET
			status = :status,
			attending_pax = :attending_pax,
			adults = :adults,
			children = :children,
			menu_choice = :menu_choice,
			allergies = :allergies,
			accessibility_needs = :accessibility_needs,
			transportation = :transportation,
			notes = :notes,
			edited_at = :edited_at,
			edited_by = :edited_by,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id
	`
	result, err := r.db.NamedExecContext(ctx, query, rsvp)
	if err != nil {
		return fmt.Errorf("update rsvp: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check update result: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrRSVPNotFound
	}

	return nil
}

// GetDashboardStats returns aggregated RSVP statistics for the dashboard.
func (r *RSVPRepository) GetDashboardStats(ctx context.Context, tenantID, eventID uuid.UUID, eventCapacity *int) (*domain.RSVPDashboard, error) {
	dashboard := &domain.RSVPDashboard{
		CapacityTotal: 0,
	}

	if eventCapacity != nil {
		dashboard.CapacityTotal = *eventCapacity
	}

	// Count the active event roster. Invitations are a delivery artifact and
	// should not determine how many guests belong to the event.
	var totalInvited int
	query := `
		SELECT COUNT(*) FROM event_guests
		WHERE tenant_id = $1 AND event_id = $2 AND deleted_at IS NULL
		  AND status = 'active'
	`
	err := r.db.GetContext(ctx, &totalInvited, query, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("dashboard total invited: %w", err)
	}
	dashboard.TotalInvited = totalInvited

	// Count sent invitations
	var totalSent int
	query = `
		SELECT COUNT(*) FROM invitations
		WHERE tenant_id = $1 AND event_id = $2 AND deleted_at IS NULL AND sent_at IS NOT NULL
	`
	err = r.db.GetContext(ctx, &totalSent, query, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("dashboard total sent: %w", err)
	}
	dashboard.TotalSent = totalSent

	// Count opened invitations
	var opened int
	query = `
		SELECT COUNT(*) FROM invitations
		WHERE tenant_id = $1 AND event_id = $2 AND deleted_at IS NULL AND opened_at IS NOT NULL
	`
	err = r.db.GetContext(ctx, &opened, query, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("dashboard opened: %w", err)
	}
	dashboard.Opened = opened

	// Aggregate RSVP counts by status
	var counts struct {
		Responded    int `db:"responded"`
		Attending    int `db:"attending"`
		AttendingPax int `db:"attending_pax"`
		NotAttending int `db:"not_attending"`
		Maybe        int `db:"maybe_count"`
		Waitlist     int `db:"waitlist_count"`
		NoResponse   int `db:"no_response"`
	}

	query = `
		SELECT
			COUNT(*) as responded,
			COUNT(*) FILTER (WHERE status = 'attending') as attending,
			COALESCE(SUM(attending_pax) FILTER (WHERE status = 'attending'), 0) as attending_pax,
			COUNT(*) FILTER (WHERE status = 'not_attending') as not_attending,
			COUNT(*) FILTER (WHERE status = 'maybe') as maybe_count,
			COUNT(*) FILTER (WHERE status = 'waitlist') as waitlist_count,
			COUNT(*) FILTER (WHERE status IN ('not_sent', 'pending', 'no_response')) as no_response
		FROM rsvp_responses
		WHERE tenant_id = $1 AND event_id = $2
	`
	err = r.db.GetContext(ctx, &counts, query, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("dashboard rsvp counts: %w", err)
	}

	dashboard.Responded = counts.Responded
	dashboard.Attending = counts.Attending
	dashboard.AttendingPax = counts.AttendingPax
	dashboard.NotAttending = counts.NotAttending
	dashboard.Maybe = counts.Maybe
	dashboard.Waitlist = counts.Waitlist
	// No response means an active roster member without any RSVP response.
	// This keeps the metric useful even before invitations are created.
	var noResponse int
	query = `
		SELECT COUNT(*)
		FROM event_guests eg
		WHERE eg.tenant_id = $1 AND eg.event_id = $2
		  AND eg.status = 'active' AND eg.deleted_at IS NULL
		  AND NOT EXISTS (
			SELECT 1 FROM rsvp_responses r
			WHERE r.tenant_id = eg.tenant_id
			  AND r.event_id = eg.event_id
			  AND r.guest_id = eg.guest_id
		  )
	`
	if err = r.db.GetContext(ctx, &noResponse, query, tenantID, eventID); err != nil {
		return nil, fmt.Errorf("dashboard no response: %w", err)
	}
	dashboard.NoResponse = noResponse
	dashboard.CapacityUsed = counts.AttendingPax

	// Calculate response rate based on sent invitations
	if dashboard.TotalSent > 0 {
		dashboard.ResponseRate = float64(dashboard.Responded) / float64(dashboard.TotalSent) * 100
	}

	return dashboard, nil
}

// SumAttendingPax returns the total attending pax for an event.
func (r *RSVPRepository) SumAttendingPax(ctx context.Context, eventID uuid.UUID) (int, error) {
	var total sql.NullInt64
	query := `
		SELECT COALESCE(SUM(attending_pax), 0)
		FROM rsvp_responses
		WHERE event_id = $1 AND status = $2
	`
	err := r.db.GetContext(ctx, &total, query, eventID, domain.RSVPStatusAttending)
	if err != nil {
		return 0, fmt.Errorf("sum attending pax: %w", err)
	}
	if total.Valid {
		return int(total.Int64), nil
	}
	return 0, nil
}
