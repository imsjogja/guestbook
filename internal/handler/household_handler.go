package handler

import (
	"errors"
	"strconv"
	"strings"

	"guestflow/internal/domain"
	"guestflow/internal/service"
	appresponse "guestflow/pkg/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// HouseholdHandler handles HTTP requests for household operations.
type HouseholdHandler struct {
	householdService *service.HouseholdService
}

// NewHouseholdHandler creates a new HouseholdHandler.
func NewHouseholdHandler(householdService *service.HouseholdService) *HouseholdHandler {
	return &HouseholdHandler{
		householdService: householdService,
	}
}

// Create handles POST /api/v1/tenants/:id/households - creates a new household.
func (h *HouseholdHandler) Create(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	var req domain.HouseholdCreateRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	if req.Name == "" {
		return appresponse.BadRequest(c, "Name is required")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	household, err := h.householdService.Create(c.Request().Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return appresponse.ValidationError(c, err.Error())
		}
		return appresponse.InternalError(c, "Failed to create household")
	}

	return appresponse.Created(c, household)
}

// List handles GET /api/v1/tenants/:id/households - lists households.
func (h *HouseholdHandler) List(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	params := domain.HouseholdListParams{
		TenantID: tenantID,
		Search:   strings.TrimSpace(c.QueryParam("search")),
		Page:     page,
		PerPage:  perPage,
	}

	households, total, err := h.householdService.List(c.Request().Context(), params)
	if err != nil {
		return appresponse.InternalError(c, "Failed to list households")
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	return appresponse.Paginated(c, households, appresponse.Meta{
		CurrentPage: page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
	})
}

// Get handles GET /api/v1/tenants/:id/households/:householdId - retrieves a household.
func (h *HouseholdHandler) Get(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	householdID, err := uuid.Parse(c.Param("householdId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid household ID")
	}

	household, err := h.householdService.Get(c.Request().Context(), tenantID, householdID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return appresponse.NotFound(c, "Household")
		}
		return appresponse.InternalError(c, "Failed to retrieve household")
	}

	return appresponse.Success(c, household)
}

// Update handles PATCH /api/v1/tenants/:id/households/:householdId - updates a household.
func (h *HouseholdHandler) Update(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	householdID, err := uuid.Parse(c.Param("householdId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid household ID")
	}

	var req domain.HouseholdUpdateRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	household, err := h.householdService.Update(c.Request().Context(), tenantID, householdID, req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return appresponse.NotFound(c, "Household")
		}
		return appresponse.InternalError(c, "Failed to update household")
	}

	return appresponse.Success(c, household)
}

// Delete handles DELETE /api/v1/tenants/:id/households/:householdId - soft-deletes a household.
func (h *HouseholdHandler) Delete(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	householdID, err := uuid.Parse(c.Param("householdId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid household ID")
	}

	if err := h.householdService.Delete(c.Request().Context(), tenantID, householdID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return appresponse.NotFound(c, "Household")
		}
		return appresponse.InternalError(c, "Failed to delete household")
	}

	return appresponse.NoContent(c)
}

// AddMember handles POST /api/v1/tenants/:id/households/:householdId/members - adds a member.
func (h *HouseholdHandler) AddMember(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	householdID, err := uuid.Parse(c.Param("householdId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid household ID")
	}

	var req struct {
		GuestID   uuid.UUID `json:"guest_id"`
		IsPrimary bool      `json:"is_primary"`
		Role      string    `json:"role,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	if req.GuestID == uuid.Nil {
		return appresponse.BadRequest(c, "guest_id is required")
	}

	if err := h.householdService.AddMember(c.Request().Context(), tenantID, householdID, req.GuestID, req.IsPrimary, req.Role); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return appresponse.NotFound(c, "Household or guest")
		}
		return appresponse.InternalError(c, "Failed to add member")
	}

	return appresponse.Success(c, map[string]string{"message": "Member added"})
}

// RemoveMember handles DELETE /api/v1/tenants/:id/households/:householdId/members/:guestId - removes a member.
func (h *HouseholdHandler) RemoveMember(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	householdID, err := uuid.Parse(c.Param("householdId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid household ID")
	}

	guestID, err := uuid.Parse(c.Param("guestId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid guest ID")
	}

	if err := h.householdService.RemoveMember(c.Request().Context(), tenantID, householdID, guestID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return appresponse.NotFound(c, "Member")
		}
		return appresponse.InternalError(c, "Failed to remove member")
	}

	return appresponse.NoContent(c)
}

// ListMembers handles GET /api/v1/tenants/:id/households/:householdId/members - lists members.
func (h *HouseholdHandler) ListMembers(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	householdID, err := uuid.Parse(c.Param("householdId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid household ID")
	}

	members, err := h.householdService.ListMembers(c.Request().Context(), tenantID, householdID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return appresponse.NotFound(c, "Household")
		}
		return appresponse.InternalError(c, "Failed to list members")
	}

	return appresponse.Success(c, members)
}
