package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// AuthHandler handles authentication-related HTTP requests.
// This is a minimal stub for compilation; the full implementation belongs in the Auth module.
type AuthHandler struct{}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// Register handles user registration.
func (h *AuthHandler) Register(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "register"})
}

// Login handles user login.
func (h *AuthHandler) Login(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "login"})
}

// Refresh handles token refresh.
func (h *AuthHandler) Refresh(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "refresh"})
}

// Logout handles user logout.
func (h *AuthHandler) Logout(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "logout"})
}

// Me returns the current user's information.
func (h *AuthHandler) Me(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "me"})
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// healthHandler handles health check requests.
func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, HealthResponse{
		Status:  "healthy",
		Version: "1.0.0",
	})
}
