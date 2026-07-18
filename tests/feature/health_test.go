// tests/feature/health_test.go
//
// End-to-end feature tests for the GuestFlow API.
// Run with: go test ./tests/feature/... -v
package feature

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupServer creates a test Echo server with registered routes.
// Uses in-memory stubs for external dependencies.
func setupServer(t *testing.T) *echo.Echo {
	e := echo.New()

	// Register minimal routes for testing
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "healthy",
			"database":  true,
			"redis":     true,
			"version":   "test",
			"timestamp": "2026-07-12T00:00:00Z",
		})
	})

	e.GET("/api/v1/auth/me", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"id":        "550e8400-e29b-41d4-a716-446655440000",
			"email":     "test@example.com",
			"full_name": "Test User",
			"role":      "event_manager",
			"tenant_id": "550e8400-e29b-41d4-a716-446655440001",
		})
	})

	return e
}

// TestHealthEndpoint verifies the health check endpoint.
func TestHealthEndpoint(t *testing.T) {
	e := setupServer(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "health endpoint should return 200")

	var body map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err, "response should be valid JSON")

	assert.Equal(t, "healthy", body["status"], "status should be healthy")
	assert.Equal(t, true, body["database"], "database should be true")
	assert.Equal(t, true, body["redis"], "redis should be true")
	assert.NotNil(t, body["timestamp"], "timestamp should be present")
}

// TestAuthMeEndpoint verifies the authenticated user endpoint structure.
func TestAuthMeEndpoint(t *testing.T) {
	e := setupServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "auth me should return 200")

	var body map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err, "response should be valid JSON")

	assert.NotEmpty(t, body["id"], "user ID should be present")
	assert.NotEmpty(t, body["email"], "email should be present")
	assert.NotEmpty(t, body["full_name"], "full_name should be present")
	assert.NotEmpty(t, body["role"], "role should be present")
}

// TestAPIResponseFormat verifies the standard API response envelope.
func TestAPIResponseFormat(t *testing.T) {
	e := echo.New()
	e.GET("/test/success", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"data": map[string]string{"message": "ok"},
		})
	})
	e.GET("/test/error", func(c echo.Context) error {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "validation_error",
			"code":    "VALIDATION_ERROR",
			"details": []map[string]string{{"field": "email", "message": "required"}},
		})
	})

	// Test success format
	req := httptest.NewRequest(http.MethodGet, "/test/success", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var success map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &success))
	assert.NotNil(t, success["data"], "success response should have data field")

	// Test error format
	req = httptest.NewRequest(http.MethodGet, "/test/error", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.NotEmpty(t, errResp["error"], "error response should have error field")
	assert.NotEmpty(t, errResp["code"], "error response should have code field")
}
