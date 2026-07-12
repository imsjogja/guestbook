package middleware

import (
	"errors"
	"net/http"

	"guestflow/internal/domain"
	"guestflow/internal/rbac"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// RBACMiddleware creates middleware that injects RBAC services into the request context.
// It does not perform any checks itself; downstream middleware or handlers use the service.
func RBACMiddleware(rbacService *rbac.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("rbac_service", rbacService)
			return next(c)
		}
	}
}

// RequirePermission creates middleware that checks if the authenticated user has the
// specified permission within the tenant identified by the "id" path parameter.
func RequirePermission(rbacService *rbac.Service, permission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID, userID, ok := extractIDsForRBAC(c)
			if !ok {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "tenant id or user id missing from request",
				})
			}

			if err := rbacService.EnforcePermission(c.Request().Context(), tenantID, userID, permission); err != nil {
				if errors.Is(err, domain.ErrForbidden) {
					return c.JSON(http.StatusForbidden, map[string]string{
						"error": "insufficient permissions",
					})
				}
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "permission check failed",
				})
			}

			return next(c)
		}
	}
}

// RequireRole creates middleware that checks if the authenticated user has one of
// the specified roles within the tenant identified by the "id" path parameter.
func RequireRole(rbacService *rbac.Service, roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID, userID, ok := extractIDsForRBAC(c)
			if !ok {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "tenant id or user id missing from request",
				})
			}

			role, err := rbacService.GetRole(c.Request().Context(), tenantID, userID)
			if err != nil {
				if errors.Is(err, domain.ErrMembershipNotFound) {
					return c.JSON(http.StatusForbidden, map[string]string{
						"error": "not a member of this tenant",
					})
				}
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "role check failed",
				})
			}

			for _, allowed := range roles {
				if role == allowed {
					return next(c)
				}
			}

			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "required role not satisfied",
			})
		}
	}
}

// extractIDsForRBAC extracts tenantID from path parameter "id" and userID from context.
func extractIDsForRBAC(c echo.Context) (tenantID, userID uuid.UUID, ok bool) {
	tenantIDStr := c.Param("id")
	if tenantIDStr == "" {
		return uuid.UUID{}, uuid.UUID{}, false
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, false
	}

	userID = GetUserID(c)
	if userID == uuid.Nil {
		return uuid.UUID{}, uuid.UUID{}, false
	}

	return tenantID, userID, true
}
