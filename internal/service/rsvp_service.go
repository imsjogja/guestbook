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

// RSVPService encapsulates business logic for RSVP operations.
type RSVPService struct {
	rsvpRepo       *repository.RSVPRepository
	invitationRepo *repository.InvitationRepository
	eventRepo      *repository.EventRepository
	eventGuestRepo *repository.EventGuestRepository
	auditSvc       *audit.Service
}

// NewRSVPService creates a new RSVPService.
func NewRSVPService(
	rsvpRepo *repository.RSVPRepository,
	invitationRepo *repository.InvitationRepository,
	eventRepo *repository.EventRepository,
	eventGuestRepo *repository.EventGuestRepository,
	auditSvc *audit.Service,
) *RSVPService {
	return &RSVPService{
		rsvpRepo:       rsvpRepo,
		invitationRepo: invitationRepo,
		eventRepo:      eventRepo,
		eventGuestRepo: eventGuestRepo,
		auditSvc:       auditSvc,
	}
}

// Submit processes a public RSVP submission by token.
func (s *RSVPService) Submit(ctx context.Context, req domain.RSVPSubmitRequest, ipAddress string) (*domain.RSVPResponse, error) {
	// Validate the token and get the invitation.
	invitation, err := s.validateTokenForRSVP(ctx, req.Token)
	if err != nil {
		return nil, fmt.Errorf("submit rsvp: %w", err)
	}

	// Validate RSVP deadline.
	if err := s.ValidateDeadline(ctx, invitation.EventID); err != nil {
		return nil, fmt.Errorf("submit rsvp: %w", err)
	}

	// Validate status.
	if !isValidRSVPSubmitStatus(req.Status) {
		return nil, fmt.Errorf("submit rsvp: invalid status: %w", domain.ErrInvalidRSVPStatus)
	}

	// For not_attending, attending_pax should be 0.
	attendingPax := req.AttendingPax
	if req.Status == domain.RSVPStatusNotAttending {
		attendingPax = 0
	}

	// Validate capacity for attending responses.
	if req.Status == domain.RSVPStatusAttending {
		if err := s.ValidateCapacity(ctx, invitation.EventID, invitation.TenantID, attendingPax, invitation.ID); err != nil {
			return nil, fmt.Errorf("submit rsvp: %w", err)
		}
	}

	// Validate attending_pax does not exceed invitation max_pax.
	if attendingPax > invitation.MaxPax {
		return nil, fmt.Errorf("submit rsvp: attending pax exceeds invitation maximum: %w", domain.ErrInvalidInput)
	}

	// Check if RSVP already exists for this invitation.
	var existingRSVP *domain.RSVPResponse
	existingRSVP, err = s.rsvpRepo.GetByInvitation(ctx, invitation.ID)
	if err != nil && !errors.Is(err, domain.ErrRSVPNotFound) {
		return nil, fmt.Errorf("submit rsvp: check existing: %w", err)
	}

	now := time.Now().UTC()

	if existingRSVP != nil {
		// Update existing RSVP.
		existingRSVP.Status = req.Status
		existingRSVP.AttendingPax = attendingPax
		existingRSVP.Adults = req.Adults
		existingRSVP.Children = req.Children
		existingRSVP.AttendingSessions = req.AttendingSessions
		existingRSVP.EditedAt = &now
		existingRSVP.IPAddress = &ipAddress

		if req.MenuChoice != "" {
			existingRSVP.MenuChoice = &req.MenuChoice
		}
		if req.Allergies != "" {
			existingRSVP.Allergies = &req.Allergies
		}
		if req.AccessibilityNeeds != "" {
			existingRSVP.AccessibilityNeeds = &req.AccessibilityNeeds
		}
		if req.Transportation != "" {
			existingRSVP.Transportation = &req.Transportation
		}
		if req.Notes != "" {
			existingRSVP.Notes = &req.Notes
		}
		existingRSVP.UpdatedAt = now

		if err := s.rsvpRepo.Update(ctx, existingRSVP); err != nil {
			return nil, fmt.Errorf("submit rsvp: update: %w", err)
		}

		// Update invitation status to responded.
		_ = s.invitationRepo.UpdateStatus(ctx, invitation.ID, invitation.TenantID, domain.InvitationStatusResponded)

		return existingRSVP, nil
	}

	// Create new RSVP.
	rsvp := &domain.RSVPResponse{
		Base:              domain.NewBase(),
		TenantID:          invitation.TenantID,
		EventID:           invitation.EventID,
		InvitationID:      invitation.ID,
		GuestID:           invitation.GuestID,
		EventGuestID:      invitation.EventGuestID,
		Status:            req.Status,
		AttendingPax:      attendingPax,
		Adults:            req.Adults,
		Children:          req.Children,
		AttendingSessions: req.AttendingSessions,
		RespondedAt:       &now,
		IPAddress:         &ipAddress,
	}

	if req.MenuChoice != "" {
		rsvp.MenuChoice = &req.MenuChoice
	}
	if req.Allergies != "" {
		rsvp.Allergies = &req.Allergies
	}
	if req.AccessibilityNeeds != "" {
		rsvp.AccessibilityNeeds = &req.AccessibilityNeeds
	}
	if req.Transportation != "" {
		rsvp.Transportation = &req.Transportation
	}
	if req.Notes != "" {
		rsvp.Notes = &req.Notes
	}

	if err := s.rsvpRepo.Create(ctx, rsvp); err != nil {
		return nil, fmt.Errorf("submit rsvp: create: %w", err)
	}

	// Update invitation status to responded.
	_ = s.invitationRepo.UpdateStatus(ctx, invitation.ID, invitation.TenantID, domain.InvitationStatusResponded)

	return rsvp, nil
}

// GetByEvent retrieves RSVP responses for an event.
func (s *RSVPService) GetByEvent(ctx context.Context, tenantID, eventID uuid.UUID, status string, page, perPage int) ([]*domain.RSVPResponseWithGuest, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	rsvps, err := s.rsvpRepo.GetByEvent(ctx, tenantID, eventID, status, page, perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("get rsvps by event: %w", err)
	}

	total, err := s.rsvpRepo.CountByEvent(ctx, tenantID, eventID, status)
	if err != nil {
		return nil, 0, fmt.Errorf("get rsvps by event: count: %w", err)
	}

	return rsvps, total, nil
}

// GetDashboard returns aggregated RSVP stats for an event dashboard.
func (s *RSVPService) GetDashboard(ctx context.Context, tenantID, eventID uuid.UUID) (*domain.RSVPDashboard, error) {
	// Verify the event exists and belongs to the tenant.
	event, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get dashboard: %w", err)
	}

	dashboard, err := s.rsvpRepo.GetDashboardStats(ctx, tenantID, eventID, event.Capacity)
	if err != nil {
		return nil, fmt.Errorf("get dashboard: %w", err)
	}

	return dashboard, nil
}

// UpdateByOfficer allows an authorized officer to manually update an RSVP.
func (s *RSVPService) UpdateByOfficer(ctx context.Context, tenantID, eventID, rsvpID, officerID uuid.UUID, req domain.RSVPUpdateRequest) (*domain.RSVPResponse, error) {
	// Validate status.
	if !domain.IsValidRSVPStatus(req.Status) {
		return nil, fmt.Errorf("update rsvp: invalid status: %w", domain.ErrInvalidRSVPStatus)
	}

	// Get existing RSVP.
	rsvp, err := s.rsvpRepo.GetByIDForTenant(ctx, rsvpID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("update rsvp: %w", err)
	}
	if rsvp.EventID != eventID {
		return nil, fmt.Errorf("update rsvp: %w", domain.ErrRSVPNotFound)
	}

	// If changing to attending, validate capacity.
	if req.Status == domain.RSVPStatusAttending && rsvp.Status != domain.RSVPStatusAttending {
		if err := s.ValidateCapacity(ctx, eventID, tenantID, req.AttendingPax, rsvp.InvitationID); err != nil {
			return nil, fmt.Errorf("update rsvp: %w", err)
		}
	}

	now := time.Now().UTC()

	rsvp.Status = req.Status
	rsvp.AttendingPax = req.AttendingPax
	rsvp.Adults = req.Adults
	rsvp.Children = req.Children
	rsvp.AttendingSessions = req.AttendingSessions
	rsvp.EditedAt = &now
	rsvp.EditedBy = &officerID
	rsvp.UpdatedAt = now

	if req.MenuChoice != "" {
		rsvp.MenuChoice = &req.MenuChoice
	}
	if req.Allergies != "" {
		rsvp.Allergies = &req.Allergies
	}
	if req.AccessibilityNeeds != "" {
		rsvp.AccessibilityNeeds = &req.AccessibilityNeeds
	}
	if req.Transportation != "" {
		rsvp.Transportation = &req.Transportation
	}
	if req.Notes != "" {
		rsvp.Notes = &req.Notes
	}

	if err := s.rsvpRepo.Update(ctx, rsvp); err != nil {
		return nil, fmt.Errorf("update rsvp: %w", err)
	}

	// Audit log.
	_ = s.auditSvc.LogWithUser(ctx, officerID, tenantID, domain.AuditActionUpdate, domain.EntityTypeRSVP, rsvpID, nil, map[string]interface{}{
		"status":        req.Status,
		"attending_pax": req.AttendingPax,
	})

	return rsvp, nil
}

// UpsertByOfficer creates or updates an RSVP for a guest within an event.
func (s *RSVPService) UpsertByOfficer(ctx context.Context, tenantID, eventID, guestID, officerID uuid.UUID, req domain.RSVPUpdateRequest) (*domain.RSVPResponse, error) {
	if !domain.IsValidRSVPStatus(req.Status) {
		return nil, fmt.Errorf("upsert rsvp: invalid status: %w", domain.ErrInvalidRSVPStatus)
	}

	if _, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID); err != nil {
		return nil, fmt.Errorf("upsert rsvp: %w", err)
	}
	eventGuest, err := s.eventGuestRepo.GetByEventAndGuest(ctx, tenantID, eventID, guestID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("upsert rsvp: guest is not in event roster: %w", domain.ErrInvalidInput)
		}
		return nil, fmt.Errorf("upsert rsvp: check event roster: %w", err)
	}

	invitation, err := s.invitationRepo.GetByEventAndGuest(ctx, eventID, guestID)
	if err != nil {
		return nil, fmt.Errorf("upsert rsvp: %w", err)
	}
	if invitation.TenantID != tenantID {
		return nil, fmt.Errorf("upsert rsvp: %w", domain.ErrInvitationNotFound)
	}

	if req.AttendingPax < 0 {
		req.AttendingPax = 0
	}
	if req.Status == domain.RSVPStatusNotAttending {
		req.AttendingPax = 0
	}

	existing, err := s.rsvpRepo.GetByInvitation(ctx, invitation.ID)
	if err != nil && !errors.Is(err, domain.ErrRSVPNotFound) {
		return nil, fmt.Errorf("upsert rsvp: %w", err)
	}

	if req.Status == domain.RSVPStatusAttending {
		if err := s.ValidateCapacity(ctx, eventID, tenantID, req.AttendingPax, invitation.ID); err != nil {
			return nil, fmt.Errorf("upsert rsvp: %w", err)
		}
	}

	now := time.Now().UTC()
	if existing != nil {
		existing.Status = req.Status
		existing.AttendingPax = req.AttendingPax
		existing.Adults = req.Adults
		existing.Children = req.Children
		existing.AttendingSessions = req.AttendingSessions
		existing.EditedAt = &now
		existing.EditedBy = &officerID
		existing.UpdatedAt = now
		if existing.RespondedAt == nil {
			existing.RespondedAt = &now
		}

		if req.MenuChoice != "" {
			existing.MenuChoice = &req.MenuChoice
		}
		if req.Allergies != "" {
			existing.Allergies = &req.Allergies
		}
		if req.AccessibilityNeeds != "" {
			existing.AccessibilityNeeds = &req.AccessibilityNeeds
		}
		if req.Transportation != "" {
			existing.Transportation = &req.Transportation
		}
		if req.Notes != "" {
			existing.Notes = &req.Notes
		}

		if err := s.rsvpRepo.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("upsert rsvp: update: %w", err)
		}

		_ = s.invitationRepo.UpdateStatus(ctx, invitation.ID, invitation.TenantID, domain.InvitationStatusResponded)
		_ = s.auditSvc.LogWithUser(ctx, officerID, tenantID, domain.AuditActionUpdate, domain.EntityTypeRSVP, existing.ID, nil, map[string]interface{}{
			"status":        req.Status,
			"attending_pax": req.AttendingPax,
			"guest_id":      guestID.String(),
			"mode":          "upsert",
		})
		return existing, nil
	}

	rsvp := &domain.RSVPResponse{
		Base:              domain.NewBase(),
		TenantID:          tenantID,
		EventID:           eventID,
		InvitationID:      invitation.ID,
		GuestID:           guestID,
		EventGuestID:      &eventGuest.ID,
		Status:            req.Status,
		AttendingPax:      req.AttendingPax,
		Adults:            req.Adults,
		Children:          req.Children,
		AttendingSessions: req.AttendingSessions,
		RespondedAt:       &now,
		EditedAt:          &now,
		EditedBy:          &officerID,
	}

	if req.MenuChoice != "" {
		rsvp.MenuChoice = &req.MenuChoice
	}
	if req.Allergies != "" {
		rsvp.Allergies = &req.Allergies
	}
	if req.AccessibilityNeeds != "" {
		rsvp.AccessibilityNeeds = &req.AccessibilityNeeds
	}
	if req.Transportation != "" {
		rsvp.Transportation = &req.Transportation
	}
	if req.Notes != "" {
		rsvp.Notes = &req.Notes
	}

	if err := s.rsvpRepo.Create(ctx, rsvp); err != nil {
		return nil, fmt.Errorf("upsert rsvp: create: %w", err)
	}

	_ = s.invitationRepo.UpdateStatus(ctx, invitation.ID, invitation.TenantID, domain.InvitationStatusResponded)
	_ = s.auditSvc.LogWithUser(ctx, officerID, tenantID, domain.AuditActionCreate, domain.EntityTypeRSVP, rsvp.ID, nil, map[string]interface{}{
		"status":        req.Status,
		"attending_pax": req.AttendingPax,
		"guest_id":      guestID.String(),
		"mode":          "upsert",
	})

	return rsvp, nil
}

// GetByInvitation retrieves an RSVP by invitation ID.
// Returns nil if no RSVP exists for the invitation.
func (s *RSVPService) GetByInvitation(ctx context.Context, tenantID, eventID, invitationID uuid.UUID) (*domain.RSVPResponse, error) {
	rsvp, err := s.rsvpRepo.GetByInvitation(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	// Verify tenant and event match
	if rsvp.TenantID != tenantID || rsvp.EventID != eventID {
		return nil, domain.ErrNotFound
	}
	return rsvp, nil
}

// ValidateDeadline ensures the RSVP deadline for an event hasn't passed.
func (s *RSVPService) ValidateDeadline(ctx context.Context, eventID uuid.UUID) error {
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("validate deadline: %w", err)
	}

	if event.RSVPDeadline != nil && time.Now().UTC().After(*event.RSVPDeadline) {
		return domain.ErrRSVPDeadlinePassed
	}

	return nil
}

// ValidateCapacity ensures total attending pax does not exceed event capacity.
func (s *RSVPService) ValidateCapacity(ctx context.Context, eventID, tenantID uuid.UUID, requestedPax int, excludeInvitationID uuid.UUID) error {
	event, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID)
	if err != nil {
		return fmt.Errorf("validate capacity: %w", err)
	}

	// If no capacity is set, no limit.
	if event.Capacity == nil || *event.Capacity <= 0 {
		return nil
	}

	// Get current total attending pax.
	currentTotal, err := s.rsvpRepo.SumAttendingPax(ctx, eventID)
	if err != nil {
		return fmt.Errorf("validate capacity: %w", err)
	}

	// If updating an existing RSVP, find its current attending pax to adjust the total.
	var existingPax int
	if excludeInvitationID != uuid.Nil {
		rsvp, err := s.rsvpRepo.GetByInvitation(ctx, excludeInvitationID)
		if err == nil && rsvp.Status == domain.RSVPStatusAttending {
			existingPax = rsvp.AttendingPax
		}
	}

	// newTotal = currentTotal - existingPax + requestedPax
	newTotal := currentTotal - existingPax + requestedPax

	if newTotal > *event.Capacity {
		return fmt.Errorf("validate capacity: %w", domain.ErrEventAtCapacity)
	}

	return nil
}

// validateTokenForRSVP validates a token for RSVP purposes.
func (s *RSVPService) validateTokenForRSVP(ctx context.Context, token string) (*domain.Invitation, error) {
	if token == "" {
		return nil, fmt.Errorf("empty token: %w", domain.ErrTokenInvalid)
	}

	tokenHash := crypto.SHA256Hash(token)

	invitation, err := s.invitationRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrInvitationNotFound) {
			return nil, domain.ErrTokenInvalid
		}
		return nil, err
	}

	if invitation.Status == domain.InvitationStatusRevoked {
		return nil, domain.ErrInvitationRevoked
	}

	if invitation.DeletedAt != nil {
		return nil, domain.ErrInvitationNotFound
	}

	if invitation.ExpiresAt != nil && time.Now().UTC().After(*invitation.ExpiresAt) {
		return nil, domain.ErrInvitationExpired
	}

	return invitation, nil
}

// isValidRSVPSubmitStatus checks if the status is valid for public submission.
func isValidRSVPSubmitStatus(status string) bool {
	switch status {
	case domain.RSVPStatusAttending, domain.RSVPStatusNotAttending, domain.RSVPStatusMaybe:
		return true
	}
	return false
}
