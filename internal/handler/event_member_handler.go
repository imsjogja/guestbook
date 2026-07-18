package handler

import (
	"errors"

	"guestflow/internal/domain"
	"guestflow/internal/service"
	apperrors "guestflow/pkg/errors"
	"guestflow/pkg/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type EventMemberHandler struct {
	memberService *service.EventMemberService
	accessService *service.EventAccessService
}

func NewEventMemberHandler(memberService *service.EventMemberService, accessService *service.EventAccessService) *EventMemberHandler {
	return &EventMemberHandler{memberService: memberService, accessService: accessService}
}

type eventMemberResponse struct {
	ID         string                  `json:"id"`
	TenantID   string                  `json:"tenantId"`
	EventID    string                  `json:"eventId"`
	UserID     string                  `json:"userId"`
	Role       string                  `json:"role"`
	Status     string                  `json:"status"`
	AssignedAt string                  `json:"assignedAt"`
	User       eventMemberUserResponse `json:"user"`
}

type eventMemberUserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"fullName"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

func (h *EventMemberHandler) List(c echo.Context) error {
	tenantID, eventID, err := parseEventScope(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	members, err := h.memberService.List(c.Request().Context(), tenantID, eventID)
	if err != nil {
		return response.InternalError(c, "failed to list event members")
	}
	items := make([]eventMemberResponse, 0, len(members))
	for _, member := range members {
		items = append(items, mapEventMember(member))
	}
	return response.Success(c, items)
}

func (h *EventMemberHandler) Create(c echo.Context) error {
	tenantID, eventID, err := parseEventScope(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return response.Unauthorized(c, "unauthenticated")
	}
	var req domain.EventMemberCreateRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	member, err := h.memberService.Create(c.Request().Context(), tenantID, eventID, userID, req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidRole), errors.Is(err, domain.ErrInvalidInput):
			return response.BadRequest(c, "invalid event member role or user")
		case errors.Is(err, domain.ErrMembershipNotFound), errors.Is(err, domain.ErrUserNotFound):
			return response.NotFound(c, "tenant member")
		case errors.Is(err, domain.ErrAlreadyExists):
			return response.Conflict(c, "user already assigned to event")
		default:
			return response.InternalError(c, "failed to assign event member")
		}
	}
	return response.Created(c, member)
}

func (h *EventMemberHandler) UpdateRole(c echo.Context) error {
	tenantID, eventID, err := parseEventScope(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return response.BadRequest(c, "invalid user id")
	}
	changedBy, err := getUserIDFromEchoContext(c)
	if err != nil {
		return response.Unauthorized(c, "unauthenticated")
	}
	var req domain.EventMemberUpdateRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if err := h.memberService.UpdateRole(c.Request().Context(), tenantID, eventID, changedBy, targetUserID, req.Role); err != nil {
		if errors.Is(err, domain.ErrInvalidRole) {
			return response.BadRequest(c, "invalid event member role")
		}
		if errors.Is(err, domain.ErrMembershipNotFound) {
			return response.NotFound(c, "event member")
		}
		return response.InternalError(c, "failed to update event member")
	}
	return response.Success(c, map[string]string{"message": "event member role updated"})
}

func (h *EventMemberHandler) Delete(c echo.Context) error {
	tenantID, eventID, err := parseEventScope(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return response.BadRequest(c, "invalid user id")
	}
	removedBy, err := getUserIDFromEchoContext(c)
	if err != nil {
		return response.Unauthorized(c, "unauthenticated")
	}
	if err := h.memberService.Remove(c.Request().Context(), tenantID, eventID, removedBy, targetUserID); err != nil {
		if errors.Is(err, domain.ErrMembershipNotFound) {
			return response.NotFound(c, "event member")
		}
		return response.InternalError(c, "failed to remove event member")
	}
	return response.NoContent(c)
}

func (h *EventMemberHandler) Access(c echo.Context) error {
	tenantID, eventID, err := parseEventScope(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	userID, err := getUserIDFromEchoContext(c)
	if err != nil {
		return response.Unauthorized(c, "unauthenticated")
	}
	access, err := h.accessService.GetAccess(c.Request().Context(), tenantID, eventID, userID)
	if err != nil {
		return response.Error(c, apperrors.Forbidden("insufficient event permissions"))
	}
	return response.Success(c, access)
}

func mapEventMember(record service.EventMemberRecord) eventMemberResponse {
	avatar := ""
	if record.User.AvatarURL != nil {
		avatar = *record.User.AvatarURL
	}
	return eventMemberResponse{
		ID:         record.Membership.ID.String(),
		TenantID:   record.Membership.TenantID.String(),
		EventID:    record.Membership.EventID.String(),
		UserID:     record.Membership.UserID.String(),
		Role:       record.Membership.Role,
		Status:     record.Membership.Status,
		AssignedAt: record.Membership.AssignedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		User: eventMemberUserResponse{
			ID:        record.User.ID.String(),
			Email:     record.User.Email,
			FullName:  record.User.FullName,
			AvatarURL: avatar,
		},
	}
}
