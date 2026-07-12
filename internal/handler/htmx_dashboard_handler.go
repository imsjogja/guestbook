// internal/handler/htmx_dashboard_handler.go
//
// HTMX-powered dashboard fragments for real-time admin UI updates.
// These endpoints return HTML partials (not JSON) designed to be swapped
// into the admin dashboard via HTMX hx-get requests.
package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"guestflow/internal/domain"
	"guestflow/internal/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ---------------------------------------------------------------------------
// HTMXDashboardHandler serves HTML fragments for HTMX-powered dashboard.
// ---------------------------------------------------------------------------
type HTMXDashboardHandler struct {
	dashboardService *service.DashboardService
	checkinService   *service.CheckinService
	guestService     *service.GuestService
	rsvpService      *service.RSVPService
	renderer         *TemplateRenderer
}

// NewHTMXDashboardHandler creates a new HTMX dashboard handler.
func NewHTMXDashboardHandler(
	dashboardService *service.DashboardService,
	checkinService *service.CheckinService,
	guestService *service.GuestService,
	rsvpService *service.RSVPService,
	renderer *TemplateRenderer,
) *HTMXDashboardHandler {
	return &HTMXDashboardHandler{
		dashboardService: dashboardService,
		checkinService:   checkinService,
		guestService:     guestService,
		rsvpService:      rsvpService,
		renderer:         renderer,
	}
}

// ---------------------------------------------------------------------------
// View models for partial templates
// ---------------------------------------------------------------------------

type statCardVM struct {
	Title           string
	Value           string
	Icon            string
	Color           string
	Change          string
	ChangeDirection string
	Label           string
	Badge           string
	BadgeColor      string
	ProgressValue   int
	ProgressColor   string
}

type checkinRowVM struct {
	GuestName          string
	Initials           string
	GuestType          string
	CheckedAtFormatted string
	Gate               string
	ActualPax          int
	Status             string
}

type rsvpStatusVM struct {
	Label   string
	Count   int
	Percent float64
	Color   string
	Pax     string
}

type activityItemVM struct {
	Type       string
	Icon       string
	IconBg     string
	Message    template.HTML
	TimeAgo    string
	Badge      string
	BadgeColor string
}

type guestSearchResultVM struct {
	ID        string
	FullName  string
	Initials  string
	Phone     string
	GuestType string
	CheckedIn bool
	MaxPax    int
}

// ---------------------------------------------------------------------------
// Routes registration
// ---------------------------------------------------------------------------

// RegisterHTMXRoutes registers HTMX fragment routes.
// These routes are protected by JWT auth and return HTML partials.
func (h *HTMXDashboardHandler) RegisterHTMXRoutes(
	e *echo.Echo,
	jwtAuth echo.MiddlewareFunc,
	enforceTenant echo.MiddlewareFunc,
) {
	htmx := e.Group("/htmx")
	htmx.Use(jwtAuth)

	// Dashboard fragment routes - scoped to tenant and event
	dash := htmx.Group("/dashboard/:tenantId/:eventId")
	dash.Use(enforceTenant)

	dash.GET("/stats", h.GetStatsCards)
	dash.GET("/checkins", h.GetRecentCheckinsFragment)
	dash.GET("/rsvp", h.GetRSVPBreakdownFragment)
	dash.GET("/activity", h.GetActivityFeedFragment)
	dash.GET("/guest-search", h.SearchGuestsFragment)
}

// ---------------------------------------------------------------------------
// GET /htmx/dashboard/:tenantId/:eventId/stats
// Returns: HTML partial of stat cards
// ---------------------------------------------------------------------------
func (h *HTMXDashboardHandler) GetStatsCards(c echo.Context) error {
	tenantID, eventID, err := h.parseIDs(c)
	if err != nil {
		return c.HTML(http.StatusBadRequest, errorCard("Invalid parameters"))
	}

	dashboard, err := h.dashboardService.GetEventDashboard(c.Request().Context(), tenantID, eventID)
	if err != nil {
		return c.HTML(http.StatusOK, errorCard("Failed to load stats"))
	}

	// Build capacity-based progress
	capacity := 0
	if dashboard.Event.Capacity != nil {
		capacity = *dashboard.Event.Capacity
	}

	attendingProgress := 0
	if capacity > 0 && dashboard.RSVP.AttendingPax > 0 {
		attendingProgress = minInt(100, int(float64(dashboard.RSVP.AttendingPax)/float64(capacity)*100))
	}

	checkinProgress := 0
	if dashboard.RSVP.AttendingPax > 0 && dashboard.Checkin.TotalPax > 0 {
		checkinProgress = minInt(100, int(float64(dashboard.Checkin.TotalPax)/float64(dashboard.RSVP.AttendingPax)*100))
	}

	cards := []statCardVM{
		{
			Title:           "Total Undangan",
			Value:           formatInt(dashboard.RSVP.TotalInvited),
			Icon:            "💌",
			Color:           "purple",
			Change:          fmt.Sprintf("%d terkirim", dashboard.RSVP.TotalSent),
			ChangeDirection: "up",
			Label:           "dari total",
		},
		{
			Title:           "RSVP Hadir",
			Value:           fmt.Sprintf("%d (%d org)", dashboard.RSVP.Attending, dashboard.RSVP.AttendingPax),
			Icon:            "✅",
			Color:           "green",
			Change:          fmt.Sprintf("%d%%", attendingProgress),
			ChangeDirection: "up",
			Label:           "dari kapasitas",
			ProgressValue:   attendingProgress,
			ProgressColor:   "green",
		},
		{
			Title:           "Check-in",
			Value:           fmt.Sprintf("%d org", dashboard.Checkin.TotalPax),
			Icon:            "📱",
			Color:           "amber",
			Change:          "Live",
			ChangeDirection: "up",
			Label:           "real-time",
			ProgressValue:   checkinProgress,
			ProgressColor:   "amber",
		},
		{
			Title:  "Belum Respons",
			Value:  formatInt(dashboard.RSVP.NoResponse),
			Icon:   "⏳",
			Color:  "rose",
			Label:  "follow-up",
			Badge:  fmt.Sprintf("%.0f%%", dashboard.RSVP.ResponseRate),
			BadgeColor: "gray",
		},
		{
			Title:  "Walk-in",
			Value:  formatInt(dashboard.Checkin.WalkIns),
			Icon:   "🚶",
			Color:  "blue",
			Badge:  "Hari ini",
			BadgeColor: "gray",
		},
	}

	return c.Render(http.StatusOK, "partials/stats_cards.html", cards)
}

// ---------------------------------------------------------------------------
// GET /htmx/dashboard/:tenantId/:eventId/checkins
// Returns: HTML partial of recent check-ins table
// ---------------------------------------------------------------------------
func (h *HTMXDashboardHandler) GetRecentCheckinsFragment(c echo.Context) error {
	tenantID, eventID, err := h.parseIDs(c)
	if err != nil {
		return c.HTML(http.StatusBadRequest, errorCard("Invalid parameters"))
	}

	limit := 10
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	checkins, err := h.checkinService.GetRecent(c.Request().Context(), tenantID, eventID, limit)
	if err != nil {
		return c.HTML(http.StatusOK, errorCard("Failed to load check-ins"))
	}

	// Build a simple guest lookup map for names
	guestMap := make(map[uuid.UUID]string)
	for _, ci := range checkins {
		if _, ok := guestMap[ci.GuestID]; !ok {
			if guest, err := h.guestService.Get(c.Request().Context(), tenantID, ci.GuestID); err == nil && guest != nil {
				guestMap[ci.GuestID] = guest.FullName
			}
		}
	}

	rows := make([]checkinRowVM, 0, len(checkins))
	for _, ci := range checkins {
		guestName := guestMap[ci.GuestID]
		if guestName == "" {
			guestName = "Tamu " + ci.GuestID.String()[:8]
		}

		// Derive gate display from GateID
		gateDisplay := "Default"
		if ci.GateID != nil {
			gateDisplay = "Gate " + ci.GateID.String()[:4]
		}

		// Use CreatedAt as CheckedAt fallback
		checkedAt := &ci.CreatedAt

		rows = append(rows, checkinRowVM{
			GuestName:          guestName,
			Initials:           getInitials(guestName),
			GuestType:          "",
			CheckedAtFormatted: formatTimeAgo(checkedAt),
			Gate:               gateDisplay,
			ActualPax:          ci.ActualPax,
			Status:             ci.Status,
		})
	}

	return c.Render(http.StatusOK, "partials/recent_checkins.html", map[string]interface{}{
		"Checkins": rows,
	})
}

// ---------------------------------------------------------------------------
// GET /htmx/dashboard/:tenantId/:eventId/rsvp
// Returns: HTML partial of RSVP breakdown
// ---------------------------------------------------------------------------
func (h *HTMXDashboardHandler) GetRSVPBreakdownFragment(c echo.Context) error {
	tenantID, eventID, err := h.parseIDs(c)
	if err != nil {
		return c.HTML(http.StatusBadRequest, errorCard("Invalid parameters"))
	}

	dashboard, err := h.dashboardService.GetEventDashboard(c.Request().Context(), tenantID, eventID)
	if err != nil {
		return c.HTML(http.StatusOK, errorCard("Failed to load RSVP data"))
	}

	statuses := []rsvpStatusVM{
		{Label: "Hadir", Count: dashboard.RSVP.Attending, Percent: 0, Color: "#10b981", Pax: fmt.Sprintf("%d orang", dashboard.RSVP.AttendingPax)},
		{Label: "Tidak Hadir", Count: dashboard.RSVP.NotAttending, Percent: 0, Color: "#f43f5e"},
		{Label: "Tentatif", Count: dashboard.RSVP.Maybe, Percent: 0, Color: "#f59e0b"},
		{Label: "Belum Respons", Count: dashboard.RSVP.NoResponse, Percent: 0, Color: "#94a3b8"},
	}

	// Calculate percentages
	total := dashboard.RSVP.Attending + dashboard.RSVP.NotAttending + dashboard.RSVP.Maybe + dashboard.RSVP.NoResponse + dashboard.RSVP.Waitlist
	if total > 0 {
		for i := range statuses {
			statuses[i].Percent = roundFloat(float64(statuses[i].Count)/float64(total)*100, 1)
		}
	}

	return c.Render(http.StatusOK, "partials/rsvp_breakdown.html", map[string]interface{}{
		"Statuses":       statuses,
		"TotalResponses": total,
		"ResponseRate":   fmt.Sprintf("%.1f", dashboard.RSVP.ResponseRate),
	})
}

// ---------------------------------------------------------------------------
// GET /htmx/dashboard/:tenantId/:eventId/activity
// Returns: HTML partial of combined activity feed
// ---------------------------------------------------------------------------
func (h *HTMXDashboardHandler) GetActivityFeedFragment(c echo.Context) error {
	tenantID, eventID, err := h.parseIDs(c)
	if err != nil {
		return c.HTML(http.StatusBadRequest, errorCard("Invalid parameters"))
	}

	dashboard, err := h.dashboardService.GetEventDashboard(c.Request().Context(), tenantID, eventID)
	if err != nil {
		return c.HTML(http.StatusOK, errorCard("Failed to load activity"))
	}

	var activities []activityItemVM

	// Add check-in activities
	for _, ci := range dashboard.Recent.Checkins {
		// Look up guest name
		guestName := "Tamu"
		if guest, err := h.guestService.Get(c.Request().Context(), tenantID, ci.GuestID); err == nil && guest != nil {
			guestName = guest.FullName
		}

		methodLabel := ci.Method
		switch ci.Method {
		case "qr_scan":
			methodLabel = "QR Scan"
		case "manual_search":
			methodLabel = "Manual"
		case "walk_in":
			methodLabel = "Walk-in"
		case "kiosk":
			methodLabel = "Kiosk"
		}

		activities = append(activities, activityItemVM{
			Type:   "checkin",
			Icon:   "📱",
			IconBg: "rgba(16,185,129,0.1)",
			Message: template.HTML(fmt.Sprintf("<strong>%s</strong> check-in via %s (%d pax)",
				template.HTMLEscapeString(guestName),
				methodLabel,
				ci.ActualPax)),
			TimeAgo: formatTimeAgo(&ci.CreatedAt),
			Badge:   "Check-in",
			BadgeColor: "green",
		})
	}

	// Add RSVP activities
	for _, rsvp := range dashboard.Recent.RSVPs {
		var statusLabel, statusColor string
		switch rsvp.Status {
		case "attending":
			statusLabel, statusColor = "Hadir", "green"
		case "not_attending":
			statusLabel, statusColor = "Tidak Hadir", "rose"
		case "maybe":
			statusLabel, statusColor = "Tentatif", "amber"
		case "pending":
			statusLabel, statusColor = "Pending", "gray"
		default:
			statusLabel, statusColor = rsvp.Status, "gray"
		}

		guestName := "Tamu"
		if rsvp.GuestID != uuid.Nil {
			if guest, err := h.guestService.Get(c.Request().Context(), tenantID, rsvp.GuestID); err == nil && guest != nil {
				guestName = guest.FullName
			}
		}

		activities = append(activities, activityItemVM{
			Type:   "rsvp",
			Icon:   "📝",
			IconBg: "rgba(99,102,241,0.1)",
			Message: template.HTML(fmt.Sprintf("RSVP dari <strong>%s</strong>: %s (%d orang)",
				template.HTMLEscapeString(guestName),
				statusLabel,
				rsvp.AttendingPax)),
			TimeAgo: formatTimeAgo(rsvp.RespondedAt),
			Badge:   statusLabel,
			BadgeColor: statusColor,
		})
	}

	// Add message activities
	for _, msg := range dashboard.Recent.Messages {
		var icon string
		switch msg.Channel {
		case "whatsapp":
			icon = "💬"
		case "email":
			icon = "📧"
		case "sms":
			icon = "💬"
		default:
			icon = "📨"
		}

		guestName := "Tamu"
		if msg.GuestID != uuid.Nil {
			if guest, err := h.guestService.Get(c.Request().Context(), tenantID, msg.GuestID); err == nil && guest != nil {
				guestName = guest.FullName
			}
		}

		activities = append(activities, activityItemVM{
			Type:   "message",
			Icon:   icon,
			IconBg: "rgba(59,130,246,0.1)",
			Message: template.HTML(fmt.Sprintf("Pesan %s ke <strong>%s</strong>",
				msg.Channel,
				template.HTMLEscapeString(guestName))),
			TimeAgo: formatTimeAgo(msg.SentAt),
			Badge:   msg.Status,
			BadgeColor: messageStatusColor(msg.Status),
		})
	}

	// Limit to most recent 20 items
	if len(activities) > 20 {
		activities = activities[:20]
	}

	return c.Render(http.StatusOK, "partials/activity_feed.html", map[string]interface{}{
		"Activities": activities,
	})
}

// ---------------------------------------------------------------------------
// GET /htmx/dashboard/:tenantId/:eventId/guest-search
// Query: q (search term)
// Returns: HTML partial of guest search results for inline check-in
// ---------------------------------------------------------------------------
func (h *HTMXDashboardHandler) SearchGuestsFragment(c echo.Context) error {
	tenantID, eventID, err := h.parseIDs(c)
	if err != nil {
		return c.HTML(http.StatusBadRequest, errorCard("Invalid parameters"))
	}
	_ = eventID // reserved for future event-scoped search

	query := strings.TrimSpace(c.QueryParam("q"))
	if query == "" {
		return c.HTML(http.StatusOK, "")
	}

	// Search guests (tenant-scoped, not event-scoped)
	guests, err := h.guestService.Search(c.Request().Context(), tenantID, query)
	if err != nil {
		return c.HTML(http.StatusOK, errorCard("Search failed"))
	}

	// Limit to top 10 results
	if len(guests) > 10 {
		guests = guests[:10]
	}

	results := make([]guestSearchResultVM, 0, len(guests))
	for _, g := range guests {
		phone := ""
		if g.Phone != nil {
			phone = maskPhone(*g.Phone)
		}

		results = append(results, guestSearchResultVM{
			ID:        g.ID.String(),
			FullName:  g.FullName,
			Initials:  getInitials(g.FullName),
			Phone:     phone,
			GuestType: g.GuestType,
			CheckedIn: false, // TODO: lookup checkin status if needed
			MaxPax:    2,     // default
		})
	}

	return c.Render(http.StatusOK, "partials/guest_search_results.html", map[string]interface{}{
		"Guests": results,
		"Query":  query,
	})
}

// ==========================================================================
// Helpers
// ==========================================================================

func (h *HTMXDashboardHandler) parseIDs(c echo.Context) (tenantID uuid.UUID, eventID uuid.UUID, err error) {
	tenantID, err = uuid.Parse(c.Param("tenantId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid tenant ID: %w", err)
	}
	eventID, err = uuid.Parse(c.Param("eventId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid event ID: %w", err)
	}
	return tenantID, eventID, nil
}

func formatInt(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func formatTimeAgo(t *time.Time) string {
	if t == nil {
		return "-"
	}
	now := time.Now()
	diff := now.Sub(*t)

	switch {
	case diff < time.Minute:
		return "baru saja"
	case diff < time.Hour:
		return fmt.Sprintf("%d menit lalu", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d jam lalu", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%d hari lalu", int(diff.Hours()/24))
	default:
		return t.Format("2 Jan 2006")
	}
}

func getInitials(name string) string {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "?"
	}
	if len(parts) == 1 {
		if len(parts[0]) >= 2 {
			return strings.ToUpper(parts[0][:2])
		}
		return strings.ToUpper(parts[0])
	}
	return strings.ToUpper(string(parts[0][0]) + string(parts[len(parts)-1][0]))
}

func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return phone
	}
	return phone[:len(phone)-4] + "****"
}

func messageStatusColor(status string) string {
	switch status {
	case "sent", "delivered":
		return "green"
	case "read":
		return "blue"
	case "failed":
		return "rose"
	default:
		return "gray"
	}
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func roundFloat(f float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int(f*pow)) / pow
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func errorCard(msg string) string {
	return fmt.Sprintf(`<div style="padding: 20px; text-align: center; color: #f43f5e; font-size: 0.875rem;">%s</div>`, template.HTMLEscapeString(msg))
}
