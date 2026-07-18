package handler

import (
	"errors"
	"net/http"
	"time"

	"guestflow/internal/auth"
	"guestflow/internal/domain"
	"guestflow/internal/middleware"
	"guestflow/internal/service"
	apperrors "guestflow/pkg/errors"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type authUserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"fullName"`
	Role      string `json:"role"`
	Avatar    string `json:"avatar,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type authResponse struct {
	AccessToken  string           `json:"access_token,omitempty"`
	TokenType    string           `json:"token_type"`
	ExpiresIn    int              `json:"expires_in"`
	RefreshToken string           `json:"refresh_token,omitempty"`
	User         authUserResponse `json:"user"`
}

type registrationResponse struct {
	Message                   string           `json:"message"`
	EmailVerificationRequired bool             `json:"email_verification_required"`
	User                      authUserResponse `json:"user"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type emailRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type tokenRequest struct {
	Token string `json:"token" validate:"required"`
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req domain.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}
	if err := c.Validate(&req); err != nil {
		return h.validationError(c, err)
	}

	user, tokens, err := h.authService.Register(c.Request().Context(), req)
	if err != nil {
		return h.handleAuthError(c, err)
	}

	if tokens == nil {
		return c.JSON(http.StatusCreated, registrationResponse{
			Message:                   "registrasi berhasil. Silakan cek email untuk verifikasi akun.",
			EmailVerificationRequired: true,
			User:                      mapUserResponse(user),
		})
	}
	return c.JSON(http.StatusCreated, buildAuthResponse(user, tokens))
}

func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	if err := h.authService.VerifyEmail(c.Request().Context(), c.QueryParam("token")); err != nil {
		if errors.Is(err, service.ErrTokenInvalid) {
			return c.JSON(http.StatusBadRequest, map[string]string{"message": "tautan verifikasi tidak valid atau sudah kedaluwarsa"})
		}
		return h.handleAuthError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "email berhasil diverifikasi"})
}

func (h *AuthHandler) ResendVerification(c echo.Context) error {
	var req emailRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}
	if err := c.Validate(&req); err != nil {
		return h.validationError(c, err)
	}
	if err := h.authService.ResendVerification(c.Request().Context(), req.Email); err != nil {
		return h.handleAuthError(c, err)
	}
	return c.JSON(http.StatusAccepted, map[string]string{"message": "jika akun tersedia dan belum terverifikasi, email verifikasi telah dikirim ulang"})
}

func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var req emailRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}
	if err := c.Validate(&req); err != nil {
		return h.validationError(c, err)
	}
	if err := h.authService.ForgotPassword(c.Request().Context(), req.Email); err != nil {
		return h.handleAuthError(c, err)
	}
	return c.JSON(http.StatusAccepted, map[string]string{"message": "jika akun tersedia, link reset kata sandi telah dikirim"})
}

func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var req resetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}
	if err := c.Validate(&req); err != nil {
		return h.validationError(c, err)
	}
	if err := h.authService.ResetPassword(c.Request().Context(), req.Token, req.Password); err != nil {
		if errors.Is(err, service.ErrTokenInvalid) {
			return c.JSON(http.StatusBadRequest, map[string]string{"message": "link reset kata sandi tidak valid atau sudah kedaluwarsa"})
		}
		return h.handleAuthError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "kata sandi berhasil diubah"})
}

func (h *AuthHandler) RequestMagicLink(c echo.Context) error {
	var req emailRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}
	if err := c.Validate(&req); err != nil {
		return h.validationError(c, err)
	}
	if err := h.authService.RequestMagicLink(c.Request().Context(), req.Email); err != nil {
		return h.handleAuthError(c, err)
	}
	return c.JSON(http.StatusAccepted, map[string]string{"message": "jika akun tersedia dan sudah terverifikasi, link masuk telah dikirim"})
}

func (h *AuthHandler) ConsumeMagicLink(c echo.Context) error {
	var req tokenRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}
	if err := c.Validate(&req); err != nil {
		return h.validationError(c, err)
	}
	user, tokens, err := h.authService.ConsumeMagicLink(c.Request().Context(), req.Token)
	if err != nil {
		if errors.Is(err, service.ErrTokenInvalid) {
			return c.JSON(http.StatusBadRequest, map[string]string{"message": "link masuk tidak valid atau sudah kedaluwarsa"})
		}
		return h.handleAuthError(c, err)
	}
	return c.JSON(http.StatusOK, buildAuthResponse(user, tokens))
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req domain.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}
	if err := c.Validate(&req); err != nil {
		return h.validationError(c, err)
	}

	user, tokens, err := h.authService.Login(c.Request().Context(), req)
	if err != nil {
		return h.handleAuthError(c, err)
	}

	return c.JSON(http.StatusOK, buildAuthResponse(user, tokens))
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	var req refreshRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}
	if err := c.Validate(&req); err != nil {
		return h.validationError(c, err)
	}

	tokens, err := h.authService.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return h.handleAuthError(c, err)
	}

	return c.JSON(http.StatusOK, authResponse{
		AccessToken:  tokens.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokens.ExpiresIn,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *AuthHandler) Logout(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "logout"})
}

func (h *AuthHandler) Me(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "unauthenticated"})
	}

	user, err := h.authService.Me(c.Request().Context(), userID)
	if err != nil {
		return h.handleAuthError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]authUserResponse{
		"user": mapUserResponse(user),
	})
}

func (h *AuthHandler) handleAuthError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, service.ErrEmailExists):
		return c.JSON(http.StatusConflict, map[string]string{"message": "email already registered"})
	case errors.Is(err, service.ErrInvalidCredentials):
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "email atau kata sandi salah"})
	case errors.Is(err, service.ErrUserInactive):
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "akun tidak aktif"})
	case errors.Is(err, service.ErrEmailNotVerified):
		return c.JSON(http.StatusForbidden, map[string]string{"code": "EMAIL_NOT_VERIFIED", "message": "silakan verifikasi email sebelum masuk"})
	case errors.Is(err, service.ErrEmailDelivery):
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"message": "email belum berhasil dikirim. Silakan coba lagi."})
	case errors.Is(err, service.ErrUserNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"message": "user not found"})
	case errors.Is(err, service.ErrTokenInvalid):
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "refresh token tidak valid"})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "failed to process authentication"})
	}
}

func (h *AuthHandler) validationError(c echo.Context, err error) error {
	if appErr, ok := apperrors.IsAppError(err); ok && len(appErr.Details) > 0 {
		return c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
			"message": appErr.Message,
			"errors":  appErr.Details,
		})
	}

	return c.JSON(http.StatusUnprocessableEntity, map[string]string{"message": "validation failed"})
}

func buildAuthResponse(user *domain.User, tokens *auth.TokenPair) authResponse {
	return authResponse{
		AccessToken:  tokens.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokens.ExpiresIn,
		RefreshToken: tokens.RefreshToken,
		User:         mapUserResponse(user),
	}
}

func mapUserResponse(user *domain.User) authUserResponse {
	if user == nil {
		return authUserResponse{}
	}

	avatar := ""
	if user.AvatarURL != nil {
		avatar = *user.AvatarURL
	}

	return authUserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		FullName:  user.FullName,
		Role:      "viewer",
		Avatar:    avatar,
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.UTC().Format(time.RFC3339),
	}
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
