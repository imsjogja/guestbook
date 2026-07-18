// Package repository provides data access layer implementations for GuestFlow.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"guestflow/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// InvitationRepository provides data access for invitations with tenant isolation.
type InvitationRepository struct {
	db *sqlx.DB
}

// NewInvitationRepository creates a new InvitationRepository.
func NewInvitationRepository(db *sqlx.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

// Create inserts a new invitation into the database.
func (r *InvitationRepository) Create(ctx context.Context, invitation *domain.Invitation) error {
	query := `
		INSERT INTO invitations (
			id, tenant_id, event_id, guest_id, event_guest_id, token, token_hash, url,
			max_pax, adults, children, plus_one_allowed, plus_one_required,
			status, sent_at, opened_at, revoked_at, revoked_by, revoke_reason,
			expires_at, created_by, created_at, updated_at, deleted_at
		) VALUES (
			:id, :tenant_id, :event_id, :guest_id, :event_guest_id, :token, :token_hash, :url,
			:max_pax, :adults, :children, :plus_one_allowed, :plus_one_required,
			:status, :sent_at, :opened_at, :revoked_at, :revoked_by, :revoke_reason,
			:expires_at, :created_by, :created_at, :updated_at, :deleted_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, invitation)
	if err != nil {
		return fmt.Errorf("create invitation: %w", err)
	}
	return nil
}

// GetByID retrieves an invitation by its ID.
func (r *InvitationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invitation, error) {
	var invitation domain.Invitation
	query := `
		SELECT * FROM invitations
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &invitation, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("get invitation by id: %w", err)
	}
	return &invitation, nil
}

// GetByIDForTenant retrieves an invitation by ID with tenant check.
func (r *InvitationRepository) GetByIDForTenant(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*domain.Invitation, error) {
	var invitation domain.Invitation
	query := `
		SELECT * FROM invitations
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &invitation, query, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("get invitation by id for tenant: %w", err)
	}
	return &invitation, nil
}

// GetByEventAndGuest retrieves an active invitation by event and guest.
func (r *InvitationRepository) GetByEventAndGuest(ctx context.Context, eventID, guestID uuid.UUID) (*domain.Invitation, error) {
	var invitation domain.Invitation
	query := `
		SELECT * FROM invitations
		WHERE event_id = $1 AND guest_id = $2 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &invitation, query, eventID, guestID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("get invitation by event and guest: %w", err)
	}
	return &invitation, nil
}

// GetByTokenHash looks up an invitation by the SHA-256 hash of its token.
func (r *InvitationRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Invitation, error) {
	var invitation domain.Invitation
	query := `
		SELECT * FROM invitations
		WHERE token_hash = $1 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &invitation, query, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("get invitation by token hash: %w", err)
	}
	return &invitation, nil
}

// ListByEvent returns a paginated list of invitations for an event with optional filters.
func (r *InvitationRepository) ListByEvent(ctx context.Context, params domain.InvitationListParams) ([]*domain.InvitationWithGuest, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 20
	}
	offset := (params.Page - 1) * params.PerPage

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("i.tenant_id = $%d", argIdx))
	args = append(args, params.TenantID)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("i.event_id = $%d", argIdx))
	args = append(args, params.EventID)
	argIdx++

	conditions = append(conditions, "i.deleted_at IS NULL")

	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("i.status = $%d", argIdx))
		args = append(args, params.Status)
		argIdx++
	}

	query := `
		SELECT
			i.*,
			g.full_name as guest_full_name,
			g.email as guest_email,
			g.phone as guest_phone,
			COALESCE(r.status, 'not_sent') as rsvp_status,
			COALESCE(dm.status, 'not_sent') as delivery_status,
			dm.channel as delivery_channel,
			dm.sent_at as delivery_sent_at,
			dm.delivered_at as delivery_delivered_at,
			dm.read_at as delivery_read_at,
			dm.failed_at as delivery_failed_at,
			dm.error_message as delivery_error_message,
			dm.external_id as delivery_external_id,
			dm.provider_http_status as delivery_provider_http_status
		FROM invitations i
		JOIN guests g ON i.guest_id = g.id
		LEFT JOIN rsvp_responses r ON i.id = r.invitation_id
		LEFT JOIN LATERAL (
			SELECT cm.status, cm.channel, cm.sent_at, cm.delivered_at, cm.read_at,
				cm.failed_at, cm.error_message, cm.external_id, cm.provider_http_status
			FROM communication_messages cm
			WHERE cm.invitation_id = i.id AND cm.event_id = i.event_id
			ORDER BY cm.created_at DESC
			LIMIT 1
		) dm ON TRUE
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY i.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1) + `
	`
	args = append(args, params.PerPage, offset)

	var invitations []*domain.InvitationWithGuest
	err := r.db.SelectContext(ctx, &invitations, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list invitations by event: %w", err)
	}
	return invitations, nil
}

// CountByEvent counts invitations for an event matching the given filters.
func (r *InvitationRepository) CountByEvent(ctx context.Context, params domain.InvitationListParams) (int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, params.TenantID)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("event_id = $%d", argIdx))
	args = append(args, params.EventID)
	argIdx++

	conditions = append(conditions, "deleted_at IS NULL")

	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, params.Status)
		argIdx++
	}

	query := `
		SELECT COUNT(*) FROM invitations
		WHERE ` + strings.Join(conditions, " AND ")

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("count invitations by event: %w", err)
	}
	return count, nil
}

// UpdateStatus updates the status of an invitation.
func (r *InvitationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, status string) error {
	query := `
		UPDATE invitations
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, status, id, tenantID)
	if err != nil {
		return fmt.Errorf("update invitation status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check status update result: %w", err)
	}
	if rows == 0 {
		return domain.ErrInvitationNotFound
	}

	return nil
}

// MarkOpened records when an invitation link was opened.
func (r *InvitationRepository) MarkOpened(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	query := `
		UPDATE invitations
		SET status = CASE WHEN status = $1 THEN $2 ELSE status END,
		    opened_at = $3,
		    updated_at = $3
		WHERE id = $4 AND deleted_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, domain.InvitationStatusSent, domain.InvitationStatusOpened, now, id)
	if err != nil {
		return fmt.Errorf("mark invitation opened: %w", err)
	}
	return nil
}

// SoftDelete marks an invitation as deleted.
func (r *InvitationRepository) SoftDelete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	now := time.Now().UTC()
	query := `
		UPDATE invitations
		SET deleted_at = $1, updated_at = $1, status = $2
		WHERE id = $3 AND tenant_id = $4 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, now, domain.InvitationStatusRevoked, id, tenantID)
	if err != nil {
		return fmt.Errorf("soft delete invitation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check delete result: %w", err)
	}
	if rows == 0 {
		return domain.ErrInvitationNotFound
	}

	return nil
}

// Revoke revokes an invitation with a reason.
func (r *InvitationRepository) Revoke(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, revokedBy uuid.UUID, reason string) error {
	now := time.Now().UTC()
	query := `
		UPDATE invitations
		SET status = $1, revoked_at = $2, revoked_by = $3, revoke_reason = $4, updated_at = $2
		WHERE id = $5 AND tenant_id = $6 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, domain.InvitationStatusRevoked, now, revokedBy, reason, id, tenantID)
	if err != nil {
		return fmt.Errorf("revoke invitation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check revoke result: %w", err)
	}
	if rows == 0 {
		return domain.ErrInvitationNotFound
	}

	return nil
}

// ExistsForGuest checks if an active invitation already exists for a guest at an event.
func (r *InvitationRepository) ExistsForGuest(ctx context.Context, eventID, guestID uuid.UUID) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM invitations
		WHERE event_id = $1 AND guest_id = $2 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &count, query, eventID, guestID)
	if err != nil {
		return false, fmt.Errorf("check invitation exists for guest: %w", err)
	}
	return count > 0, nil
}

// LogCredentialUsage records a usage event for an invitation credential.
func (r *InvitationRepository) LogCredentialUsage(ctx context.Context, usage *domain.CredentialUsage) error {
	query := `
		INSERT INTO credential_usage_log (
			id, invitation_id, event_id, guest_id, type,
			device_id, gate_id, officer_id, ip_address, metadata, created_at
		) VALUES (
			:id, :invitation_id, :event_id, :guest_id, :type,
			:device_id, :gate_id, :officer_id, :ip_address, :metadata, :created_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, usage)
	if err != nil {
		return fmt.Errorf("log credential usage: %w", err)
	}
	return nil
}

// CountByStatus counts invitations per status for an event.
func (r *InvitationRepository) CountByStatus(ctx context.Context, eventID uuid.UUID) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM invitations
		WHERE event_id = $1 AND deleted_at IS NULL
		GROUP BY status
	`
	rows, err := r.db.QueryxContext(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("count invitations by status: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan invitation status count: %w", err)
		}
		result[status] = count
	}

	return result, nil
}
