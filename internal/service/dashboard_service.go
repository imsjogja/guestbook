// Package service provides business logic for dashboard operations.
package service

import (
	"context"
	"fmt"
	"time"

	"guestflow/internal/domain"
	"guestflow/internal/repository"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// DashboardService aggregates data from multiple repositories to build the event dashboard.
type DashboardService struct {
	db          *sqlx.DB
	eventRepo   *repository.EventRepository
	rsvpRepo    *repository.RSVPRepository
	checkinRepo *repository.CheckinRepository
	commRepo    *repository.CommunicationRepository
	seatRepo    *repository.SeatingRepository
}

// NewDashboardService creates a new DashboardService.
func NewDashboardService(
	db *sqlx.DB,
	eventRepo *repository.EventRepository,
	rsvpRepo *repository.RSVPRepository,
	checkinRepo *repository.CheckinRepository,
	commRepo *repository.CommunicationRepository,
	seatRepo *repository.SeatingRepository,
) *DashboardService {
	return &DashboardService{
		db:          db,
		eventRepo:   eventRepo,
		rsvpRepo:    rsvpRepo,
		checkinRepo: checkinRepo,
		commRepo:    commRepo,
		seatRepo:    seatRepo,
	}
}

// GetEventDashboard builds the full dashboard for an event by aggregating data from multiple sources.
func (s *DashboardService) GetEventDashboard(ctx context.Context, tenantID, eventID uuid.UUID) (*domain.EventDashboard, error) {
	now := time.Now().UTC()

	// Get event summary
	event, err := s.eventRepo.GetByIDForTenant(ctx, eventID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	eventSummary := domain.EventSummary{
		ID:        event.ID,
		Name:      event.Name,
		Type:      event.Type,
		Status:    event.Status,
		StartDate: event.StartDate,
		Capacity:  event.Capacity,
	}
	if event.TargetInvites != nil {
		eventSummary.TargetInvites = event.TargetInvites
	}

	// Get RSVP stats
	rsvpDashboard, err := s.rsvpRepo.GetDashboardStats(ctx, tenantID, eventID, event.Capacity)
	if err != nil {
		return nil, fmt.Errorf("get rsvp stats: %w", err)
	}

	// Get checkin stats
	checkinStats, err := s.getCheckinStats(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get checkin stats: %w", err)
	}

	// Get communication stats
	commStats, err := s.getCommStats(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get communication stats: %w", err)
	}

	// Get seating stats
	seatingStats, err := s.getSeatingStats(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get seating stats: %w", err)
	}

	// Get recent activity
	recent, err := s.getRecentActivity(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get recent activity: %w", err)
	}

	dashboard := &domain.EventDashboard{
		Event:         eventSummary,
		RSVP:          *rsvpDashboard,
		Checkin:       checkinStats,
		Communication: commStats,
		Seating:       seatingStats,
		Recent:        recent,
		UpdatedAt:     now,
	}

	return dashboard, nil
}

// getCheckinStats aggregates check-in statistics for the dashboard.
func (s *DashboardService) getCheckinStats(ctx context.Context, tenantID, eventID uuid.UUID) (domain.CheckinStats, error) {
	var stats domain.CheckinStats

	// Get total expected from the active event roster. RSVP is a response state,
	// not the source of truth for who belongs to the event.
	var totalExpected int
	query := `
		SELECT COUNT(*)
		FROM event_guests
		WHERE tenant_id = $1 AND event_id = $2
		  AND status = 'active' AND deleted_at IS NULL
	`
	if err := s.db.GetContext(ctx, &totalExpected, query, tenantID, eventID); err != nil {
		return stats, fmt.Errorf("count expected: %w", err)
	}
	stats.TotalExpected = totalExpected

	// Get total checked in (unique guests)
	var totalCheckedIn int
	query = `
		SELECT COUNT(DISTINCT c.guest_id) FROM checkins c
		WHERE c.tenant_id = $1 AND c.event_id = $2 AND c.status = 'success' AND c.deleted_at IS NULL
		  AND EXISTS (SELECT 1 FROM event_guests eg WHERE eg.event_id = c.event_id AND eg.guest_id = c.guest_id AND eg.deleted_at IS NULL AND eg.status = 'active')
	`
	if err := s.db.GetContext(ctx, &totalCheckedIn, query, tenantID, eventID); err != nil {
		return stats, fmt.Errorf("count checked in: %w", err)
	}
	stats.TotalCheckedIn = totalCheckedIn

	// Get total pax (actual people checked in)
	var totalPax int
	query = `
		SELECT COALESCE(SUM(c.actual_pax), 0) FROM checkins c
		WHERE c.tenant_id = $1 AND c.event_id = $2 AND c.status = 'success' AND c.deleted_at IS NULL
		  AND EXISTS (SELECT 1 FROM event_guests eg WHERE eg.event_id = c.event_id AND eg.guest_id = c.guest_id AND eg.deleted_at IS NULL AND eg.status = 'active')
	`
	if err := s.db.GetContext(ctx, &totalPax, query, tenantID, eventID); err != nil {
		return stats, fmt.Errorf("count total pax: %w", err)
	}
	stats.TotalPax = totalPax

	// Get walk-ins count
	var walkIns int
	query = `
		SELECT COUNT(DISTINCT c.guest_id) FROM checkins c
		WHERE c.tenant_id = $1 AND c.event_id = $2 AND c.method = 'walk_in' AND c.deleted_at IS NULL
		  AND EXISTS (SELECT 1 FROM event_guests eg WHERE eg.event_id = c.event_id AND eg.guest_id = c.guest_id AND eg.deleted_at IS NULL AND eg.status = 'active')
	`
	if err := s.db.GetContext(ctx, &walkIns, query, tenantID, eventID); err != nil {
		return stats, fmt.Errorf("count walk-ins: %w", err)
	}
	stats.WalkIns = walkIns

	// Calculate no-shows
	if stats.TotalExpected > stats.TotalPax {
		stats.NoShows = stats.TotalExpected - stats.TotalPax
	}

	// Calculate check-in rate
	if stats.TotalExpected > 0 {
		stats.CheckInRate = float64(stats.TotalPax) / float64(stats.TotalExpected) * 100
	}

	// Get recent checkins
	recentCheckins, err := s.checkinRepo.GetRecent(ctx, tenantID, eventID, 10)
	if err != nil {
		return stats, fmt.Errorf("get recent checkins: %w", err)
	}
	stats.RecentCheckins = recentCheckins

	// Get by gate
	gateStats, err := s.checkinRepo.CountByGate(ctx, tenantID, eventID)
	if err != nil {
		return stats, fmt.Errorf("get gate stats: %w", err)
	}
	stats.ByGate = gateStats

	// Get by method
	methodStats, err := s.checkinRepo.CountByMethod(ctx, tenantID, eventID)
	if err != nil {
		return stats, fmt.Errorf("get method stats: %w", err)
	}
	stats.ByMethod = methodStats

	// Get peak hour
	peakHour, err := s.checkinRepo.GetPeakHour(ctx, tenantID, eventID)
	if err != nil {
		return stats, fmt.Errorf("get peak hour: %w", err)
	}
	stats.PeakHour = peakHour

	return stats, nil
}

// getCommStats aggregates communication statistics for the dashboard.
func (s *DashboardService) getCommStats(ctx context.Context, tenantID, eventID uuid.UUID) (domain.CommStats, error) {
	var stats domain.CommStats

	// Count total sent messages
	var totalSent int
	query := `
		SELECT COUNT(*) FROM communication_messages
		WHERE tenant_id = $1 AND event_id = $2 AND status IN ('sent', 'delivered', 'read')
	`
	if err := s.db.GetContext(ctx, &totalSent, query, tenantID, eventID); err != nil {
		return stats, fmt.Errorf("count sent messages: %w", err)
	}
	stats.TotalSent = totalSent

	// Count delivered messages
	var totalDelivered int
	query = `
		SELECT COUNT(*) FROM communication_messages
		WHERE tenant_id = $1 AND event_id = $2 AND status IN ('delivered', 'read')
	`
	if err := s.db.GetContext(ctx, &totalDelivered, query, tenantID, eventID); err != nil {
		return stats, fmt.Errorf("count delivered: %w", err)
	}
	stats.TotalDelivered = totalDelivered

	// Count failed messages
	var totalFailed int
	query = `
		SELECT COUNT(*) FROM communication_messages
		WHERE tenant_id = $1 AND event_id = $2 AND status = 'failed'
	`
	if err := s.db.GetContext(ctx, &totalFailed, query, tenantID, eventID); err != nil {
		return stats, fmt.Errorf("count failed: %w", err)
	}
	stats.TotalFailed = totalFailed

	// Count active templates for tenant
	var templatesUsed int
	query = `
		SELECT COUNT(*) FROM communication_templates
		WHERE tenant_id = $1 AND is_active = true AND deleted_at IS NULL
	`
	if err := s.db.GetContext(ctx, &templatesUsed, query, tenantID); err != nil {
		return stats, fmt.Errorf("count templates: %w", err)
	}
	stats.TemplatesUsed = templatesUsed

	// Count active campaigns for event
	activeCampaigns, err := s.commRepo.GetActiveCampaignsCount(ctx, tenantID, eventID)
	if err != nil {
		return stats, fmt.Errorf("count active campaigns: %w", err)
	}
	stats.CampaignsActive = activeCampaigns

	return stats, nil
}

// getSeatingStats aggregates seating statistics for the dashboard.
func (s *DashboardService) getSeatingStats(ctx context.Context, tenantID, eventID uuid.UUID) (domain.SeatingStats, error) {
	var stats domain.SeatingStats

	// Get total tables and seats
	var tableStats struct {
		TotalTables int `db:"total_tables"`
		TotalSeats  int `db:"total_seats"`
	}
	query := `
		SELECT
			COUNT(*) as total_tables,
			COALESCE(SUM(capacity), 0) as total_seats
		FROM tables
		WHERE tenant_id = $1 AND event_id = $2 AND deleted_at IS NULL
	`
	if err := s.db.GetContext(ctx, &tableStats, query, tenantID, eventID); err != nil {
		return stats, fmt.Errorf("count tables: %w", err)
	}
	stats.TotalTables = tableStats.TotalTables
	stats.TotalSeats = tableStats.TotalSeats

	// Get occupied seats
	var occupiedSeats int
	query = `
		SELECT COUNT(*) FROM seat_assignments sa
		JOIN tables t ON sa.table_id = t.id
		WHERE t.tenant_id = $1 AND t.event_id = $2 AND t.deleted_at IS NULL
		  AND EXISTS (SELECT 1 FROM event_guests eg WHERE eg.event_id = t.event_id AND eg.guest_id = sa.guest_id AND eg.deleted_at IS NULL AND eg.status = 'active')
	`
	if err := s.db.GetContext(ctx, &occupiedSeats, query, tenantID, eventID); err != nil {
		return stats, fmt.Errorf("count occupied seats: %w", err)
	}
	stats.OccupiedSeats = occupiedSeats

	// Get total attending guests
	var totalAttending int
	query = `
		SELECT COALESCE(SUM(attending_pax), 0)
		FROM rsvp_responses
		WHERE tenant_id = $1 AND event_id = $2 AND status = 'attending'
	`
	if err := s.db.GetContext(ctx, &totalAttending, query, tenantID, eventID); err != nil {
		return stats, fmt.Errorf("count attending guests: %w", err)
	}

	// Unseated = attending guests - occupied seats
	if totalAttending > occupiedSeats {
		stats.UnseatedGuests = totalAttending - occupiedSeats
	}

	// Occupancy rate
	if stats.TotalSeats > 0 {
		stats.OccupancyRate = float64(stats.OccupiedSeats) / float64(stats.TotalSeats) * 100
	}

	return stats, nil
}

// getRecentActivity gathers recent checkins, RSVPs, and messages.
func (s *DashboardService) getRecentActivity(ctx context.Context, tenantID, eventID uuid.UUID) (domain.RecentActivity, error) {
	var recent domain.RecentActivity

	// Recent checkins
	recentCheckins, err := s.checkinRepo.GetRecent(ctx, tenantID, eventID, 5)
	if err != nil {
		return recent, fmt.Errorf("get recent checkins: %w", err)
	}
	recent.Checkins = recentCheckins

	// Recent RSVPs
	recentRSVPs, err := s.getRecentRSVPs(ctx, tenantID, eventID, 5)
	if err != nil {
		return recent, fmt.Errorf("get recent rsvps: %w", err)
	}
	recent.RSVPs = recentRSVPs

	// Recent messages
	recentMessages, err := s.commRepo.GetRecentMessages(ctx, tenantID, eventID, 5)
	if err != nil {
		return recent, fmt.Errorf("get recent messages: %w", err)
	}
	recent.Messages = recentMessages

	return recent, nil
}

// getRecentRSVPs returns the most recent RSVP responses.
func (s *DashboardService) getRecentRSVPs(ctx context.Context, tenantID, eventID uuid.UUID, limit int) ([]domain.RSVPResponse, error) {
	query := `
		SELECT * FROM rsvp_responses
		WHERE tenant_id = $1 AND event_id = $2
		ORDER BY responded_at DESC NULLS LAST, created_at DESC
		LIMIT $3
	`
	var rsvps []domain.RSVPResponse
	if err := s.db.SelectContext(ctx, &rsvps, query, tenantID, eventID, limit); err != nil {
		return nil, fmt.Errorf("query recent rsvps: %w", err)
	}
	return rsvps, nil
}
