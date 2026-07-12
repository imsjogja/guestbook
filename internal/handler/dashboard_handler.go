package handler

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"time"

	"guestflow/internal/domain"
	"guestflow/internal/service"
	appresponse "guestflow/pkg/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// DashboardHandler handles HTTP requests for dashboard operations.
type DashboardHandler struct {
	dashboardService *service.DashboardService
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(dashboardService *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
	}
}

// GetDashboard handles GET /api/v1/tenants/:id/events/:eventId/dashboard - full dashboard.
func (h *DashboardHandler) GetDashboard(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	dashboard, err := h.dashboardService.GetEventDashboard(c.Request().Context(), tenantID, eventID)
	if err != nil {
		if stderrors.Is(err, domain.ErrEventNotFound) {
			return appresponse.NotFound(c, "Event")
		}
		return appresponse.InternalError(c, "Failed to load dashboard")
	}

	return appresponse.Success(c, dashboard)
}

// StreamDashboard handles GET /api/v1/tenants/:id/events/:eventId/dashboard/stream - SSE for real-time updates.
func (h *DashboardHandler) StreamDashboard(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid tenant ID")
	}

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid event ID")
	}

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	// Send initial dashboard data
	dashboard, err := h.dashboardService.GetEventDashboard(c.Request().Context(), tenantID, eventID)
	if err != nil {
		data, _ := json.Marshal(map[string]string{"error": "failed to load dashboard"})
		fmt.Fprintf(c.Response(), "event: error\ndata: %s\n\n", data)
		c.Response().Flush()
		return nil
	}

	initData, err := json.Marshal(dashboard)
	if err != nil {
		data, _ := json.Marshal(map[string]string{"error": "failed to encode dashboard"})
		fmt.Fprintf(c.Response(), "event: error\ndata: %s\n\n", data)
		c.Response().Flush()
		return nil
	}

	fmt.Fprintf(c.Response(), "event: dashboard\ndata: %s\n\n", initData)
	c.Response().Flush()

	// Set up a ticker for periodic updates (every 10 seconds)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Set up request context done channel for client disconnect detection
	ctx := c.Request().Context()
	done := ctx.Done()

	for {
		select {
		case <-ticker.C:
			updatedDashboard, err := h.dashboardService.GetEventDashboard(c.Request().Context(), tenantID, eventID)
			if err != nil {
				errData, _ := json.Marshal(map[string]string{"error": "failed to refresh dashboard"})
				fmt.Fprintf(c.Response(), "event: error\ndata: %s\n\n", errData)
				c.Response().Flush()
				continue
			}

			updateData, err := json.Marshal(updatedDashboard)
			if err != nil {
				errData, _ := json.Marshal(map[string]string{"error": "failed to encode update"})
				fmt.Fprintf(c.Response(), "event: error\ndata: %s\n\n", errData)
				c.Response().Flush()
				continue
			}

			fmt.Fprintf(c.Response(), "event: dashboard\ndata: %s\n\n", updateData)
			c.Response().Flush()

		case <-done:
			// Client disconnected
			return nil
		}
	}
}
