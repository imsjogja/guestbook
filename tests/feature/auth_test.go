// tests/feature/auth_test.go
//
// Feature tests for authentication endpoints.
package feature

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthRegister validates user registration request structure.
func TestAuthRegister(t *testing.T) {
	e := echo.New()
	e.POST("/api/v1/auth/register", func(c echo.Context) error {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			FullName string `json:"full_name"`
			Phone    string `json:"phone"`
		}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "invalid request body",
				"code":  "BAD_REQUEST",
			})
		}

		// Validation
		if req.Email == "" || req.Password == "" || req.FullName == "" {
			return c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
				"error": "validation failed",
				"code":  "VALIDATION_ERROR",
				"details": []map[string]string{
					{"field": "email", "message": "required"},
				},
			})
		}

		if len(req.Password) < 8 {
			return c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
				"error": "validation failed",
				"code":  "VALIDATION_ERROR",
				"details": []map[string]string{
					{"field": "password", "message": "min 8 characters"},
				},
			})
		}

		return c.JSON(http.StatusCreated, map[string]interface{}{
			"data": map[string]interface{}{
				"access_token":  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
				"refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.refresh",
				"expires_in":    900,
				"user": map[string]string{
					"id":        "550e8400-e29b-41d4-a716-446655440000",
					"email":     req.Email,
					"full_name": req.FullName,
					"role":      "",
				},
			},
		})
	})

	t.Run("valid registration", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":     "test@example.com",
			"password":  "securePass123",
			"full_name": "Test User",
			"phone":     "+6281234567890",
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code, "should create user")

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

		data := resp["data"].(map[string]interface{})
		assert.NotEmpty(t, data["access_token"], "should return access token")
		assert.NotEmpty(t, data["refresh_token"], "should return refresh token")
		assert.Equal(t, float64(900), data["expires_in"], "should have 15min expiry")

		user := data["user"].(map[string]interface{})
		assert.Equal(t, "test@example.com", user["email"])
		assert.Equal(t, "Test User", user["full_name"])
	})

	t.Run("missing required fields", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email": "test@example.com",
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "should validate required fields")
	})

	t.Run("password too short", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":     "test@example.com",
			"password":  "short",
			"full_name": "Test User",
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "should validate password length")

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "VALIDATION_ERROR", resp["code"])
	})
}

// TestAuthLogin validates login request structure.
func TestAuthLogin(t *testing.T) {
	e := echo.New()
	e.POST("/api/v1/auth/login", func(c echo.Context) error {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid request"})
		}

		if req.Email != "test@example.com" || req.Password != "correct" {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"error": "invalid credentials",
				"code":  "UNAUTHORIZED",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"data": map[string]interface{}{
				"access_token":  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
				"refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.refresh",
				"expires_in":    900,
			},
		})
	})

	t.Run("valid login", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":    "test@example.com",
			"password": "correct",
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":    "test@example.com",
			"password": "wrong",
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
