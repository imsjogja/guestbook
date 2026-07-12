package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"guestflow/internal/auth"
)

func setupTestEcho() *echo.Echo {
	e := echo.New()
	return e
}

func TestJWTAuth(t *testing.T) {
	jwtService := auth.NewJWTService("test-access", "test-refresh", 15*time.Minute, 7*24*time.Hour)
	middleware := JWTAuth(jwtService)

	t.Run("valid token", func(t *testing.T) {
		e := setupTestEcho()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		token, _ := jwtService.GenerateAccessToken(userID, "test@example.com", uuid.Nil, "admin")

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("missing authorization header", func(t *testing.T) {
		e := setupTestEcho()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		if err == nil {
			t.Fatal("expected error for missing header")
		}
		he, ok := err.(*echo.HTTPError)
		if ok && he.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", he.Code)
		}
	})

	t.Run("invalid token format", func(t *testing.T) {
		e := setupTestEcho()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		if err == nil {
			t.Fatal("expected error for invalid token")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		e := setupTestEcho()
		shortService := auth.NewJWTService("test-access", "test-refresh", -1*time.Second, 7*24*time.Hour)
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		token, _ := shortService.GenerateAccessToken(userID, "test@example.com", uuid.Nil, "")

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Need to validate with the same service that has negative TTL
		shortMiddleware := JWTAuth(shortService)
		handler := shortMiddleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		if err == nil {
			t.Fatal("expected error for expired token")
		}
	})

	t.Run("wrong scheme", func(t *testing.T) {
		e := setupTestEcho()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		if err == nil {
			t.Fatal("expected error for wrong auth scheme")
		}
	})
}

func TestOptionalAuth(t *testing.T) {
	jwtService := auth.NewJWTService("test-access", "test-refresh", 15*time.Minute, 7*24*time.Hour)
	middleware := OptionalAuth(jwtService)

	t.Run("no token - continues", func(t *testing.T) {
		e := setupTestEcho()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("valid token - sets context", func(t *testing.T) {
		e := setupTestEcho()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		token, _ := jwtService.GenerateAccessToken(userID, "test@example.com", uuid.Nil, "user")

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("invalid token - continues without auth", func(t *testing.T) {
		e := setupTestEcho()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})
}

func TestGetUserID(t *testing.T) {
	e := setupTestEcho()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	t.Run("no claims", func(t *testing.T) {
		id := GetUserID(c)
		if id != uuid.Nil {
			t.Errorf("expected nil UUID, got %v", id)
		}
	})

	t.Run("with claims", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		claims := &auth.Claims{
			UserID:           userID,
			Email:            "test@example.com",
			RegisteredClaims: jwt.RegisteredClaims{},
		}
		c.Set(string(userContextKey), claims)

		id := GetUserID(c)
		if id != userID {
			t.Errorf("expected %v, got %v", userID, id)
		}
	})
}

func TestGetTenantID(t *testing.T) {
	e := setupTestEcho()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	t.Run("no claims", func(t *testing.T) {
		id := GetTenantID(c)
		if id != uuid.Nil {
			t.Errorf("expected nil UUID, got %v", id)
		}
	})

	t.Run("with tenant", func(t *testing.T) {
		tenantID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
		claims := &auth.Claims{
			UserID:           uuid.New(),
			TenantID:         tenantID,
			RegisteredClaims: jwt.RegisteredClaims{},
		}
		c.Set(string(userContextKey), claims)

		id := GetTenantID(c)
		if id != tenantID {
			t.Errorf("expected %v, got %v", tenantID, id)
		}
	})
}

func TestGetRole(t *testing.T) {
	e := setupTestEcho()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	t.Run("no claims", func(t *testing.T) {
		role := GetRole(c)
		if role != "" {
			t.Errorf("expected empty role, got %s", role)
		}
	})

	t.Run("with role", func(t *testing.T) {
		claims := &auth.Claims{
			UserID:           uuid.New(),
			Role:             "admin",
			RegisteredClaims: jwt.RegisteredClaims{},
		}
		c.Set(string(userContextKey), claims)

		role := GetRole(c)
		if role != "admin" {
			t.Errorf("expected admin, got %s", role)
		}
	})
}

func TestGetEmail(t *testing.T) {
	e := setupTestEcho()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	t.Run("no claims", func(t *testing.T) {
		email := GetEmail(c)
		if email != "" {
			t.Errorf("expected empty email, got %s", email)
		}
	})

	t.Run("with email", func(t *testing.T) {
		claims := &auth.Claims{
			UserID:           uuid.New(),
			Email:            "test@example.com",
			RegisteredClaims: jwt.RegisteredClaims{},
		}
		c.Set(string(userContextKey), claims)

		email := GetEmail(c)
		if email != "test@example.com" {
			t.Errorf("expected test@example.com, got %s", email)
		}
	})
}

func TestGetClaims(t *testing.T) {
	e := setupTestEcho()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	t.Run("no claims", func(t *testing.T) {
		claims := GetClaims(c)
		if claims != nil {
			t.Error("expected nil claims")
		}
	})

	t.Run("with claims", func(t *testing.T) {
		userID := uuid.New()
		claims := &auth.Claims{
			UserID:           userID,
			Email:            "test@example.com",
			Role:             "user",
			RegisteredClaims: jwt.RegisteredClaims{},
		}
		c.Set(string(userContextKey), claims)

		retrieved := GetClaims(c)
		if retrieved == nil {
			t.Fatal("expected non-nil claims")
		}
		if retrieved.UserID != userID {
			t.Errorf("expected userID %v, got %v", userID, retrieved.UserID)
		}
	})
}

func TestExtractBearerToken(t *testing.T) {
	e := setupTestEcho()

	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{"valid bearer", "Bearer test-token", "test-token"},
		{"valid bearer with spaces", "Bearer  test-token  ", "test-token"},
		{"missing header", "", ""},
		{"wrong scheme", "Basic dXNlcjpwYXNz", ""},
		{"bearer lowercase", "bearer test-token", ""},
		{"empty token", "Bearer ", ""},
		{"extra parts", "Bearer token extra", "token extra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			token := extractBearerToken(c)
			if token != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, token)
			}
		})
	}
}
