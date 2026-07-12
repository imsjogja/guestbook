// Package middleware provides Echo middleware for GuestFlow including
// structured logging, rate limiting, tenant resolution, and more.
package middleware

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// RequestIDHeader is the HTTP header used for request correlation IDs.
const RequestIDHeader = "X-Request-ID"

// LoggerConfig holds configuration for the logging middleware.
type LoggerConfig struct {
	// Skipper defines a function to skip middleware.
	Skipper middleware.Skipper
}

// DefaultLoggerConfig is the default logging middleware configuration.
var DefaultLoggerConfig = LoggerConfig{
	Skipper: middleware.DefaultSkipper,
}

// Logger returns a middleware that logs HTTP requests using structured logging (slog).
// It logs method, path, status, duration, request ID, client IP, and user agent.
func Logger() echo.MiddlewareFunc {
	return LoggerWithConfig(DefaultLoggerConfig)
}

// LoggerWithConfig returns a Logger middleware with custom configuration.
func LoggerWithConfig(cfg LoggerConfig) echo.MiddlewareFunc {
	if cfg.Skipper == nil {
		cfg.Skipper = DefaultLoggerConfig.Skipper
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if cfg.Skipper(c) {
				return next(c)
			}

			start := time.Now()
			req := c.Request()
			res := c.Response()

			// Get or generate request ID
			requestID := req.Header.Get(RequestIDHeader)
			if requestID == "" {
				requestID = res.Header().Get(echo.HeaderXRequestID)
			}

			// Execute the handler
			err := next(c)

			// Calculate duration
			duration := time.Since(start)

			// Determine log level based on status code
			status := res.Status
			level := slog.LevelInfo
			if status >= 500 {
				level = slog.LevelError
			} else if status >= 400 {
				level = slog.LevelWarn
			}

			// Build log attributes
			attrs := []slog.Attr{
				slog.String("request_id", requestID),
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.String("query", req.URL.RawQuery),
				slog.Int("status", status),
				slog.Int64("duration_ms", duration.Milliseconds()),
				slog.String("duration", duration.String()),
				slog.String("client_ip", c.RealIP()),
				slog.String("user_agent", req.UserAgent()),
				slog.Int64("bytes_in", req.ContentLength),
				slog.Int64("bytes_out", res.Size),
			}

			// Add error info if present
			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
			}

			// Add tenant info if available
			if tenantID := c.Request().Header.Get("X-Tenant-ID"); tenantID != "" {
				attrs = append(attrs, slog.String("tenant_id", tenantID))
			}

			// Log with appropriate level
			slog.LogAttrs(c.Request().Context(), level, "http_request", attrs...)

			return err
		}
	}
}

// RequestID returns a middleware that generates or propagates request correlation IDs.
// It reads X-Request-ID from the incoming request or generates a new UUID.
func RequestID() echo.MiddlewareFunc {
	return middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		TargetHeader: echo.HeaderXRequestID,
	})
}

// Recover returns a middleware that recovers from panics and logs them.
// It prevents the server from crashing and returns a 500 Internal Server Error.
func Recover() echo.MiddlewareFunc {
	return middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize:         4 << 10, // 4KB
		DisableStackAll:   false,
		DisablePrintStack: false,
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			slog.ErrorContext(c.Request().Context(), "panic recovered",
				slog.String("error", err.Error()),
				slog.String("path", c.Request().URL.Path),
				slog.String("method", c.Request().Method),
				slog.String("stack", string(stack)),
			)
			return nil
		},
	})
}
