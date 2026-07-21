package handler

import (
	"guestflow/internal/auth"
	"guestflow/internal/domain"
	"guestflow/internal/middleware"
	"guestflow/internal/rbac"
	"guestflow/internal/service"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

// RegisterRoutes registers all API routes for the GuestFlow application.
func RegisterRoutes(
	e *echo.Echo,
	authHandler *AuthHandler,
	tenantHandler *TenantHandler,
	eventHandler *EventHandler,
	eventMemberHandler *EventMemberHandler,
	guestHandler *GuestHandler,
	eventGuestHandler *EventGuestHandler,
	householdHandler *HouseholdHandler,
	invitationHandler *InvitationHandler,
	rsvpHandler *RSVPHandler,
	checkinHandler *CheckinHandler,
	seatingHandler *SeatingHandler,
	communicationHandler *CommunicationHandler,
	whatsappIntegrationHandler *WhatsAppIntegrationHandler,
	dashboardHandler *DashboardHandler,
	invitationSiteHandler *InvitationSiteHandler,
	htmxDashboardHandler *HTMXDashboardHandler,
	jwtService *auth.JWTService,
	rbacService *rbac.Service,
	eventAccessService *service.EventAccessService,
	db *sqlx.DB,
) {
	// Public API group.
	api := e.Group("/api/v1")

	// Public authentication routes (no JWT required).
	authGroup := api.Group("/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.Refresh)
	authGroup.GET("/verify-email", authHandler.VerifyEmail)
	authGroup.POST("/resend-verification", authHandler.ResendVerification)
	authGroup.POST("/forgot-password", authHandler.ForgotPassword)
	authGroup.POST("/reset-password", authHandler.ResetPassword)
	authGroup.POST("/magic-link", authHandler.RequestMagicLink)
	authGroup.POST("/magic-link/consume", authHandler.ConsumeMagicLink)

	// Public RSVP submission route (no auth required - accessed by token).
	api.POST("/rsvp", rsvpHandler.Submit)

	// Protected routes require valid JWT.
	protected := api.Group("")
	protected.Use(middleware.JWTAuth(jwtService))

	// Auth routes that require authentication.
	authGroup.POST("/logout", authHandler.Logout, middleware.JWTAuth(jwtService))
	authGroup.GET("/me", authHandler.Me, middleware.JWTAuth(jwtService))
	authGroup.PATCH("/me", authHandler.UpdateMe, middleware.JWTAuth(jwtService))
	authGroup.PATCH("/me/password", authHandler.ChangePassword, middleware.JWTAuth(jwtService))

	// Tenant routes (protected).
	tenants := protected.Group("/tenants")
	tenantEventRead := middleware.RequirePermission(rbacService, domain.PermEventRead)
	tenantEventWrite := middleware.RequirePermission(rbacService, domain.PermEventWrite)
	tenantTeamRead := middleware.RequirePermission(rbacService, domain.PermTeamRead)
	tenantTeamWrite := middleware.RequirePermission(rbacService, domain.PermTeamWrite)
	tenantCommunicationWrite := middleware.RequirePermission(rbacService, domain.PermCommunicationWrite)
	tenantSettingsRead := middleware.RequirePermission(rbacService, domain.PermSettingsRead)
	tenantSettingsWrite := middleware.RequirePermission(rbacService, domain.PermSettingsWrite)
	tenantGuestRead := middleware.RequirePermission(rbacService, domain.PermGuestRead)
	tenantGuestWrite := middleware.RequirePermission(rbacService, domain.PermGuestWrite)
	tenantGuestDelete := middleware.RequirePermission(rbacService, domain.PermGuestDelete)
	tenantCommunicationRead := middleware.RequirePermission(rbacService, domain.PermCommunicationRead)
	tenantMember := middleware.RequireRole(rbacService, rbac.AllRoles()...)
	tenantOwnerOnly := middleware.RequireRole(rbacService, domain.RoleTenantOwner)
	tenants.POST("", tenantHandler.Create)
	tenants.GET("", tenantHandler.List)
	tenants.GET("/:id", tenantHandler.Get, tenantMember)
	tenants.GET("/:id/access", tenantHandler.Access, tenantTeamRead)
	tenants.PATCH("/:id", tenantHandler.Update, tenantOwnerOnly)
	tenants.GET("/:id/users", tenantHandler.ListUsers, tenantTeamRead)
	tenants.POST("/:id/users", tenantHandler.AddUser, tenantTeamWrite)
	tenants.DELETE("/:id/users/:userId", tenantHandler.RemoveUser, tenantTeamWrite)
	tenants.PATCH("/:id/users/:userId/role", tenantHandler.UpdateUserRole, tenantTeamWrite)

	// Guest routes (protected, tenant-scoped).
	// Tenants :id is the tenant ID; guest endpoints are nested under it.
	guests := tenants.Group("/:id/guests")
	guests.POST("", guestHandler.Create, tenantGuestWrite)
	guests.GET("", guestHandler.List, tenantGuestRead)
	guests.GET("/search", guestHandler.Search, tenantGuestRead)
	guests.POST("/import", guestHandler.ImportCSV, tenantGuestWrite)
	guests.GET("/import/template", guestHandler.DownloadTemplate, tenantGuestRead)
	guests.GET("/:guestId", guestHandler.Get, tenantGuestRead)
	guests.PATCH("/:guestId", guestHandler.Update, tenantGuestWrite)
	guests.DELETE("/:guestId", guestHandler.Delete, tenantGuestDelete)

	// Household routes (protected, tenant-scoped).
	households := tenants.Group("/:id/households")
	households.POST("", householdHandler.Create, tenantGuestWrite)
	households.GET("", householdHandler.List, tenantGuestRead)
	households.GET("/:householdId", householdHandler.Get, tenantGuestRead)
	households.PATCH("/:householdId", householdHandler.Update, tenantGuestWrite)
	households.DELETE("/:householdId", householdHandler.Delete, tenantGuestDelete)
	households.GET("/:householdId/members", householdHandler.ListMembers, tenantGuestRead)
	households.POST("/:householdId/members", householdHandler.AddMember, tenantGuestWrite)
	households.DELETE("/:householdId/members/:guestId", householdHandler.RemoveMember, tenantGuestDelete)

	// Event routes (protected, tenant-scoped).
	events := tenants.Group("/:id/events")
	events.POST("", eventHandler.Create, tenantEventWrite)
	events.GET("", eventHandler.List, tenantEventRead)
	eventRead := middleware.RequireEventPermission(eventAccessService, domain.PermEventRead)
	eventWrite := middleware.RequireEventPermission(eventAccessService, domain.PermEventWrite)
	eventDelete := middleware.RequireEventPermission(eventAccessService, domain.PermEventDelete)
	guestRead := middleware.RequireEventPermission(eventAccessService, domain.PermGuestRead)
	guestWrite := middleware.RequireEventPermission(eventAccessService, domain.PermGuestWrite)
	guestDelete := middleware.RequireEventPermission(eventAccessService, domain.PermGuestDelete)
	invitationRead := middleware.RequireEventPermission(eventAccessService, domain.PermInvitationRead)
	invitationWrite := middleware.RequireEventPermission(eventAccessService, domain.PermInvitationWrite)
	rsvpRead := middleware.RequireEventPermission(eventAccessService, domain.PermRSVPRead)
	rsvpWrite := middleware.RequireEventPermission(eventAccessService, domain.PermRSVPWrite)
	checkinRead := middleware.RequireEventPermission(eventAccessService, domain.PermCheckinRead)
	checkinWrite := middleware.RequireEventPermission(eventAccessService, domain.PermCheckinWrite)
	seatingRead := middleware.RequireEventPermission(eventAccessService, domain.PermSeatingRead)
	seatingWrite := middleware.RequireEventPermission(eventAccessService, domain.PermSeatingWrite)
	communicationRead := middleware.RequireEventPermission(eventAccessService, domain.PermCommunicationRead)
	communicationWrite := middleware.RequireEventPermission(eventAccessService, domain.PermCommunicationWrite)
	communicationSend := middleware.RequireEventPermission(eventAccessService, domain.PermCommunicationSend)
	reportRead := middleware.RequireEventPermission(eventAccessService, domain.PermReportRead)
	eventTeamRead := middleware.RequireEventPermission(eventAccessService, domain.PermEventTeamRead)
	eventTeamWrite := middleware.RequireEventPermission(eventAccessService, domain.PermEventTeamWrite)
	events.GET("/:eventId", eventHandler.Get, eventRead)
	events.PATCH("/:eventId", eventHandler.Update, eventWrite)
	events.DELETE("/:eventId", eventHandler.Delete, eventDelete)
	events.POST("/:eventId/publish", eventHandler.Publish, eventWrite)

	// Event staff assignment routes.
	eventMembers := events.Group("/:eventId/members")
	eventMembers.GET("", eventMemberHandler.List, eventTeamRead)
	eventMembers.GET("/access", eventMemberHandler.Access, eventRead)
	eventMembers.POST("", eventMemberHandler.Create, eventTeamWrite)
	eventMembers.PATCH("/:userId", eventMemberHandler.UpdateRole, eventTeamWrite)
	eventMembers.DELETE("/:userId", eventMemberHandler.Delete, eventTeamWrite)

	// Event guest roster routes. These are intentionally separate from the
	// tenant guest master so every operational flow can use event scope.
	eventGuests := events.Group("/:eventId/guests")
	eventGuests.POST("", eventGuestHandler.Create, guestWrite)
	eventGuests.GET("", eventGuestHandler.List, guestRead)
	eventGuests.POST("/import", eventGuestHandler.ImportCSV, guestWrite)
	eventGuests.DELETE("/:eventGuestId", eventGuestHandler.Cancel, guestDelete)

	// Invitation routes (protected, tenant-scoped, nested under events).
	invitations := events.Group("/:eventId/invitations")
	invitations.POST("", invitationHandler.Create, invitationWrite)
	invitations.GET("", invitationHandler.List, invitationRead)
	invitations.POST("/batch", invitationHandler.BatchCreate, invitationWrite)
	invitations.GET("/:invitationId", invitationHandler.Get, invitationRead)
	invitations.DELETE("/:invitationId", invitationHandler.Delete, invitationWrite)
	invitations.GET("/:invitationId/qr", invitationHandler.GetQRData, invitationRead)

	// RSVP routes (protected, tenant-scoped, nested under events).
	rsvps := events.Group("/:eventId/rsvp")
	rsvps.GET("", rsvpHandler.List, rsvpRead)
	rsvps.GET("/dashboard", rsvpHandler.Dashboard, rsvpRead)
	rsvps.PATCH("/:rsvpId", rsvpHandler.UpdateByOfficer, rsvpWrite)
	rsvps.POST("/by-guest/:guestId", rsvpHandler.UpsertByGuest, rsvpWrite)
	rsvps.GET("/reminders/candidates", communicationHandler.ListRSVPReminderCandidates, communicationRead)
	rsvps.POST("/reminders", communicationHandler.SendRSVPReminders, communicationSend)

	// Check-in routes (protected, tenant-scoped, nested under events).
	checkins := events.Group("/:eventId/checkin")
	checkins.POST("", checkinHandler.Checkin, checkinWrite)
	checkins.GET("/stats", checkinHandler.GetStats, checkinRead)
	checkins.GET("/search", checkinHandler.SearchGuests, checkinRead)
	checkins.GET("/recent", checkinHandler.GetRecent, checkinRead)
	checkins.POST("/walkin", checkinHandler.Walkin, checkinWrite)

	// Seating / Table routes (protected, tenant-scoped, nested under events).
	tables := events.Group("/:eventId/tables")
	tables.POST("", seatingHandler.CreateTable, seatingWrite)
	tables.GET("", seatingHandler.ListTables, seatingRead)
	tables.GET("/:tableId", seatingHandler.GetTable, seatingRead)
	tables.PATCH("/:tableId", seatingHandler.UpdateTable, seatingWrite)
	tables.DELETE("/:tableId", seatingHandler.DeleteTable, seatingWrite)
	tables.POST("/:tableId/assign", seatingHandler.AssignGuest, seatingWrite)
	tables.DELETE("/:tableId/assign/:guestId", seatingHandler.UnassignGuest, seatingWrite)

	// Seating zone routes (protected, tenant-scoped, nested under events).
	zones := events.Group("/:eventId/zones")
	zones.POST("", seatingHandler.CreateZone, seatingWrite)
	zones.GET("", seatingHandler.ListZones, seatingRead)

	// Seating layout route (protected, tenant-scoped, nested under events).
	seating := events.Group("/:eventId/seating")
	seating.GET("/layout", seatingHandler.GetLayout, seatingRead)
	seating.POST("/auto-assign", seatingHandler.AutoAssign, seatingWrite)

	// Communication template routes (protected, tenant-scoped).
	templates := tenants.Group("/:id/templates")
	templates.POST("", communicationHandler.CreateTemplate, tenantCommunicationWrite)
	templates.POST("/defaults", communicationHandler.GenerateDefaultTemplates, tenantCommunicationWrite)
	templates.GET("", communicationHandler.ListTemplates, tenantCommunicationRead)
	templates.GET("/:templateId", communicationHandler.GetTemplate, tenantCommunicationRead)
	templates.PATCH("/:templateId", communicationHandler.UpdateTemplate, tenantCommunicationWrite)
	templates.DELETE("/:templateId", communicationHandler.DeleteTemplate, tenantCommunicationWrite)

	// Tenant-scoped integration settings. Credentials are encrypted at rest and
	// are never returned; PATCH applies them to the runtime client immediately.
	integrations := tenants.Group("/:id/integrations")
	integrations.GET("/whatsapp", whatsappIntegrationHandler.Get, tenantSettingsRead)
	integrations.PATCH("/whatsapp", whatsappIntegrationHandler.Update, tenantSettingsWrite)

	// Communication message routes (protected, tenant-scoped, nested under events).
	messages := events.Group("/:eventId/messages")
	messages.POST("/send", communicationHandler.SendMessage, communicationSend)
	messages.GET("", communicationHandler.ListMessages, communicationRead)

	// Communication campaign routes (protected, tenant-scoped, nested under events).
	campaigns := events.Group("/:eventId/campaigns")
	campaigns.POST("", communicationHandler.CreateCampaign, communicationWrite)
	campaigns.GET("", communicationHandler.ListCampaigns, communicationRead)
	campaigns.POST("/:campaignId/launch", communicationHandler.LaunchCampaign, communicationSend)
	campaigns.POST("/:campaignId/cancel", communicationHandler.CancelCampaign, communicationWrite)

	// Dashboard routes (protected, tenant-scoped, nested under events).
	dashboard := events.Group("/:eventId/dashboard")
	dashboard.GET("", dashboardHandler.GetDashboard, reportRead)
	dashboard.GET("/stream", dashboardHandler.StreamDashboard, reportRead)

	// ------------------------------------------------------------------
	// HTMX fragment routes (protected, returns HTML partials)
	// ------------------------------------------------------------------
	if htmxDashboardHandler != nil {
		htmxDashboardHandler.RegisterHTMXRoutes(e, middleware.JWTAuth(jwtService), middleware.TenantResolver(middleware.DefaultTenantResolverConfig(db)))
	}

	// ------------------------------------------------------------------
	// Public-facing site routes (no authentication)
	// ------------------------------------------------------------------
	invitationSiteHandler.RegisterSiteRoutes(e)
}
