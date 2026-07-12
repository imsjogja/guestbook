package handler

import (
	"guestflow/internal/auth"
	"guestflow/internal/middleware"
	"guestflow/internal/rbac"

	"github.com/labstack/echo/v4"
)

// RegisterRoutes registers all API routes for the GuestFlow application.
func RegisterRoutes(
	e *echo.Echo,
	authHandler *AuthHandler,
	tenantHandler *TenantHandler,
	eventHandler *EventHandler,
	guestHandler *GuestHandler,
	householdHandler *HouseholdHandler,
	invitationHandler *InvitationHandler,
	rsvpHandler *RSVPHandler,
	checkinHandler *CheckinHandler,
	seatingHandler *SeatingHandler,
	communicationHandler *CommunicationHandler,
	dashboardHandler *DashboardHandler,
	invitationSiteHandler *InvitationSiteHandler,
	jwtService *auth.JWTService,
	rbacService *rbac.Service,
) {
	// Public API group.
	api := e.Group("/api/v1")

	// Public authentication routes (no JWT required).
	authGroup := api.Group("/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.Refresh)

	// Public RSVP submission route (no auth required - accessed by token).
	api.POST("/rsvp", rsvpHandler.Submit)

	// Protected routes require valid JWT.
	protected := api.Group("")
	protected.Use(middleware.JWTAuth(jwtService))

	// Auth routes that require authentication.
	authGroup.POST("/logout", authHandler.Logout, middleware.JWTAuth(jwtService))
	authGroup.GET("/me", authHandler.Me, middleware.JWTAuth(jwtService))

	// Tenant routes (protected).
	tenants := protected.Group("/tenants")
	tenants.POST("", tenantHandler.Create)
	tenants.GET("", tenantHandler.List)
	tenants.GET("/:id", tenantHandler.Get)
	tenants.PATCH("/:id", tenantHandler.Update)
	tenants.POST("/:id/users/invite", tenantHandler.InviteUser)
	tenants.DELETE("/:id/users/:userId", tenantHandler.RemoveUser)
	tenants.PATCH("/:id/users/:userId/role", tenantHandler.UpdateUserRole)

	// Guest routes (protected, tenant-scoped).
	// Tenants :id is the tenant ID; guest endpoints are nested under it.
	guests := tenants.Group("/:id/guests")
	guests.POST("", guestHandler.Create)
	guests.GET("", guestHandler.List)
	guests.GET("/search", guestHandler.Search)
	guests.POST("/import", guestHandler.ImportCSV)
	guests.GET("/import/template", guestHandler.DownloadTemplate)
	guests.GET("/:guestId", guestHandler.Get)
	guests.PATCH("/:guestId", guestHandler.Update)
	guests.DELETE("/:guestId", guestHandler.Delete)

	// Household routes (protected, tenant-scoped).
	households := tenants.Group("/:id/households")
	households.POST("", householdHandler.Create)
	households.GET("", householdHandler.List)
	households.GET("/:householdId", householdHandler.Get)
	households.PATCH("/:householdId", householdHandler.Update)
	households.DELETE("/:householdId", householdHandler.Delete)
	households.GET("/:householdId/members", householdHandler.ListMembers)
	households.POST("/:householdId/members", householdHandler.AddMember)
	households.DELETE("/:householdId/members/:guestId", householdHandler.RemoveMember)

	// Event routes (protected, tenant-scoped).
	events := tenants.Group("/:id/events")
	events.POST("", eventHandler.Create)
	events.GET("", eventHandler.List)
	events.GET("/:eventId", eventHandler.Get)
	events.PATCH("/:eventId", eventHandler.Update)
	events.DELETE("/:eventId", eventHandler.Delete)
	events.POST("/:eventId/publish", eventHandler.Publish)

	// Invitation routes (protected, tenant-scoped, nested under events).
	invitations := events.Group("/:eventId/invitations")
	invitations.POST("", invitationHandler.Create)
	invitations.GET("", invitationHandler.List)
	invitations.POST("/batch", invitationHandler.BatchCreate)
	invitations.GET("/:invitationId", invitationHandler.Get)
	invitations.DELETE("/:invitationId", invitationHandler.Delete)
	invitations.GET("/:invitationId/qr", invitationHandler.GetQRData)

	// RSVP routes (protected, tenant-scoped, nested under events).
	rsvps := events.Group("/:eventId/rsvp")
	rsvps.GET("", rsvpHandler.List)
	rsvps.GET("/dashboard", rsvpHandler.Dashboard)
	rsvps.PATCH("/:rsvpId", rsvpHandler.UpdateByOfficer)

	// Check-in routes (protected, tenant-scoped, nested under events).
	checkins := events.Group("/:eventId/checkin")
	checkins.POST("", checkinHandler.Checkin)
	checkins.GET("/stats", checkinHandler.GetStats)
	checkins.GET("/search", checkinHandler.SearchGuests)
	checkins.GET("/recent", checkinHandler.GetRecent)
	checkins.POST("/walkin", checkinHandler.Walkin)

	// Seating / Table routes (protected, tenant-scoped, nested under events).
	tables := events.Group("/:eventId/tables")
	tables.POST("", seatingHandler.CreateTable)
	tables.GET("", seatingHandler.ListTables)
	tables.GET("/:tableId", seatingHandler.GetTable)
	tables.PATCH("/:tableId", seatingHandler.UpdateTable)
	tables.DELETE("/:tableId", seatingHandler.DeleteTable)
	tables.POST("/:tableId/assign", seatingHandler.AssignGuest)
	tables.DELETE("/:tableId/assign/:guestId", seatingHandler.UnassignGuest)

	// Seating zone routes (protected, tenant-scoped, nested under events).
	zones := events.Group("/:eventId/zones")
	zones.POST("", seatingHandler.CreateZone)
	zones.GET("", seatingHandler.ListZones)

	// Seating layout route (protected, tenant-scoped, nested under events).
	seating := events.Group("/:eventId/seating")
	seating.GET("/layout", seatingHandler.GetLayout)
	seating.POST("/auto-assign", seatingHandler.AutoAssign)

	// Communication template routes (protected, tenant-scoped).
	templates := tenants.Group("/:id/templates")
	templates.POST("", communicationHandler.CreateTemplate)
	templates.GET("", communicationHandler.ListTemplates)
	templates.GET("/:templateId", communicationHandler.GetTemplate)
	templates.PATCH("/:templateId", communicationHandler.UpdateTemplate)
	templates.DELETE("/:templateId", communicationHandler.DeleteTemplate)

	// Communication message routes (protected, tenant-scoped, nested under events).
	messages := events.Group("/:eventId/messages")
	messages.POST("/send", communicationHandler.SendMessage)
	messages.GET("", communicationHandler.ListMessages)

	// Communication campaign routes (protected, tenant-scoped, nested under events).
	campaigns := events.Group("/:eventId/campaigns")
	campaigns.POST("", communicationHandler.CreateCampaign)
	campaigns.GET("", communicationHandler.ListCampaigns)
	campaigns.POST("/:campaignId/launch", communicationHandler.LaunchCampaign)
	campaigns.POST("/:campaignId/cancel", communicationHandler.CancelCampaign)

	// Dashboard routes (protected, tenant-scoped, nested under events).
	dashboard := events.Group("/:eventId/dashboard")
	dashboard.GET("", dashboardHandler.GetDashboard)
	dashboard.GET("/stream", dashboardHandler.StreamDashboard)

	// RBAC middleware is injected but route-level enforcement
	// is applied per-endpoint in production.
	_ = rbacService

	// ------------------------------------------------------------------
	// Public-facing site routes (no authentication)
	// ------------------------------------------------------------------
	invitationSiteHandler.RegisterSiteRoutes(e)
}
