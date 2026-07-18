// internal/handler/invitation_site_handler.go
//
// Guest-facing invitation microsite handler.
// This handler serves the public invitation pages that guests view
// when they open their invitation link. It renders HTML templates
// (not JSON) and handles RSVP form submission.
package handler

import (
	stderrors "errors"
	"html/template"
	"net/http"
	"time"

	"guestflow/internal/domain"
	"guestflow/internal/service"

	"github.com/labstack/echo/v4"
)

// InvitationSiteHandler serves the public invitation microsite.
type InvitationSiteHandler struct {
	invitationService *service.InvitationService
	rsvpService       *service.RSVPService
	eventService      *service.EventService
	guestService      *service.GuestService
}

// NewInvitationSiteHandler creates a new invitation site handler.
func NewInvitationSiteHandler(
	invitationService *service.InvitationService,
	rsvpService *service.RSVPService,
	eventService *service.EventService,
	guestService *service.GuestService,
) *InvitationSiteHandler {
	return &InvitationSiteHandler{
		invitationService: invitationService,
		rsvpService:       rsvpService,
		eventService:      eventService,
		guestService:      guestService,
	}
}

// invitationViewModel holds data for the invitation template.
type invitationViewModel struct {
	Language          string
	Event             eventViewModel
	Guest             *guestViewModel
	InvitationToken   string
	RSVPSubmitted     bool
	RSVP              *rsvpViewModel
	MaxPaxOptions     []int
	MenuOptions       []menuOption
	Sessions          []sessionViewModel
	GalleryImages     []galleryImage
	RSVPDeadline      string
	RSVPDeadlineISO   string
	EventStartDateISO string
}

type eventViewModel struct {
	ID                 string
	Name               string
	Description        string
	TypeLabel          string
	CoverURL           string
	StartDate          string
	StartDateISO       string
	StartDateFormatted string
	EndDate            string
	DressCode          string
	LocationName       string
	LocationAddress    string
	LocationMapsURL    string
	PrivacyNotice      string
	GuestPolicy        string
	Capacity           int
}

type guestViewModel struct {
	ID       string
	FullName string
	Nickname string
}

type rsvpViewModel struct {
	Status       string
	AttendingPax int
	QRURL        string
}

type menuOption struct {
	Value string
	Label string
}

type sessionViewModel struct {
	Name               string
	Description        string
	StartTime          string
	StartTimeFormatted string
}

type galleryImage struct {
	URL     string
	Caption string
}

// ---------------------------------------------------------------------------
// Routes (registered in routes.go)
// ---------------------------------------------------------------------------

// RegisterSiteRoutes registers public-facing site routes.
// These routes do NOT require authentication.
func (h *InvitationSiteHandler) RegisterSiteRoutes(e *echo.Echo) {
	// Invitation microsite - the main guest-facing page
	e.GET("/i/:token", h.ShowInvitation)

	// Admin dashboard SPA (static HTML, JS handles API calls)
	e.GET("/admin", h.ShowAdminDashboard)
}

// ---------------------------------------------------------------------------
// ShowInvitation renders the invitation microsite for a guest.
// GET /i/{token}
// ---------------------------------------------------------------------------
func (h *InvitationSiteHandler) ShowInvitation(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.HTML(http.StatusBadRequest, errorPage("Invalid invitation link"))
	}

	// Look up the invitation by its opaque token
	invitation, err := h.invitationService.ValidateToken(c.Request().Context(), token)
	if err != nil {
		if stderrors.Is(err, domain.ErrNotFound) {
			return c.HTML(http.StatusNotFound, errorPage("Invitation not found or has been revoked"))
		}
		return c.HTML(http.StatusInternalServerError, errorPage("Unable to load invitation. Please try again later."))
	}

	// Check if invitation has been revoked
	if invitation.Status == domain.InvitationStatusRevoked {
		return c.HTML(http.StatusGone, errorPage("This invitation has been revoked"))
	}

	// Check expiry
	if invitation.ExpiresAt != nil && invitation.ExpiresAt.Before(time.Now()) {
		return c.HTML(http.StatusGone, errorPage("This invitation has expired"))
	}

	// Fetch related data
	ctx := c.Request().Context()

	event, err := h.eventService.Get(ctx, invitation.TenantID, invitation.EventID)
	if err != nil {
		return c.HTML(http.StatusInternalServerError, errorPage("Unable to load event details"))
	}

	// Fetch guest
	guest, err := h.guestService.Get(ctx, invitation.TenantID, invitation.GuestID)
	if err != nil {
		// Guest not found is not fatal - we can still show the event
		guest = nil
	}

	// Check if RSVP already submitted
	rsvp, _ := h.rsvpService.GetByInvitation(ctx, invitation.TenantID, invitation.EventID, invitation.ID)

	// Build view model
	vm := h.buildViewModel(invitation, event, guest, rsvp, token)

	// Render template
	return c.Render(http.StatusOK, "invitation.html", vm)
}

// ---------------------------------------------------------------------------
// ShowAdminDashboard serves the admin dashboard HTML.
// GET /admin
// ---------------------------------------------------------------------------
func (h *InvitationSiteHandler) ShowAdminDashboard(c echo.Context) error {
	return c.File("web/templates/admin.html")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (h *InvitationSiteHandler) buildViewModel(
	inv *domain.Invitation,
	event *domain.Event,
	guest *domain.Guest,
	rsvp *domain.RSVPResponse,
	token string,
) invitationViewModel {
	// Determine language
	lang := "id"
	if guest != nil && guest.Language != "" {
		lang = guest.Language
	}

	// Build event view model
	evm := eventViewModel{
		ID:                 event.ID.String(),
		Name:               event.Name,
		Description:        safeStr(event.Description),
		TypeLabel:          formatEventType(event.Type, lang),
		CoverURL:           safeStr(event.CoverURL),
		StartDate:          event.StartDate.Format("Monday, 2 January 2006"),
		StartDateISO:       event.StartDate.Format(time.RFC3339),
		StartDateFormatted: formatDateLong(event.StartDate, lang),
		DressCode:          safeStr(event.DressCode),
		PrivacyNotice:      safeStr(event.PrivacyNotice),
		GuestPolicy:        safeStr(event.GuestPolicy),
	}

	if event.EndDate != nil {
		evm.EndDate = event.EndDate.Format("Monday, 2 January 2006")
	}
	if event.Capacity != nil {
		evm.Capacity = *event.Capacity
	}

	// Build guest view model
	var gvm *guestViewModel
	if guest != nil {
		gvm = &guestViewModel{
			ID:       guest.ID.String(),
			FullName: guest.FullName,
			Nickname: safeStr(guest.Nickname),
		}
	}

	// Build RSVP view model
	var rvm *rsvpViewModel
	if rsvp != nil {
		rvm = &rsvpViewModel{
			Status:       rsvp.Status,
			AttendingPax: rsvp.AttendingPax,
			QRURL:        "/api/v1/qr/" + token,
		}
	}

	// Max pax options
	maxPax := inv.MaxPax
	if maxPax < 1 {
		maxPax = 2
	}
	paxOptions := make([]int, maxPax)
	for i := range paxOptions {
		paxOptions[i] = i + 1
	}

	// RSVP deadline
	rsvpDeadline := ""
	rsvpDeadlineISO := ""
	if event.RSVPDeadline != nil {
		rsvpDeadline = event.RSVPDeadline.Format("2 January 2006")
		rsvpDeadlineISO = event.RSVPDeadline.Format(time.RFC3339)
	}

	return invitationViewModel{
		Language:          lang,
		Event:             evm,
		Guest:             gvm,
		InvitationToken:   token,
		RSVPSubmitted:     rsvp != nil,
		RSVP:              rvm,
		MaxPaxOptions:     paxOptions,
		MenuOptions:       defaultMenuOptions(lang),
		RSVPDeadline:      rsvpDeadline,
		RSVPDeadlineISO:   rsvpDeadlineISO,
		EventStartDateISO: event.StartDate.Format(time.RFC3339),
	}
}

// ---------------------------------------------------------------------------
// Utility functions
// ---------------------------------------------------------------------------

func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func formatEventType(t string, lang string) string {
	types := map[string]map[string]string{
		"wedding":    {"id": "Pernikahan", "en": "Wedding"},
		"corporate":  {"id": "Acara Korporat", "en": "Corporate Event"},
		"seminar":    {"id": "Seminar", "en": "Seminar"},
		"conference": {"id": "Konferensi", "en": "Conference"},
		"gathering":  {"id": "Gathering", "en": "Gathering"},
		"government": {"id": "Acara Pemerintahan", "en": "Government Event"},
		"community":  {"id": "Event Komunitas", "en": "Community Event"},
		"vip":        {"id": "Acara VIP", "en": "VIP Event"},
		"family":     {"id": "Acara Keluarga", "en": "Family Event"},
	}
	if m, ok := types[t]; ok {
		if label, ok := m[lang]; ok {
			return label
		}
		return m["en"]
	}
	return t
}

func formatDateLong(t time.Time, lang string) string {
	if lang == "id" {
		return t.Format("Monday, 2 January 2006")
	}
	return t.Format("Monday, January 2, 2006")
}

func defaultMenuOptions(lang string) []menuOption {
	if lang == "id" {
		return []menuOption{
			{Value: "regular", Label: "Menu Reguler"},
			{Value: "vegetarian", Label: "Vegetarian"},
			{Value: "vegan", Label: "Vegan"},
			{Value: "halal", Label: "Halal"},
			{Value: "kosher", Label: "Kosher"},
		}
	}
	return []menuOption{
		{Value: "regular", Label: "Regular"},
		{Value: "vegetarian", Label: "Vegetarian"},
		{Value: "vegan", Label: "Vegan"},
		{Value: "halal", Label: "Halal"},
		{Value: "kosher", Label: "Kosher"},
	}
}

// errorPage returns a simple HTML error page.
func errorPage(message string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>GuestFlow - Error</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:system-ui,-apple-system,sans-serif;background:#f8fafc;display:flex;align-items:center;justify-content:center;min-height:100vh;text-align:center;padding:24px}
.container{max-width:400px}
h1{font-size:3rem;margin-bottom:8px}
p{color:#64748b;margin-bottom:24px;line-height:1.6}
a{color:#6366f1;text-decoration:none;font-weight:600}
a:hover{text-decoration:underline}
</style>
</head>
<body>
<div class="container">
<h1>😕</h1>
<h2>Oops!</h2>
<p>` + template.HTMLEscapeString(message) + `</p>
<a href="/">Back to Home</a>
</div>
</body>
</html>`
}
