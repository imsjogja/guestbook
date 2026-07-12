package handler

import (
	"errors"
	"net/http"

	"guestflow/internal/domain"
	mid "guestflow/internal/middleware"
	"guestflow/internal/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// TenantHandler handles HTTP requests for tenant operations.
type TenantHandler struct {
	tenantService *service.TenantService
}

// NewTenantHandler creates a new TenantHandler.
func NewTenantHandler(tenantService *service.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
	}
}

// TenantResponse is the standardized API response for tenant operations.
type TenantResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// Create handles POST /api/v1/tenants - creates a new tenant.
func (h *TenantHandler) Create(c echo.Context) error {
	var req domain.TenantCreateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, TenantResponse{Error: "invalid request body"})
	}

	// Validate required fields.
	if req.Name == "" || req.Slug == "" {
		return c.JSON(http.StatusBadRequest, TenantResponse{Error: "name and slug are required"})
	}

	// Get user ID from context (set by auth middleware).
	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, TenantResponse{Error: "unauthenticated"})
	}

	tenant, err := h.tenantService.Create(c.Request().Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDuplicateSlug):
			return c.JSON(http.StatusConflict, TenantResponse{Error: "tenant slug already exists"})
		case errors.Is(err, domain.ErrInvalidInput):
			return c.JSON(http.StatusBadRequest, TenantResponse{Error: err.Error()})
		default:
			return c.JSON(http.StatusInternalServerError, TenantResponse{Error: "failed to create tenant"})
		}
	}

	return c.JSON(http.StatusCreated, TenantResponse{Data: tenant})
}

// Get handles GET /api/v1/tenants/:id - retrieves a tenant.
func (h *TenantHandler) Get(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, TenantResponse{Error: "invalid tenant id"})
	}

	tenant, err := h.tenantService.Get(c.Request().Context(), tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrTenantNotFound) {
			return c.JSON(http.StatusNotFound, TenantResponse{Error: "tenant not found"})
		}
		return c.JSON(http.StatusInternalServerError, TenantResponse{Error: "failed to retrieve tenant"})
	}

	return c.JSON(http.StatusOK, TenantResponse{Data: tenant})
}

// Update handles PATCH /api/v1/tenants/:id - updates a tenant.
func (h *TenantHandler) Update(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, TenantResponse{Error: "invalid tenant id"})
	}

	var req domain.TenantUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, TenantResponse{Error: "invalid request body"})
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, TenantResponse{Error: "unauthenticated"})
	}

	tenant, err := h.tenantService.Update(c.Request().Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, domain.ErrTenantNotFound) {
			return c.JSON(http.StatusNotFound, TenantResponse{Error: "tenant not found"})
		}
		return c.JSON(http.StatusInternalServerError, TenantResponse{Error: "failed to update tenant"})
	}

	return c.JSON(http.StatusOK, TenantResponse{Data: tenant})
}

// List handles GET /api/v1/tenants - lists the current user's tenants.
func (h *TenantHandler) List(c echo.Context) error {
	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, TenantResponse{Error: "unauthenticated"})
	}

	tenants, err := h.tenantService.ListMyTenants(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, TenantResponse{Error: "failed to list tenants"})
	}

	return c.JSON(http.StatusOK, TenantResponse{Data: tenants})
}

// InviteUser handles POST /api/v1/tenants/:id/users/invite - invites a user.
func (h *TenantHandler) InviteUser(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, TenantResponse{Error: "invalid tenant id"})
	}

	var req domain.TenantInvitationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, TenantResponse{Error: "invalid request body"})
	}

	if req.Email == "" || req.Role == "" {
		return c.JSON(http.StatusBadRequest, TenantResponse{Error: "email and role are required"})
	}

	invitedBy, err := getUserIDFromEchoContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, TenantResponse{Error: "unauthenticated"})
	}

	if err := h.tenantService.InviteUser(c.Request().Context(), tenantID, invitedBy, req.Email, req.Role); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidRole):
			return c.JSON(http.StatusBadRequest, TenantResponse{Error: "invalid role"})
		case errors.Is(err, domain.ErrForbidden):
			return c.JSON(http.StatusForbidden, TenantResponse{Error: "forbidden"})
		case errors.Is(err, domain.ErrAlreadyExists):
			return c.JSON(http.StatusConflict, TenantResponse{Error: "user is already a member"})
		case errors.Is(err, domain.ErrTenantNotFound):
			return c.JSON(http.StatusNotFound, TenantResponse{Error: "tenant not found"})
		default:
			return c.JSON(http.StatusInternalServerError, TenantResponse{Error: "failed to invite user"})
		}
	}

	return c.JSON(http.StatusOK, TenantResponse{Message: "invitation sent"})
}

// RemoveUser handles DELETE /api/v1/tenants/:id/users/:userId - removes a user.
func (h *TenantHandler) RemoveUser(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, TenantResponse{Error: "invalid tenant id"})
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, TenantResponse{Error: "invalid user id"})
	}

	removedBy, err := getUserIDFromEchoContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, TenantResponse{Error: "unauthenticated"})
	}

	if err := h.tenantService.RemoveUser(c.Request().Context(), tenantID, removedBy, targetUserID); err != nil {
		switch {
		case errors.Is(err, domain.ErrMembershipNotFound):
			return c.JSON(http.StatusNotFound, TenantResponse{Error: "membership not found"})
		case errors.Is(err, domain.ErrCannotRemoveOwner):
			return c.JSON(http.StatusForbidden, TenantResponse{Error: "cannot remove tenant owner"})
		default:
			return c.JSON(http.StatusInternalServerError, TenantResponse{Error: "failed to remove user"})
		}
	}

	return c.JSON(http.StatusOK, TenantResponse{Message: "user removed"})
}

// UpdateUserRole handles PATCH /api/v1/tenants/:id/users/:userId/role - updates a member's role.
func (h *TenantHandler) UpdateUserRole(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, TenantResponse{Error: "invalid tenant id"})
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, TenantResponse{Error: "invalid user id"})
	}

	var req domain.TenantRoleUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, TenantResponse{Error: "invalid request body"})
	}

	if req.Role == "" {
		return c.JSON(http.StatusBadRequest, TenantResponse{Error: "role is required"})
	}

	changedBy, err := getUserIDFromEchoContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, TenantResponse{Error: "unauthenticated"})
	}

	if err := h.tenantService.UpdateUserRole(c.Request().Context(), tenantID, changedBy, targetUserID, req.Role); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidRole):
			return c.JSON(http.StatusBadRequest, TenantResponse{Error: "invalid role"})
		case errors.Is(err, domain.ErrForbidden):
			return c.JSON(http.StatusForbidden, TenantResponse{Error: "forbidden"})
		case errors.Is(err, domain.ErrMembershipNotFound):
			return c.JSON(http.StatusNotFound, TenantResponse{Error: "membership not found"})
		case errors.Is(err, domain.ErrOwnerRoleImmutable):
			return c.JSON(http.StatusForbidden, TenantResponse{Error: "owner role cannot be changed"})
		default:
			return c.JSON(http.StatusInternalServerError, TenantResponse{Error: "failed to update role"})
		}
	}

	return c.JSON(http.StatusOK, TenantResponse{Message: "role updated"})
}

// getUserIDFromEchoContext extracts the user ID from the Echo context.
func getUserIDFromEchoContext(c echo.Context) (uuid.UUID, error) {
	userID := mid.GetUserID(c)
	if userID == uuid.Nil {
		return uuid.UUID{}, errors.New("user_id not found in context")
	}
	return userID, nil
}
