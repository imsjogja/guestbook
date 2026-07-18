package middleware

import (
	"context"
	"errors"
	"net/http"

	"guestflow/internal/domain"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type EventAccessChecker interface {
	Authorize(ctx context.Context, tenantID, eventID, userID uuid.UUID, permission string) error
}

// RequireEventPermission enforces the effective role for a specific event.
func RequireEventPermission(accessService EventAccessChecker, permission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID, err := uuid.Parse(c.Param("id"))
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant id"})
			}
			eventID, err := uuid.Parse(c.Param("eventId"))
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid event id"})
			}
			userID := GetUserID(c)
			if userID == uuid.Nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
			}
			if err := accessService.Authorize(c.Request().Context(), tenantID, eventID, userID, permission); err != nil {
				if errors.Is(err, domain.ErrEventNotFound) {
					return c.JSON(http.StatusNotFound, map[string]string{"error": "event not found"})
				}
				return c.JSON(http.StatusForbidden, map[string]string{"error": "insufficient event permissions"})
			}
			return next(c)
		}
	}
}
