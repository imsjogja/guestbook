package domain

import (
	"time"

	"github.com/google/uuid"
)

// EventDashboard represents the full dashboard for an event
type EventDashboard struct {
	Event         EventSummary   `json:"event"`
	RSVP          RSVPDashboard  `json:"rsvp"`
	Checkin       CheckinStats   `json:"checkin"`
	Communication CommStats      `json:"communication"`
	Seating       SeatingStats   `json:"seating"`
	Recent        RecentActivity `json:"recent_activity"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// EventSummary represents a condensed event view for the dashboard
type EventSummary struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	StartDate     time.Time `json:"start_date"`
	Capacity      *int      `json:"capacity,omitempty"`
	TargetInvites *int      `json:"target_invites,omitempty"`
}

// CommStats represents communication statistics for the dashboard
type CommStats struct {
	TotalSent       int `json:"total_sent"`
	TotalDelivered  int `json:"total_delivered"`
	TotalFailed     int `json:"total_failed"`
	TemplatesUsed   int `json:"templates_used"`
	CampaignsActive int `json:"campaigns_active"`
}

// SeatingStats represents seating statistics for the dashboard
type SeatingStats struct {
	TotalTables    int     `json:"total_tables"`
	TotalSeats     int     `json:"total_seats"`
	OccupiedSeats  int     `json:"occupied_seats"`
	UnseatedGuests int     `json:"unseated_guests"`
	OccupancyRate  float64 `json:"occupancy_rate"`
}

// RecentActivity represents recent activity on the dashboard
type RecentActivity struct {
	Checkins []Checkin              `json:"checkins,omitempty"`
	RSVPs    []RSVPResponse         `json:"rsvps,omitempty"`
	Messages []CommunicationMessage `json:"messages,omitempty"`
}
