// GuestFlow - HTTP Server Entry Point
//
// This is the main entry point for the GuestFlow application server.
// It initializes all dependencies using proper dependency injection:
//
// Config -> DB/Redis -> Repositories -> Services -> Handlers -> Routes
//
// Architecture: Modular Monolith (Go + Echo + PostgreSQL + Redis)
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"guestflow/internal/audit"
	"guestflow/internal/auth"
	"guestflow/internal/config"
	"guestflow/internal/handler"
	"guestflow/internal/middleware"
	"guestflow/internal/rbac"
	"guestflow/internal/repository"
	"guestflow/internal/service"
	"guestflow/internal/validator"
	appresponse "guestflow/pkg/response"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
)

// Build information set at compile time via -ldflags.
var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	var healthCheck bool
	flag.BoolVar(&healthCheck, "health-check", false, "Run a quick health check and exit")
	flag.Parse()

	if healthCheck {
		os.Exit(0)
	}

	// -------------------------------------------------------------------------
	// 1. Load Configuration
	// -------------------------------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	initLogger(cfg.Log)

	slog.Info("starting GuestFlow server",
		"version", version,
		"build_time", buildTime,
		"environment", cfg.App.Env,
		"port", cfg.Server.Port,
	)

	// -------------------------------------------------------------------------
	// 2. Initialize Infrastructure (DB, Redis)
	// -------------------------------------------------------------------------
	db, err := repository.NewPostgresConnection(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	redisClient, err := repository.NewRedisConnection(cfg.Redis)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	slog.Info("infrastructure connected",
		"database", cfg.Database.Host,
		"redis", cfg.Redis.Addr(),
	)

	// -------------------------------------------------------------------------
	// 3. Build Dependency Graph (Repository -> Service -> Handler)
	// -------------------------------------------------------------------------
	e := createServer(cfg, db, redisClient)

	// -------------------------------------------------------------------------
	// 4. Start Server with Graceful Shutdown
	// -------------------------------------------------------------------------
	startServer(e, cfg)
}

// createServer sets up the Echo instance with full dependency injection.
func createServer(cfg *config.Config, db *sqlx.DB, redisClient *redis.Client) *echo.Echo {
	e := echo.New()
	e.HideBanner = cfg.App.Env == "production"
	e.HidePort = cfg.App.Env == "production"

	// =====================================================================
	// Middleware Stack (order matters)
	// =====================================================================
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins:     cfg.CORS.AllowedOrigins,
		AllowMethods:     cfg.CORS.AllowedMethods,
		AllowHeaders:     append(cfg.CORS.AllowedHeaders, cfg.Tenant.Header),
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	if redisClient != nil {
		rlConfig := middleware.DefaultRateLimitConfig(redisClient)
		rlConfig.RequestsPerSecond = cfg.RateLimit.RequestsPerSecond
		rlConfig.Burst = cfg.RateLimit.Burst
		rlConfig.Window = cfg.RateLimit.TTL
		e.Use(middleware.RateLimit(rlConfig))
	}

	e.Use(echomiddleware.BodyLimit("10M"))
	e.Use(echomiddleware.GzipWithConfig(echomiddleware.GzipConfig{
		Level: 5,
		Skipper: func(c echo.Context) bool {
			// Keep API JSON responses uncompressed. This avoids clients receiving
			// an empty response when a browser closes a gzip stream early.
			return c.Request().URL.Path == "/health" || strings.HasPrefix(c.Request().URL.Path, "/api/")
		},
	}))
	e.Use(echomiddleware.SecureWithConfig(echomiddleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000,
		ContentSecurityPolicy: "default-src 'self'",
	}))

	// Validator
	e.Validator = validator.New()

	// =====================================================================
	// Dependency Injection: Layer 1 - Auth & RBAC Services
	// =====================================================================
	jwtService := auth.NewJWTService(
		cfg.JWT.Secret,
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
	)
	refreshSvc := auth.NewRefreshTokenService(db)
	authService := service.NewAuthService(db, jwtService, refreshSvc)
	rbacService := rbac.NewService(repository.NewTenantUserRepository(db))

	// =====================================================================
	// Dependency Injection: Layer 2 - Repositories
	// =====================================================================
	// Auth & Tenant
	tenantRepo := repository.NewTenantRepository(db)
	tenantUserRepo := repository.NewTenantUserRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	// Event
	eventRepo := repository.NewEventRepository(db)
	eventSessionRepo := repository.NewEventSessionRepository(db)
	eventLocationRepo := repository.NewEventLocationRepository(db)

	// Guest
	guestRepo := repository.NewGuestRepository(db)
	eventGuestRepo := repository.NewEventGuestRepository(db)
	householdRepo := repository.NewHouseholdRepository(db)
	guestTagRepo := repository.NewGuestTagRepository(db)

	// Invitation & RSVP
	invitationRepo := repository.NewInvitationRepository(db)
	rsvpRepo := repository.NewRSVPRepository(db)

	// Check-in & Seating
	checkinRepo := repository.NewCheckinRepository(db)
	seatingRepo := repository.NewSeatingRepository(db)

	// Communication
	commRepo := repository.NewCommunicationRepository(db)

	// =====================================================================
	// Dependency Injection: Layer 3 - Audit Service
	// =====================================================================
	auditService := audit.NewService(auditRepo)

	// =====================================================================
	// Dependency Injection: Layer 4 - Business Services
	// =====================================================================
	tenantService := service.NewTenantService(tenantRepo, tenantUserRepo, repository.NewUserRepository(db), auditService)
	eventService := service.NewEventService(eventRepo, eventSessionRepo, eventLocationRepo, auditService)
	guestService := service.NewGuestService(guestRepo, householdRepo, guestTagRepo, auditService)
	eventGuestService := service.NewEventGuestService(eventGuestRepo, eventRepo, guestRepo, guestService, auditService)
	householdService := service.NewHouseholdService(householdRepo, auditService)
	invitationService := service.NewInvitationService(invitationRepo, eventRepo, rsvpRepo, guestRepo, eventGuestRepo, auditService)
	rsvpService := service.NewRSVPService(rsvpRepo, invitationRepo, eventRepo, eventGuestRepo, auditService)
	checkinService := service.NewCheckinService(checkinRepo, guestRepo, invitationRepo, eventGuestRepo, eventRepo, seatingRepo, auditService)
	seatingService := service.NewSeatingService(seatingRepo, guestRepo, eventGuestRepo, auditService)
	commService := service.NewCommunicationService(commRepo, guestRepo, eventGuestRepo, eventRepo)
	dashboardService := service.NewDashboardService(db, eventRepo, rsvpRepo, checkinRepo, commRepo, seatingRepo)

	// =====================================================================
	// Dependency Injection: Layer 5 - HTTP Handlers
	// =====================================================================
	authHandler := handler.NewAuthHandler(authService)
	tenantHandler := handler.NewTenantHandler(tenantService)
	eventHandler := handler.NewEventHandler(eventService)
	guestHandler := handler.NewGuestHandler(guestService)
	eventGuestHandler := handler.NewEventGuestHandler(eventGuestService)
	householdHandler := handler.NewHouseholdHandler(householdService)
	invitationHandler := handler.NewInvitationHandler(invitationService)
	rsvpHandler := handler.NewRSVPHandler(rsvpService, invitationService)
	checkinHandler := handler.NewCheckinHandler(checkinService)
	seatingHandler := handler.NewSeatingHandler(seatingService)
	communicationHandler := handler.NewCommunicationHandler(commService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	invitationSiteHandler := handler.NewInvitationSiteHandler(invitationService, rsvpService, eventService, guestService)

	// =====================================================================
	// Template Renderer for HTML views (required for HTMX fragments)
	// =====================================================================
	renderer, err := handler.NewTemplateRenderer()
	if err != nil {
		slog.Warn("failed to load HTML templates, using default", "error", err)
	} else {
		e.Renderer = renderer
	}

	// =====================================================================
	// HTMX Dashboard Handler (for real-time HTML fragment updates)
	// =====================================================================
	var htmxDashboardHandler *handler.HTMXDashboardHandler
	if renderer != nil {
		htmxDashboardHandler = handler.NewHTMXDashboardHandler(
			dashboardService,
			checkinService,
			guestService,
			rsvpService,
			renderer,
		)
		slog.Info("HTMX dashboard handler initialized")
	} else {
		slog.Warn("HTMX dashboard handler disabled: no template renderer available")
	}

	// =====================================================================
	// Routes: Register all API routes via handler package
	// =====================================================================
	// Static assets
	e.Static("/static", "web/static")

	// Register all API routes (API + HTMX + public site)
	handler.RegisterRoutes(
		e,
		authHandler,
		tenantHandler,
		eventHandler,
		guestHandler,
		eventGuestHandler,
		householdHandler,
		invitationHandler,
		rsvpHandler,
		checkinHandler,
		seatingHandler,
		communicationHandler,
		dashboardHandler,
		invitationSiteHandler,
		htmxDashboardHandler,
		jwtService,
		rbacService,
		db,
	)

	// Health checks (public)
	e.GET("/health", handleHealth(db, redisClient))
	e.GET("/healthz", handleHealth(db, redisClient))
	e.GET("/ready", handleReadiness(db, redisClient))

	return e
}

// startServer starts the Echo server and handles graceful shutdown.
func startServer(e *echo.Echo, cfg *config.Config) {
	go func() {
		addr := cfg.Server.ListenAddr()
		slog.Info("server listening", "address", addr)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server stopped")
}

// ------------------------------------------------------------------------------
// Logger Setup
// ------------------------------------------------------------------------------

func initLogger(cfg config.LogConfig) {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

// ------------------------------------------------------------------------------
// Health Check Handlers
// ------------------------------------------------------------------------------

func handleHealth(db *sqlx.DB, redisClient *redis.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
		defer cancel()

		dbOK := true
		if err := repository.HealthCheck(ctx, db); err != nil {
			dbOK = false
			slog.WarnContext(ctx, "health check: database unavailable", "error", err)
		}

		redisOK := true
		if err := repository.RedisHealthCheck(ctx, redisClient); err != nil {
			redisOK = false
			slog.WarnContext(ctx, "health check: redis unavailable", "error", err)
		}

		status := "healthy"
		if !dbOK || !redisOK {
			status = "degraded"
		}

		return appresponse.Success(c, map[string]interface{}{
			"status":    status,
			"database":  dbOK,
			"redis":     redisOK,
			"version":   version,
			"timestamp": time.Now().UTC(),
		})
	}
}

func handleReadiness(db *sqlx.DB, redisClient *redis.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
		defer cancel()

		if err := repository.HealthCheck(ctx, db); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
				"ready":   false,
				"reason":  "database unavailable",
				"version": version,
			})
		}

		return appresponse.Success(c, map[string]interface{}{
			"ready":   true,
			"version": version,
		})
	}
}
