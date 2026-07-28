package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"guestflow/internal/auth"
)

func TestExtractIDsForRBACUsesResolvedTenantContext(t *testing.T) {
	e := echo.New()
	tenantID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/subscription", nil)
	req = req.WithContext(context.WithValue(req.Context(), tenantContextKey, tenantID))
	c := e.NewContext(req, httptest.NewRecorder())
	c.Set(string(userContextKey), &auth.Claims{UserID: userID, RegisteredClaims: jwt.RegisteredClaims{}})

	resolvedTenant, resolvedUser, ok := extractIDsForRBAC(c)
	if !ok || resolvedTenant != tenantID || resolvedUser != userID {
		t.Fatalf("expected billing scope %s/%s, got %s/%s (ok=%v)", tenantID, userID, resolvedTenant, resolvedUser, ok)
	}
}

type eventAccessRecorder struct {
	tenantID   uuid.UUID
	eventID    uuid.UUID
	userID     uuid.UUID
	permission string
}

func (r *eventAccessRecorder) Authorize(_ context.Context, tenantID, eventID, userID uuid.UUID, permission string) error {
	r.tenantID = tenantID
	r.eventID = eventID
	r.userID = userID
	r.permission = permission
	return nil
}

func TestRequireEventPermissionSupportsTenantIDPath(t *testing.T) {
	e := echo.New()
	tenantID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	eventID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	userID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	recorder := &eventAccessRecorder{}

	req := httptest.NewRequest(http.MethodGet, "/htmx/dashboard/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/cccccccc-cccc-cccc-cccc-cccccccccccc/stats", nil)
	c := e.NewContext(req, httptest.NewRecorder())
	c.SetParamNames("tenantId", "eventId")
	c.SetParamValues(tenantID.String(), eventID.String())
	c.Set(string(userContextKey), &auth.Claims{UserID: userID, RegisteredClaims: jwt.RegisteredClaims{}})

	handler := RequireEventPermission(recorder, "report:read")(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c)
	if err := handler; err != nil {
		t.Fatalf("unexpected authorization error: %v", err)
	}
	if recorder.tenantID != tenantID || recorder.eventID != eventID || recorder.userID != userID || recorder.permission != "report:read" {
		t.Fatalf("unexpected event scope: %+v", recorder)
	}
}
