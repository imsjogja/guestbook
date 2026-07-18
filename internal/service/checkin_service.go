// Package service provides business logic layer implementations for GuestFlow.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"guestflow/internal/audit"
	"guestflow/internal/domain"
	"guestflow/internal/repository"
	"guestflow/pkg/crypto"

	"github.com/google/uuid"
)

// CheckinService encapsulates business logic for check-in operations.
type CheckinService struct {
	checkinRepo    *repository.CheckinRepository
	guestRepo      *repository.GuestRepository
	invitationRepo *repository.InvitationRepository
	eventGuestRepo *repository.EventGuestRepository
	eventRepo      *repository.EventRepository
	seatingRepo    *repository.SeatingRepository
	auditSvc       *audit.Service
}

// NewCheckinService creates a new CheckinService.
func NewCheckinService(
	checkinRepo *repository.CheckinRepository,
	guestRepo *repository.GuestRepository,
	invitationRepo *repository.InvitationRepository,
	eventGuestRepo *repository.EventGuestRepository,
	eventRepo *repository.EventRepository,
	seatingRepo *repository.SeatingRepository,
	auditSvc *audit.Service,
) *CheckinService {
	return &CheckinService{
		checkinRepo:    checkinRepo,
		guestRepo:      guestRepo,
		invitationRepo: invitationRepo,
		eventGuestRepo: eventGuestRepo,
		eventRepo:      eventRepo,
		seatingRepo:    seatingRepo,
		auditSvc:       auditSvc,
	}
}

// ProcessCheckin handles the main check-in flow by dispatching to the appropriate sub-method.
func (s *CheckinService) ProcessCheckin(ctx context.Context, tenantID, eventID uuid.UUID, officerID *uuid.UUID, req domain.CheckinRequest) (*domain.Checkin, error) {
	switch req.Method {
	case domain.CheckinMethodQRScan:
		return s.ProcessQRScan(ctx, tenantID, eventID, officerID, req)
	case domain.CheckinMethodManual:
		return s.ProcessManualSearch(ctx, tenantID, eventID, officerID, req)
	case domain.CheckinMethodWalkin:
		return nil, fmt.Errorf("walk-in check-in should use ProcessWalkin: %w", domain.ErrInvalidInput)
	default:
		return nil, fmt.Errorf("unsupported check-in method: %w", domain.ErrInvalidInput)
	}
}

// ProcessQRScan processes a QR code scan check-in.
func (s *CheckinService) ProcessQRScan(ctx context.Context, tenantID, eventID uuid.UUID, officerID *uuid.UUID, req domain.CheckinRequest) (*domain.Checkin, error) {
	token := strings.TrimSpace(req.Token)
	if token == "" {
		return nil, fmt.Errorf("QR token is required: %w", domain.ErrInvalidInput)
	}

	// In a real implementation, the token would be hashed and looked up in a credentials table.
	// For now, we attempt to find the guest by treating the token as a guest code reference.
	// The credential lookup would map the token hash to a guest_id and invitation_id.
	// This is a simplified flow - production would have a separate credential validation step.

	guest, invitation, err := s.findGuestByToken(ctx, tenantID, eventID, token)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return s.recordFailedCheckin(ctx, tenantID, eventID, officerID, req, domain.CheckinStatusInvalid)
		}
		return nil, fmt.Errorf("process qr scan: %w", err)
	}

	var invitationID *uuid.UUID
	if invitation != nil {
		invitationID = &invitation.ID
	}
	return s.performCheckin(ctx, tenantID, eventID, guest.ID, invitationID, officerID, req)
}

// ProcessManualSearch processes a manual search check-in by guest ID.
func (s *CheckinService) ProcessManualSearch(ctx context.Context, tenantID, eventID uuid.UUID, officerID *uuid.UUID, req domain.CheckinRequest) (*domain.Checkin, error) {
	if req.GuestID == nil || *req.GuestID == uuid.Nil {
		return nil, fmt.Errorf("guest_id is required for manual check-in: %w", domain.ErrInvalidInput)
	}

	// Verify guest exists and belongs to tenant
	guest, err := s.guestRepo.GetByIDForTenant(ctx, tenantID, *req.GuestID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return s.recordFailedCheckin(ctx, tenantID, eventID, officerID, req, domain.CheckinStatusInvalid)
		}
		return nil, fmt.Errorf("process manual check-in: %w", err)
	}
	if _, err := s.eventGuestRepo.GetByEventAndGuest(ctx, tenantID, eventID, guest.ID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return s.recordFailedCheckin(ctx, tenantID, eventID, officerID, req, domain.CheckinStatusWrongEvent)
		}
		return nil, fmt.Errorf("process manual check-in: check event roster: %w", err)
	}

	return s.performCheckin(ctx, tenantID, eventID, guest.ID, nil, officerID, req)
}

// ProcessWalkin handles walk-in registration and check-in.
func (s *CheckinService) ProcessWalkin(ctx context.Context, tenantID, eventID uuid.UUID, officerID *uuid.UUID, req domain.WalkinRequest) (*domain.Checkin, error) {
	// Validate input
	req.FullName = strings.TrimSpace(req.FullName)
	if req.FullName == "" {
		return nil, fmt.Errorf("full_name is required: %w", domain.ErrInvalidInput)
	}
	if req.GuestType == "" {
		return nil, fmt.Errorf("guest_type is required: %w", domain.ErrInvalidInput)
	}

	// Create a new guest record for the walk-in
	// Walk-ins are created by the officer performing the check-in (or system)
	createdBy := uuid.Nil
	if officerID != nil {
		createdBy = *officerID
	}

	guestReq := domain.GuestCreateRequest{
		FullName:  req.FullName,
		GuestType: req.GuestType,
		Segment:   req.Segment,
	}

	if req.Phone != "" {
		guestReq.Phone = req.Phone
	}
	if req.Email != "" {
		guestReq.Email = req.Email
	}

	guest := domain.NewGuest(tenantID, createdBy, guestReq)
	if err := s.guestRepo.Create(ctx, guest); err != nil {
		return nil, fmt.Errorf("create walk-in guest: %w", err)
	}
	eventGuest := &domain.EventGuest{
		Base: domain.NewBase(), TenantID: tenantID, EventID: eventID, GuestID: guest.ID,
		Status: domain.EventGuestStatusActive, Source: domain.EventGuestSourceWalkIn,
		MaxPax: req.ActualPax, Adults: req.Adults, Children: req.Children, CreatedBy: createdBy,
	}
	if eventGuest.Adults == 0 && eventGuest.Children == 0 {
		eventGuest.Adults = req.ActualPax
	}
	if err := s.eventGuestRepo.Create(ctx, eventGuest); err != nil {
		return nil, fmt.Errorf("add walk-in guest to event: %w", err)
	}

	// Build check-in request from walk-in data
	checkinReq := domain.CheckinRequest{
		Method:         domain.CheckinMethodWalkin,
		ActualPax:      req.ActualPax,
		Adults:         req.Adults,
		Children:       req.Children,
		OverrideReason: req.OverrideReason,
		ApprovedBy:     req.ApprovedBy,
		Notes:          req.Notes,
	}

	return s.performCheckin(ctx, tenantID, eventID, guest.ID, nil, officerID, checkinReq)
}

// GetStats aggregates real-time check-in stats for an event.
func (s *CheckinService) GetStats(ctx context.Context, tenantID, eventID uuid.UUID) (*domain.CheckinStats, error) {
	// Total checked in (successful checkins)
	totalCheckedIn, err := s.checkinRepo.CountByEvent(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get checkin stats: %w", err)
	}

	// Total pax
	totalPax, err := s.checkinRepo.CountPaxByEvent(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get checkin stats: %w", err)
	}

	// Walk-ins
	walkIns, err := s.checkinRepo.CountWalkInsByEvent(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get checkin stats: %w", err)
	}

	// By gate
	byGate, err := s.checkinRepo.CountByGate(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get checkin stats: %w", err)
	}

	// By method
	byMethod, err := s.checkinRepo.CountByMethod(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get checkin stats: %w", err)
	}

	// Recent checkins
	recentCheckins, err := s.checkinRepo.GetRecent(ctx, tenantID, eventID, 20)
	if err != nil {
		return nil, fmt.Errorf("get checkin stats: %w", err)
	}

	// Peak hour
	peakHour, err := s.checkinRepo.GetPeakHour(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get checkin stats: %w", err)
	}

	// Total expected from invitations / guest list
	// For now we estimate from the guest list associated with the event
	totalExpected := s.estimateTotalExpected(ctx, tenantID, eventID)

	noShows := totalExpected - totalCheckedIn
	if noShows < 0 {
		noShows = 0
	}

	checkInRate := 0.0
	if totalExpected > 0 {
		checkInRate = float64(totalCheckedIn) / float64(totalExpected) * 100
	}

	return &domain.CheckinStats{
		TotalExpected:  totalExpected,
		TotalCheckedIn: totalCheckedIn,
		TotalPax:       totalPax,
		WalkIns:        walkIns,
		NoShows:        noShows,
		CheckInRate:    checkInRate,
		RecentCheckins: recentCheckins,
		ByGate:         byGate,
		ByMethod:       byMethod,
		PeakHour:       peakHour,
	}, nil
}

// SearchGuests searches for guests available for manual check-in.
// Sensitive data is masked for registration officers.
func (s *CheckinService) SearchGuests(ctx context.Context, tenantID, eventID uuid.UUID, query string, maskSensitive bool) ([]*domain.GuestSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query is required: %w", domain.ErrInvalidInput)
	}

	eventGuests, err := s.eventGuestRepo.List(ctx, domain.EventGuestListParams{
		TenantID: tenantID, EventID: eventID, Search: query,
		Status: domain.EventGuestStatusActive, Page: 1, PerPage: 50,
	})
	if err != nil {
		return nil, fmt.Errorf("search guests for checkin: %w", err)
	}

	results := make([]*domain.GuestSearchResult, 0, len(eventGuests))
	for _, eventGuest := range eventGuests {
		g := eventGuest.Guest
		if g == nil {
			continue
		}
		isCheckedIn, err := s.checkinRepo.IsCheckedIn(ctx, tenantID, eventID, g.ID)
		if err != nil {
			continue // Skip guests we can't verify
		}

		result := &domain.GuestSearchResult{
			GuestID:     g.ID,
			FullName:    g.FullName,
			Nickname:    g.Nickname,
			GuestType:   g.GuestType,
			Segment:     g.Segment,
			RSVPStatus:  "confirmed", // Default - would come from invitation in full implementation
			IsCheckedIn: isCheckedIn,
			MaxPax:      eventGuest.MaxPax,
		}

		// Mask sensitive data for registration officers
		if maskSensitive {
			if g.Phone != nil && len(*g.Phone) > 4 {
				masked := maskPhone(*g.Phone)
				result.Phone = &masked
			}
			if g.Email != nil && len(*g.Email) > 4 {
				masked := maskEmail(*g.Email)
				result.Email = &masked
			}
		} else {
			result.Phone = g.Phone
			result.Email = g.Email
		}

		results = append(results, result)
	}

	return results, nil
}

// GetRecent returns recent check-ins for an event.
func (s *CheckinService) GetRecent(ctx context.Context, tenantID, eventID uuid.UUID, limit int) ([]domain.Checkin, error) {
	return s.checkinRepo.GetRecent(ctx, tenantID, eventID, limit)
}

// ─── Internal Helpers ─────────────────────────────────────────────────────────

// performCheckin performs the actual check-in after all validations pass.
func (s *CheckinService) performCheckin(ctx context.Context, tenantID, eventID, guestID uuid.UUID, invitationID *uuid.UUID, officerID *uuid.UUID, req domain.CheckinRequest) (*domain.Checkin, error) {
	// Verify event exists
	if _, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID); err != nil {
		return nil, fmt.Errorf("perform checkin: event not found: %w", err)
	}
	eventGuest, err := s.eventGuestRepo.GetByEventAndGuest(ctx, tenantID, eventID, guestID)
	if err != nil {
		return nil, fmt.Errorf("perform checkin: guest is not in event roster: %w", domain.ErrInvalidInput)
	}

	// Check for duplicate check-in
	isCheckedIn, err := s.checkinRepo.IsCheckedIn(ctx, tenantID, eventID, guestID)
	if err != nil {
		return nil, fmt.Errorf("perform checkin: %w", err)
	}

	if isCheckedIn {
		// Record duplicate attempt but return existing checkin info
		return s.recordFailedCheckin(ctx, tenantID, eventID, officerID, req, domain.CheckinStatusDuplicate)
	}

	// Validate actual_pax
	actualPax := req.ActualPax
	if actualPax < 1 {
		actualPax = 1
	}

	adults := req.Adults
	if adults < 1 {
		adults = actualPax // Default: all are adults
	}
	children := req.Children
	if adults+children != actualPax {
		// Auto-balance: if adults + children != actual_pax, use actual_pax as adults
		adults = actualPax
		children = 0
	}

	var deviceIDPtr *string
	if req.DeviceID != "" {
		deviceIDPtr = &req.DeviceID
	}

	var notesPtr *string
	if req.Notes != "" {
		notesPtr = &req.Notes
	}

	var overrideReasonPtr *string
	if req.OverrideReason != "" {
		overrideReasonPtr = &req.OverrideReason
	}

	var latPtr, lonPtr *float64
	if req.Latitude != 0 {
		latPtr = &req.Latitude
	}
	if req.Longitude != 0 {
		lonPtr = &req.Longitude
	}

	checkin := &domain.Checkin{
		Base:           domain.NewBase(),
		TenantID:       tenantID,
		EventID:        eventID,
		GuestID:        guestID,
		EventGuestID:   &eventGuest.ID,
		InvitationID:   invitationID,
		Method:         req.Method,
		Status:         domain.CheckinStatusSuccess,
		DeviceID:       deviceIDPtr,
		GateID:         req.GateID,
		OfficerID:      officerID,
		ActualPax:      actualPax,
		Adults:         adults,
		Children:       children,
		OverrideReason: overrideReasonPtr,
		ApprovedBy:     req.ApprovedBy,
		Latitude:       latPtr,
		Longitude:      lonPtr,
		Notes:          notesPtr,
		OfflineSynced:  false,
	}

	if err := s.checkinRepo.Create(ctx, checkin); err != nil {
		return nil, fmt.Errorf("perform checkin: %w", err)
	}

	// Audit log
	actor := uuid.Nil
	if officerID != nil {
		actor = *officerID
	}
	_ = s.auditSvc.LogWithUser(ctx, actor, tenantID, domain.AuditActionCreate, domain.EntityTypeCheckin, checkin.ID, nil, map[string]interface{}{
		"guest_id":   guestID.String(),
		"event_id":   eventID.String(),
		"method":     req.Method,
		"actual_pax": actualPax,
		"status":     domain.CheckinStatusSuccess,
	})

	return checkin, nil
}

// recordFailedCheckin records a failed check-in attempt and returns an appropriate error.
func (s *CheckinService) recordFailedCheckin(ctx context.Context, tenantID, eventID uuid.UUID, officerID *uuid.UUID, req domain.CheckinRequest, status string) (*domain.Checkin, error) {
	checkin := &domain.Checkin{
		Base:          domain.NewBase(),
		TenantID:      tenantID,
		EventID:       eventID,
		Method:        req.Method,
		Status:        status,
		DeviceID:      nil,
		GateID:        req.GateID,
		OfficerID:     officerID,
		ActualPax:     0,
		OfflineSynced: false,
	}

	if req.GuestID != nil {
		checkin.GuestID = *req.GuestID
	}

	var deviceIDPtr *string
	if req.DeviceID != "" {
		deviceIDPtr = &req.DeviceID
		checkin.DeviceID = deviceIDPtr
	}

	// Don't fail the whole operation if we can't record the failed attempt
	_ = s.checkinRepo.Create(ctx, checkin)

	// Return appropriate error
	switch status {
	case domain.CheckinStatusDuplicate:
		return checkin, fmt.Errorf("guest already checked in: %w", domain.ErrAlreadyExists)
	case domain.CheckinStatusInvalid:
		return checkin, fmt.Errorf("invalid credential or guest not found: %w", domain.ErrInvalidInput)
	default:
		return checkin, fmt.Errorf("check-in failed with status %s: %w", status, domain.ErrInvalidInput)
	}
}

// findGuestByToken looks up a guest by a QR token.
// In production, this would hash the token and look it up in a credentials table.
func (s *CheckinService) findGuestByToken(ctx context.Context, tenantID, eventID uuid.UUID, token string) (*domain.Guest, *domain.Invitation, error) {
	if s.invitationRepo != nil {
		tokenHash := crypto.SHA256Hash(token)
		invitation, err := s.invitationRepo.GetByTokenHash(ctx, tokenHash)
		if err == nil && invitation != nil {
			if invitation.EventID != eventID {
				return nil, nil, fmt.Errorf("token belongs to another event: %w", domain.ErrInvalidInput)
			}
			if invitation.EventGuestID != nil {
				if _, rosterErr := s.eventGuestRepo.GetByID(ctx, tenantID, eventID, *invitation.EventGuestID); rosterErr != nil {
					return nil, nil, fmt.Errorf("invitation guest is not active in event: %w", domain.ErrInvalidInput)
				}
			}
			guest, guestErr := s.guestRepo.GetByIDForTenant(ctx, tenantID, invitation.GuestID)
			if guestErr == nil && guest != nil {
				return guest, invitation, nil
			}
		}
	}
	return nil, nil, domain.ErrNotFound
}

// estimateTotalExpected estimates the total expected attendance for an event.
func (s *CheckinService) estimateTotalExpected(ctx context.Context, tenantID, eventID uuid.UUID) int {
	total, err := s.eventGuestRepo.Count(ctx, domain.EventGuestListParams{
		TenantID: tenantID, EventID: eventID, Status: domain.EventGuestStatusActive,
	})
	if err != nil {
		return 0
	}
	return total
}

// maskPhone masks a phone number, showing only last 4 digits.
func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(phone)-4) + phone[len(phone)-4:]
}

// maskEmail masks an email address, showing only first 2 chars and domain.
func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "****"
	}
	local := parts[0]
	domain := parts[1]
	if len(local) <= 2 {
		return "**@" + domain
	}
	return local[:2] + strings.Repeat("*", len(local)-2) + "@" + domain
}
