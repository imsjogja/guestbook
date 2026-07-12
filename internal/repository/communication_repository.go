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

// CommunicationRepository provides data access for communication templates,
// campaigns, and messages with tenant isolation.
type CommunicationRepository struct {
	db *sqlx.DB
}

// NewCommunicationRepository creates a new CommunicationRepository.
func NewCommunicationRepository(db *sqlx.DB) *CommunicationRepository {
	return &CommunicationRepository{db: db}
}

// ---------------------------------------------------------------------------
// Templates
// ---------------------------------------------------------------------------

// CreateTemplate inserts a new communication template.
func (r *CommunicationRepository) CreateTemplate(ctx context.Context, template *domain.CommunicationTemplate) error {
	query := `
		INSERT INTO communication_templates (
			id, tenant_id, name, channel, type, subject, body, variables,
			is_active, is_system, description, language,
			created_at, updated_at, deleted_at
		) VALUES (
			:id, :tenant_id, :name, :channel, :type, :subject, :body, :variables,
			:is_active, :is_system, :description, :language,
			:created_at, :updated_at, :deleted_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, template)
	if err != nil {
		return fmt.Errorf("create template: %w", err)
	}
	return nil
}

// GetTemplate retrieves a template by ID with tenant isolation.
func (r *CommunicationRepository) GetTemplate(ctx context.Context, tenantID, id uuid.UUID) (*domain.CommunicationTemplate, error) {
	var template domain.CommunicationTemplate
	query := `
		SELECT * FROM communication_templates
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &template, query, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("get template: %w", err)
	}
	return &template, nil
}

// UpdateTemplate modifies an existing template.
func (r *CommunicationRepository) UpdateTemplate(ctx context.Context, template *domain.CommunicationTemplate) error {
	query := `
		UPDATE communication_templates SET
			name = :name,
			channel = :channel,
			type = :type,
			subject = :subject,
			body = :body,
			variables = :variables,
			is_active = :is_active,
			description = :description,
			language = :language,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id AND deleted_at IS NULL
	`
	result, err := r.db.NamedExecContext(ctx, query, template)
	if err != nil {
		return fmt.Errorf("update template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check update result: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrTemplateNotFound
	}
	return nil
}

// SoftDeleteTemplate marks a template as deleted.
func (r *CommunicationRepository) SoftDeleteTemplate(ctx context.Context, tenantID, id uuid.UUID) error {
	now := time.Now().UTC()
	query := `
		UPDATE communication_templates
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, now, id, tenantID)
	if err != nil {
		return fmt.Errorf("soft delete template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check delete result: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrTemplateNotFound
	}
	return nil
}

// ListTemplatesByTenant lists templates for a tenant with optional filters.
func (r *CommunicationRepository) ListTemplatesByTenant(ctx context.Context, params domain.TemplateListParams) ([]*domain.CommunicationTemplate, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 20
	}
	offset := (params.Page - 1) * params.PerPage

	where, args := r.buildTemplateWhereClause(params)
	query := fmt.Sprintf(`
		SELECT * FROM communication_templates
		%s
		ORDER BY created_at DESC
		LIMIT %d OFFSET %d
	`, where, params.PerPage, offset)

	var templates []*domain.CommunicationTemplate
	err := r.db.SelectContext(ctx, &templates, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	return templates, nil
}

// CountTemplatesByTenant counts templates matching the filter.
func (r *CommunicationRepository) CountTemplatesByTenant(ctx context.Context, params domain.TemplateListParams) (int, error) {
	where, args := r.buildTemplateWhereClause(params)
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM communication_templates
		%s
	`, where)

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("count templates: %w", err)
	}
	return count, nil
}

func (r *CommunicationRepository) buildTemplateWhereClause(params domain.TemplateListParams) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d AND deleted_at IS NULL", argIdx))
	args = append(args, params.TenantID)
	argIdx++

	if params.Channel != "" {
		conditions = append(conditions, fmt.Sprintf("channel = $%d", argIdx))
		args = append(args, params.Channel)
		argIdx++
	}

	if params.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, params.Type)
		argIdx++
	}

	if params.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *params.IsActive)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")
	return where, args
}

// ---------------------------------------------------------------------------
// Campaigns
// ---------------------------------------------------------------------------

// CreateCampaign inserts a new communication campaign.
func (r *CommunicationRepository) CreateCampaign(ctx context.Context, campaign *domain.CommunicationCampaign) error {
	query := `
		INSERT INTO communication_campaigns (
			id, tenant_id, event_id, name, template_id, channel, type, status,
			recipient_filter, scheduled_at, sent_at, completed_at,
			total_recipients, sent_count, failed_count, created_by,
			created_at, updated_at, deleted_at
		) VALUES (
			:id, :tenant_id, :event_id, :name, :template_id, :channel, :type, :status,
			:recipient_filter, :scheduled_at, :sent_at, :completed_at,
			:total_recipients, :sent_count, :failed_count, :created_by,
			:created_at, :updated_at, :deleted_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, campaign)
	if err != nil {
		return fmt.Errorf("create campaign: %w", err)
	}
	return nil
}

// GetCampaign retrieves a campaign by ID with tenant isolation.
func (r *CommunicationRepository) GetCampaign(ctx context.Context, tenantID, id uuid.UUID) (*domain.CommunicationCampaign, error) {
	var campaign domain.CommunicationCampaign
	query := `
		SELECT * FROM communication_campaigns
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &campaign, query, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCampaignNotFound
		}
		return nil, fmt.Errorf("get campaign: %w", err)
	}
	return &campaign, nil
}

// UpdateCampaign modifies an existing campaign.
func (r *CommunicationRepository) UpdateCampaign(ctx context.Context, campaign *domain.CommunicationCampaign) error {
	query := `
		UPDATE communication_campaigns SET
			name = :name,
			template_id = :template_id,
			channel = :channel,
			type = :type,
			status = :status,
			recipient_filter = :recipient_filter,
			scheduled_at = :scheduled_at,
			sent_at = :sent_at,
			completed_at = :completed_at,
			total_recipients = :total_recipients,
			sent_count = :sent_count,
			failed_count = :failed_count,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id AND deleted_at IS NULL
	`
	result, err := r.db.NamedExecContext(ctx, query, campaign)
	if err != nil {
		return fmt.Errorf("update campaign: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check update result: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrCampaignNotFound
	}
	return nil
}

// UpdateCampaignStatus updates only the status and related timestamp fields of a campaign.
func (r *CommunicationRepository) UpdateCampaignStatus(ctx context.Context, tenantID, id uuid.UUID, status string, sentAt, completedAt *time.Time, sentCount, failedCount int) error {
	now := time.Now().UTC()
	query := `
		UPDATE communication_campaigns
		SET status = $1, sent_at = $2, completed_at = $3,
		    sent_count = $4, failed_count = $5, updated_at = $6
		WHERE id = $7 AND tenant_id = $8 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, status, sentAt, completedAt, sentCount, failedCount, now, id, tenantID)
	if err != nil {
		return fmt.Errorf("update campaign status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check update result: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrCampaignNotFound
	}
	return nil
}

// ListCampaignsByEvent lists campaigns for an event with optional status filter.
func (r *CommunicationRepository) ListCampaignsByEvent(ctx context.Context, params domain.CampaignListParams) ([]*domain.CommunicationCampaign, error) {
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

	where := "WHERE " + strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`
		SELECT * FROM communication_campaigns
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, params.PerPage, offset)

	var campaigns []*domain.CommunicationCampaign
	err := r.db.SelectContext(ctx, &campaigns, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list campaigns: %w", err)
	}
	return campaigns, nil
}

// CountCampaignsByEvent counts campaigns matching the filter.
func (r *CommunicationRepository) CountCampaignsByEvent(ctx context.Context, params domain.CampaignListParams) (int, error) {
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

	where := "WHERE " + strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM communication_campaigns
		%s
	`, where)

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("count campaigns: %w", err)
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// CreateMessage inserts a new communication message.
func (r *CommunicationRepository) CreateMessage(ctx context.Context, message *domain.CommunicationMessage) error {
	query := `
		INSERT INTO communication_messages (
			id, tenant_id, campaign_id, event_id, guest_id, invitation_id,
			channel, type, subject, body, status,
			sent_at, delivered_at, read_at, failed_at,
			error_message, external_id, cost,
			created_at, updated_at
		) VALUES (
			:id, :tenant_id, :campaign_id, :event_id, :guest_id, :invitation_id,
			:channel, :type, :subject, :body, :status,
			:sent_at, :delivered_at, :read_at, :failed_at,
			:error_message, :external_id, :cost,
			:created_at, :updated_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, message)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return nil
}

// GetMessage retrieves a message by ID with tenant isolation.
func (r *CommunicationRepository) GetMessage(ctx context.Context, tenantID, id uuid.UUID) (*domain.CommunicationMessage, error) {
	var message domain.CommunicationMessage
	query := `
		SELECT * FROM communication_messages
		WHERE id = $1 AND tenant_id = $2
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &message, query, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMessageNotFound
		}
		return nil, fmt.Errorf("get message: %w", err)
	}
	return &message, nil
}

// UpdateMessageStatus updates the status and related fields of a message.
func (r *CommunicationRepository) UpdateMessageStatus(ctx context.Context, tenantID, id uuid.UUID, status string, sentAt, deliveredAt, readAt, failedAt *time.Time, errorMessage, externalID *string, cost *float64) error {
	now := time.Now().UTC()
	query := `
		UPDATE communication_messages
		SET status = $1, sent_at = $2, delivered_at = $3, read_at = $4,
		    failed_at = $5, error_message = $6, external_id = $7, cost = $8,
		    updated_at = $9
		WHERE id = $10 AND tenant_id = $11
	`
	_, err := r.db.ExecContext(ctx, query, status, sentAt, deliveredAt, readAt, failedAt, errorMessage, externalID, cost, now, id, tenantID)
	if err != nil {
		return fmt.Errorf("update message status: %w", err)
	}
	return nil
}

// ListMessagesByCampaign lists messages for a campaign.
func (r *CommunicationRepository) ListMessagesByCampaign(ctx context.Context, params domain.MessageListParams) ([]*domain.CommunicationMessage, error) {
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

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, params.TenantID)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("event_id = $%d", argIdx))
	args = append(args, params.EventID)
	argIdx++

	if params.CampaignID != nil {
		conditions = append(conditions, fmt.Sprintf("campaign_id = $%d", argIdx))
		args = append(args, *params.CampaignID)
		argIdx++
	}

	if params.GuestID != nil {
		conditions = append(conditions, fmt.Sprintf("guest_id = $%d", argIdx))
		args = append(args, *params.GuestID)
		argIdx++
	}

	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, params.Status)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`
		SELECT * FROM communication_messages
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, params.PerPage, offset)

	var messages []*domain.CommunicationMessage
	err := r.db.SelectContext(ctx, &messages, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	return messages, nil
}

// CountMessagesByCampaign counts messages for a campaign.
func (r *CommunicationRepository) CountMessagesByCampaign(ctx context.Context, params domain.MessageListParams) (int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, params.TenantID)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("event_id = $%d", argIdx))
	args = append(args, params.EventID)
	argIdx++

	if params.CampaignID != nil {
		conditions = append(conditions, fmt.Sprintf("campaign_id = $%d", argIdx))
		args = append(args, *params.CampaignID)
		argIdx++
	}

	if params.GuestID != nil {
		conditions = append(conditions, fmt.Sprintf("guest_id = $%d", argIdx))
		args = append(args, *params.GuestID)
		argIdx++
	}

	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, params.Status)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM communication_messages
		%s
	`, where)

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("count messages: %w", err)
	}
	return count, nil
}

// ListMessagesByGuest lists messages for a specific guest.
func (r *CommunicationRepository) ListMessagesByGuest(ctx context.Context, tenantID, eventID, guestID uuid.UUID, page, perPage int) ([]*domain.CommunicationMessage, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	query := `
		SELECT * FROM communication_messages
		WHERE tenant_id = $1 AND event_id = $2 AND guest_id = $3
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`
	var messages []*domain.CommunicationMessage
	err := r.db.SelectContext(ctx, &messages, query, tenantID, eventID, guestID, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("list messages by guest: %w", err)
	}
	return messages, nil
}

// CountMessagesByGuest counts messages for a guest.
func (r *CommunicationRepository) CountMessagesByGuest(ctx context.Context, tenantID, eventID, guestID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM communication_messages
		WHERE tenant_id = $1 AND event_id = $2 AND guest_id = $3
	`
	var count int
	err := r.db.GetContext(ctx, &count, query, tenantID, eventID, guestID)
	if err != nil {
		return 0, fmt.Errorf("count messages by guest: %w", err)
	}
	return count, nil
}

// CountByStatus returns aggregated message counts grouped by status for an event.
func (r *CommunicationRepository) CountByStatus(ctx context.Context, tenantID, eventID uuid.UUID) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM communication_messages
		WHERE tenant_id = $1 AND event_id = $2
		GROUP BY status
	`
	rows, err := r.db.QueryxContext(ctx, query, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("count messages by status: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan message status count: %w", err)
		}
		result[status] = count
	}
	return result, nil
}

// GetActiveCampaignsCount returns the number of active (sending or scheduled) campaigns for an event.
func (r *CommunicationRepository) GetActiveCampaignsCount(ctx context.Context, tenantID, eventID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM communication_campaigns
		WHERE tenant_id = $1 AND event_id = $2
		  AND status IN ('sending', 'scheduled')
		  AND deleted_at IS NULL
	`
	var count int
	err := r.db.GetContext(ctx, &count, query, tenantID, eventID)
	if err != nil {
		return 0, fmt.Errorf("count active campaigns: %w", err)
	}
	return count, nil
}

// GetRecentMessages returns the most recent messages for an event.
func (r *CommunicationRepository) GetRecentMessages(ctx context.Context, tenantID, eventID uuid.UUID, limit int) ([]domain.CommunicationMessage, error) {
	if limit < 1 {
		limit = 10
	}
	query := `
		SELECT * FROM communication_messages
		WHERE tenant_id = $1 AND event_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`
	var messages []domain.CommunicationMessage
	err := r.db.SelectContext(ctx, &messages, query, tenantID, eventID, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent messages: %w", err)
	}
	return messages, nil
}
