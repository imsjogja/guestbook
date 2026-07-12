// Package service provides business logic layer implementations for GuestFlow.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"guestflow/internal/audit"
	"guestflow/internal/domain"
	"guestflow/internal/repository"
	"guestflow/pkg/crypto"

	"github.com/google/uuid"
)

const (
	// invitationTokenBytes is the byte length for random token generation.
	// 32 bytes = 256 bits of entropy.
	invitationTokenBytes = 32

	// invitationBaseURL is the base URL for invitation links.
	invitationBaseURL = "https://app.guestflow.id/i"
)

// InvitationService encapsulates business logic for invitation operations.
type InvitationService struct {
	invitationRepo *repository.InvitationRepository
	eventRepo      *repository.EventRepository
	rsvpRepo       *repository.RSVPRepository
	guestRepo      *repository.GuestRepository
	auditSvc       *audit.Service
	baseURL        string
}

// NewInvitationService creates a new InvitationService.
func NewInvitationService(
	invitationRepo *repository.InvitationRepository,
	eventRepo *repository.EventRepository,
	rsvpRepo *repository.RSVPRepository,
	guestRepo *repository.GuestRepository,
	auditSvc *audit.Service,
) *InvitationService {
	return &InvitationService{
		invitationRepo: invitationRepo,
		eventRepo:      eventRepo,
		rsvpRepo:       rsvpRepo,
		guestRepo:      guestRepo,
		auditSvc:       auditSvc,
		baseURL:        invitationBaseURL,
	}
}

// SetBaseURL allows overriding the default base URL (useful for testing/custom domains).
func (s *InvitationService) SetBaseURL(url string) {
	s.baseURL = url
}

// Create creates a single invitation for a guest.
func (s *InvitationService) Create(ctx context.Context, tenantID, eventID, createdBy uuid.UUID, req domain.InvitationCreateRequest, guestID uuid.UUID) (*domain.Invitation, error) {
	// Verify the event exists and belongs to the tenant.
	event, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("create invitation: %w", err)
	}

	// Check if event is in a valid state for creating invitations.
	if event.Status == domain.EventStatusCancelled || event.Status == domain.EventStatusArchived {
		return nil, fmt.Errorf("create invitation: cannot create invitations for %s events: %w", event.Status, domain.ErrInvalidInput)
	}

	// Verify the guest exists and belongs to the tenant.
	_, err = s.guestRepo.GetByIDForTenant(ctx, tenantID, guestID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("create invitation: guest not found: %w", domain.ErrInvalidInput)
		}
		return nil, fmt.Errorf("create invitation: %w", err)
	}

	// Check if invitation already exists for this guest at this event.
	exists, err := s.invitationRepo.ExistsForGuest(ctx, eventID, guestID)
	if err != nil {
		return nil, fmt.Errorf("create invitation: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("create invitation: invitation already exists for this guest: %w", domain.ErrAlreadyExists)
	}

	// Generate cryptographically secure opaque token.
	token, err := crypto.GenerateRandomToken(invitationTokenBytes)
	if err != nil {
		return nil, fmt.Errorf("create invitation: failed to generate token: %w", err)
	}

	// Compute SHA-256 hash for storage (we never store the raw token).
	tokenHash := crypto.SHA256Hash(token)

	// Build the public URL.
	url := fmt.Sprintf("%s/%s", s.baseURL, token)

	now := time.Now().UTC()
	invitation := &domain.Invitation{
		Base:            domain.NewBase(),
		TenantID:        tenantID,
		EventID:         eventID,
		GuestID:         guestID,
		Token:           token, // Raw token - returned to caller ONCE, never stored in plain text
		TokenHash:       tokenHash,
		URL:             url,
		MaxPax:          req.MaxPax,
		Adults:          req.Adults,
		Children:        req.Children,
		PlusOneAllowed:  req.PlusOneAllowed,
		PlusOneRequired: req.PlusOneRequired,
		Status:          domain.InvitationStatusDraft,
		ExpiresAt:       req.ExpiresAt,
		CreatedBy:       createdBy,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.invitationRepo.Create(ctx, invitation); err != nil {
		return nil, fmt.Errorf("create invitation: %w", err)
	}

	// Audit log.
	_ = s.auditSvc.LogWithUser(ctx, createdBy, tenantID, domain.AuditActionCreate, domain.EntityTypeInvitation, invitation.ID, nil, map[string]interface{}{
		"event_id": eventID.String(),
		"guest_id": guestID.String(),
		"url":      url,
	})

	return invitation, nil
}

// GenerateBatch creates invitations for multiple guests in a single operation.
func (s *InvitationService) GenerateBatch(ctx context.Context, tenantID, eventID, createdBy uuid.UUID, req domain.InvitationCreateRequest) ([]*domain.Invitation, error) {
	if len(req.GuestIDs) == 0 {
		return nil, fmt.Errorf("generate batch invitations: guest_ids required: %w", domain.ErrInvalidInput)
	}

	if len(req.GuestIDs) > 500 {
		return nil, fmt.Errorf("generate batch invitations: maximum 500 guests per batch: %w", domain.ErrInvalidInput)
	}

	// Verify the event exists and belongs to the tenant.
	event, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("generate batch invitations: %w", err)
	}

	if event.Status == domain.EventStatusCancelled || event.Status == domain.EventStatusArchived {
		return nil, fmt.Errorf("generate batch invitations: cannot create invitations for %s events: %w", event.Status, domain.ErrInvalidInput)
	}

	var invitations []*domain.Invitation
	var errorsList []error

	for _, guestID := range req.GuestIDs {
		inv, err := s.Create(ctx, tenantID, eventID, createdBy, req, guestID)
		if err != nil {
			errorsList = append(errorsList, fmt.Errorf("guest %s: %w", guestID, err))
			continue
		}
		invitations = append(invitations, inv)
	}

	if len(invitations) == 0 && len(errorsList) > 0 {
		return nil, fmt.Errorf("generate batch invitations: all failed: %w", errorsList[0])
	}

	return invitations, nil
}

// Get retrieves an invitation by ID with tenant isolation.
func (s *InvitationService) Get(ctx context.Context, tenantID, invitationID uuid.UUID) (*domain.Invitation, error) {
	invitation, err := s.invitationRepo.GetByIDForTenant(ctx, invitationID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get invitation: %w", err)
	}
	// Clear the raw token - only return hash and URL
	invitation.Token = ""
	return invitation, nil
}

// List lists invitations for an event with filtering and pagination.
func (s *InvitationService) List(ctx context.Context, tenantID, eventID uuid.UUID, status string, page, perPage int) ([]*domain.InvitationWithGuest, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	params := domain.InvitationListParams{
		TenantID: tenantID,
		EventID:  eventID,
		Status:   status,
		Page:     page,
		PerPage:  perPage,
	}

	invitations, err := s.invitationRepo.ListByEvent(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list invitations: %w", err)
	}

	total, err := s.invitationRepo.CountByEvent(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list invitations: count: %w", err)
	}

	return invitations, total, nil
}

// ValidateToken validates a raw token and returns the associated invitation.
func (s *InvitationService) ValidateToken(ctx context.Context, token string) (*domain.Invitation, error) {
	if token == "" {
		return nil, fmt.Errorf("validate token: empty token: %w", domain.ErrTokenInvalid)
	}

	// Hash the provided token for lookup.
	tokenHash := crypto.SHA256Hash(token)

	invitation, err := s.invitationRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrInvitationNotFound) {
			return nil, fmt.Errorf("validate token: %w", domain.ErrTokenInvalid)
		}
		return nil, fmt.Errorf("validate token: %w", err)
	}

	// Check if invitation is revoked.
	if invitation.Status == domain.InvitationStatusRevoked {
		return nil, fmt.Errorf("validate token: %w", domain.ErrInvitationRevoked)
	}

	// Check if invitation is soft-deleted.
	if invitation.DeletedAt != nil {
		return nil, fmt.Errorf("validate token: %w", domain.ErrInvitationNotFound)
	}

	// Check if invitation has expired.
	if invitation.ExpiresAt != nil && time.Now().UTC().After(*invitation.ExpiresAt) {
		return nil, fmt.Errorf("validate token: %w", domain.ErrInvitationExpired)
	}

	// Clear the raw token before returning
	invitation.Token = ""

	return invitation, nil
}

// Revoke revokes an invitation.
func (s *InvitationService) Revoke(ctx context.Context, tenantID, invitationID, revokedBy uuid.UUID, reason string) error {
	invitation, err := s.invitationRepo.GetByIDForTenant(ctx, invitationID, tenantID)
	if err != nil {
		return fmt.Errorf("revoke invitation: %w", err)
	}

	if invitation.Status == domain.InvitationStatusRevoked {
		return fmt.Errorf("revoke invitation: %w", domain.ErrInvitationRevoked)
	}

	if err := s.invitationRepo.Revoke(ctx, invitationID, tenantID, revokedBy, reason); err != nil {
		return fmt.Errorf("revoke invitation: %w", err)
	}

	// Audit log.
	_ = s.auditSvc.LogWithUser(ctx, revokedBy, tenantID, domain.AuditActionReject, domain.EntityTypeInvitation, invitationID, map[string]interface{}{
		"previous_status": invitation.Status,
	}, map[string]interface{}{
		"status": domain.InvitationStatusRevoked,
		"reason": reason,
	})

	return nil
}

// SoftDelete soft-deletes an invitation.
func (s *InvitationService) SoftDelete(ctx context.Context, tenantID, invitationID uuid.UUID) error {
	if err := s.invitationRepo.SoftDelete(ctx, invitationID, tenantID); err != nil {
		return fmt.Errorf("delete invitation: %w", err)
	}
	return nil
}

// GetQRData returns QR code data for an invitation (URL only, no personal data).
func (s *InvitationService) GetQRData(ctx context.Context, tenantID, invitationID uuid.UUID) (*domain.QRCodeData, error) {
	invitation, err := s.invitationRepo.GetByIDForTenant(ctx, invitationID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get qr data: %w", err)
	}

	// Return only the URL and identifiers - NO personal data.
	return &domain.QRCodeData{
		URL:       invitation.URL,
		TokenHash: invitation.TokenHash,
		EventID:   invitation.EventID,
		GuestID:   invitation.GuestID,
	}, nil
}

// RecordOpen records that an invitation link was opened.
func (s *InvitationService) RecordOpen(ctx context.Context, token string, ipAddress string) error {
	// Validate the token first.
	invitation, err := s.ValidateToken(ctx, token)
	if err != nil {
		return fmt.Errorf("record open: %w", err)
	}

	// Mark as opened.
	if err := s.invitationRepo.MarkOpened(ctx, invitation.ID); err != nil {
		return fmt.Errorf("record open: %w", err)
	}

	// Log the credential usage.
	usage := &domain.CredentialUsage{
		Base:         domain.NewBase(),
		InvitationID: invitation.ID,
		EventID:      invitation.EventID,
		GuestID:      invitation.GuestID,
		Type:         "opened",
		IPAddress:    &ipAddress,
	}
	if err := s.invitationRepo.LogCredentialUsage(ctx, usage); err != nil {
		// Non-fatal: log but don't fail the request.
		_ = err
	}

	return nil
}

// Send marks invitations as sent.
func (s *InvitationService) Send(ctx context.Context, tenantID, eventID uuid.UUID, invitationIDs []uuid.UUID, sentBy uuid.UUID) error {
	for _, id := range invitationIDs {
		if err := s.invitationRepo.MarkSent(ctx, id, tenantID); err != nil {
			if errors.Is(err, domain.ErrInvitationNotFound) {
				continue // Skip not found
			}
			return fmt.Errorf("send invitation %s: %w", id, err)
		}
	}

	// Audit log.
	_ = s.auditSvc.LogWithUser(ctx, sentBy, tenantID, domain.AuditActionSend, domain.EntityTypeInvitation, eventID, nil, map[string]interface{}{
		"count": len(invitationIDs),
	})

	return nil
}

// CountByEvent returns invitation counts by status for an event.
func (s *InvitationService) CountByEvent(ctx context.Context, tenantID, eventID uuid.UUID) (map[string]int, error) {
	counts, err := s.invitationRepo.CountByStatus(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("count invitations by status: %w", err)
	}
	return counts, nil
}
