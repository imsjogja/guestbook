package audit

import (
	"context"
	"fmt"

	"guestflow/internal/domain"
	mid "guestflow/internal/middleware"
	"guestflow/internal/repository"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Service handles audit logging operations.
type Service struct {
	repo *repository.AuditRepository
}

// NewService creates a new audit Service.
func NewService(repo *repository.AuditRepository) *Service {
	return &Service{repo: repo}
}

// Log records an audit entry. UserID and TenantID are extracted from the context.
func (s *Service) Log(ctx context.Context, action, entityType string, entityID uuid.UUID, oldValues, newValues map[string]interface{}) error {
	tenantID, userID, ipAddress, userAgent := extractFromContext(ctx)

	log := domain.NewAuditLog()
	log.TenantID = tenantID
	log.UserID = userID
	log.Action = action
	log.EntityType = entityType
	log.EntityID = &entityID
	if oldValues != nil {
		log.OldValues = oldValues
	}
	if newValues != nil {
		log.NewValues = newValues
	}
	log.IPAddress = ipAddress
	log.UserAgent = userAgent

	if err := s.repo.Create(ctx, log); err != nil {
		return fmt.Errorf("audit log: %w", err)
	}
	return nil
}

// LogWithUser records an audit entry with explicit user and tenant IDs.
func (s *Service) LogWithUser(ctx context.Context, userID, tenantID uuid.UUID, action, entityType string, entityID uuid.UUID, oldValues, newValues map[string]interface{}) error {
	log := domain.NewAuditLog()
	log.TenantID = &tenantID
	log.UserID = &userID
	log.Action = action
	log.EntityType = entityType
	log.EntityID = &entityID
	if oldValues != nil {
		log.OldValues = oldValues
	}
	if newValues != nil {
		log.NewValues = newValues
	}

	// Attempt to extract IP and user agent from context.
	_, _, ipAddress, userAgent := extractFromContext(ctx)
	log.IPAddress = ipAddress
	log.UserAgent = userAgent

	if err := s.repo.Create(ctx, log); err != nil {
		return fmt.Errorf("audit log with user: %w", err)
	}
	return nil
}

// LogSimple is a convenience method that logs a simple action without old/new values.
func (s *Service) LogSimple(ctx context.Context, action, entityType string, entityID uuid.UUID) error {
	return s.Log(ctx, action, entityType, entityID, nil, nil)
}

// extractFromContext attempts to extract tenantID, userID, IP address, and user agent
// from the Echo context if available, otherwise returns nil values.
func extractFromContext(ctx context.Context) (*uuid.UUID, *uuid.UUID, *string, *string) {
	var tenantID, userID *uuid.UUID
	var ipAddress, userAgent *string

	// Try to get Echo context from the Go context.
	echoCtx, ok := ctx.Value("echo_context").(echo.Context)
	if !ok {
		return tenantID, userID, ipAddress, userAgent
	}

	// Extract tenant_id from path parameter.
	if tidStr := echoCtx.Param("id"); tidStr != "" {
		if tid, err := uuid.Parse(tidStr); err == nil {
			tenantID = &tid
		}
	}

	// Extract user_id from context using the middleware helper.
	if uid := mid.GetUserID(echoCtx); uid != uuid.Nil {
		userID = &uid
	}

	// Extract IP address.
	ip := echoCtx.RealIP()
	if ip == "" {
		ip = echoCtx.Request().RemoteAddr
	}
	if ip != "" {
		ipAddress = &ip
	}

	// Extract user agent.
	ua := echoCtx.Request().UserAgent()
	if ua != "" {
		userAgent = &ua
	}

	return tenantID, userID, ipAddress, userAgent
}
