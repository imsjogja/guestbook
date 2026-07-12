package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"guestflow/internal/service"
)

// AuthHandler handles authentication-related HTTP requests.
// This minimal implementation keeps the server wiring compiling while the
// service layer handles the actual authentication logic.
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
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
