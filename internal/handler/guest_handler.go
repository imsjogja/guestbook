package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"guestflow/internal/domain"
	"guestflow/internal/service"
	appresponse "guestflow/pkg/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// GuestHandler handles HTTP requests for guest operations.
type GuestHandler struct {
	guestService *service.GuestService
}

// NewGuestHandler creates a new GuestHandler.
func NewGuestHandler(guestService *service.GuestService) *GuestHandler {
	return &GuestHandler{
		guestService: guestService,
	}
}

// Create handles POST /api/v1/tenants/:id/guests - creates a new guest.
func (h *GuestHandler) Create(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	var req domain.GuestCreateRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	guest, err := h.guestService.Create(c.Request().Context(), tenantID, userID, req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAlreadyExists):
			return appresponse.Conflict(c, "Guest with this phone or email already exists")
		case errors.Is(err, domain.ErrInvalidInput):
			return appresponse.ValidationError(c, err.Error())
		default:
			return appresponse.InternalError(c, "Failed to create guest")
		}
	}

	return appresponse.Created(c, guest)
}

// List handles GET /api/v1/tenants/:id/guests - lists guests with filters.
func (h *GuestHandler) List(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	// Parse pagination
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

	// Parse optional status filter
	var status *bool
	if statusParam := c.QueryParam("status"); statusParam != "" {
		s, err := strconv.ParseBool(statusParam)
		if err == nil {
			status = &s
		}
	}

	params := domain.GuestListParams{
		TenantID:  tenantID,
		Search:    strings.TrimSpace(c.QueryParam("search")),
		GuestType: strings.TrimSpace(c.QueryParam("type")),
		Segment:   strings.TrimSpace(c.QueryParam("segment")),
		Status:    status,
		Page:      page,
		PerPage:   perPage,
	}

	guests, total, err := h.guestService.List(c.Request().Context(), params)
	if err != nil {
		return appresponse.InternalError(c, "Failed to list guests")
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	return appresponse.Paginated(c, guests, appresponse.Meta{
		CurrentPage: page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
	})
}

// Get handles GET /api/v1/tenants/:id/guests/:guestId - retrieves a guest.
func (h *GuestHandler) Get(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	guestID, err := uuid.Parse(c.Param("guestId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid guest ID")
	}

	guest, err := h.guestService.Get(c.Request().Context(), tenantID, guestID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return appresponse.NotFound(c, "Guest")
		}
		return appresponse.InternalError(c, "Failed to retrieve guest")
	}

	return appresponse.Success(c, guest)
}

// Update handles PATCH /api/v1/tenants/:id/guests/:guestId - updates a guest.
func (h *GuestHandler) Update(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	guestID, err := uuid.Parse(c.Param("guestId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid guest ID")
	}

	var req domain.GuestUpdateRequest
	if err := c.Bind(&req); err != nil {
		return appresponse.BadRequest(c, "Invalid request body")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	guest, err := h.guestService.Update(c.Request().Context(), tenantID, guestID, userID, req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			return appresponse.NotFound(c, "Guest")
		case errors.Is(err, domain.ErrAlreadyExists):
			return appresponse.Conflict(c, "Guest with this phone or email already exists")
		default:
			return appresponse.InternalError(c, "Failed to update guest")
		}
	}

	return appresponse.Success(c, guest)
}

// Delete handles DELETE /api/v1/tenants/:id/guests/:guestId - soft-deletes a guest.
func (h *GuestHandler) Delete(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	guestID, err := uuid.Parse(c.Param("guestId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid guest ID")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	if err := h.guestService.Delete(c.Request().Context(), tenantID, guestID, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return appresponse.NotFound(c, "Guest")
		}
		return appresponse.InternalError(c, "Failed to delete guest")
	}

	return appresponse.NoContent(c)
}

// ImportCSV handles POST /api/v1/tenants/:id/guests/import - imports guests from CSV.
func (h *GuestHandler) ImportCSV(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return appresponse.BadRequest(c, "CSV file is required (form field: 'file')")
	}

	src, err := file.Open()
	if err != nil {
		return appresponse.InternalError(c, "Failed to read uploaded file")
	}
	defer src.Close()

	// Read file content
	content := make([]byte, file.Size)
	_, err = src.Read(content)
	if err != nil {
		return appresponse.InternalError(c, "Failed to read file content")
	}

	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return appresponse.Unauthorized(c, "Authentication required")
	}

	result, err := h.guestService.ImportCSV(c.Request().Context(), tenantID, userID, content)
	if err != nil {
		return appresponse.InternalError(c, "Failed to import guests: "+err.Error())
	}

	return appresponse.Success(c, result)
}

// DownloadTemplate handles GET /api/v1/tenants/:id/guests/import/template - downloads CSV template.
func (h *GuestHandler) DownloadTemplate(c echo.Context) error {
	template := csvTemplateBytes()

	c.Response().Header().Set("Content-Type", "text/csv; charset=utf-8")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="guest_import_template.csv"`)
	return c.Blob(http.StatusOK, "text/csv", template)
}

// csvTemplateBytes returns the CSV template content as bytes.
func csvTemplateBytes() []byte {
	return []byte("full_name,nickname,phone,email,address,city,country,guest_type,segment,institution,title,relationship,pic,accessibility_needs,dietary_restrictions,allergies,notes\n")
}

// Search handles GET /api/v1/tenants/:id/guests/search - searches guests.
func (h *GuestHandler) Search(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	query := strings.TrimSpace(c.QueryParam("q"))
	if query == "" {
		return appresponse.BadRequest(c, "Search query 'q' is required")
	}

	guests, err := h.guestService.Search(c.Request().Context(), tenantID, query)
	if err != nil {
		return appresponse.InternalError(c, "Failed to search guests")
	}

	return appresponse.Success(c, guests)
}
