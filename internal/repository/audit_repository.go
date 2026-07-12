package repository

import (
	"context"
	"fmt"

	"guestflow/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// AuditRepository provides data access for audit logs.
type AuditRepository struct {
	db *sqlx.DB
}

// NewAuditRepository creates a new AuditRepository.
func NewAuditRepository(db *sqlx.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create inserts a new audit log entry.
func (r *AuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			id, tenant_id, user_id, action, entity_type, entity_id,
			old_values, new_values, ip_address, user_agent, metadata, created_at
		) VALUES (
			:id, :tenant_id, :user_id, :action, :entity_type, :entity_id,
			:old_values, :new_values, :ip_address, :user_agent, :metadata, :created_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, log)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

// ListByTenant lists audit logs for a specific tenant with pagination.
// Tenant isolation: only logs belonging to the given tenant are returned.
func (r *AuditRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var logs []*domain.AuditLog
	query := `
		SELECT * FROM audit_logs
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &logs, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs by tenant: %w", err)
	}
	return logs, nil
}

// ListByEntity lists audit logs for a specific entity type and entity ID.
func (r *AuditRepository) ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*domain.AuditLog, error) {
	var logs []*domain.AuditLog
	query := `
		SELECT * FROM audit_logs
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
	`
	err := r.db.SelectContext(ctx, &logs, query, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs by entity: %w", err)
	}
	return logs, nil
}

// CountByTenant counts the total number of audit log entries for a tenant.
func (r *AuditRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM audit_logs
		WHERE tenant_id = $1
	`
	err := r.db.GetContext(ctx, &count, query, tenantID)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}
	return count, nil
}
