package handler

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"guestflow/internal/whatsapp"
	appresponse "guestflow/pkg/response"

	"github.com/labstack/echo/v4"
)

// githubWhatsAppSender is deliberately small so the webhook can be tested
// without connecting to a real GOWA instance.
type githubWhatsAppSender interface {
	SendTo(context.Context, string, string) (whatsapp.SendReceipt, error)
}

// GitHubIssueNotificationHandler relays trusted GitHub Actions issue events
// to the platform WhatsApp device. It does not expose GOWA to the internet.
type GitHubIssueNotificationHandler struct {
	sender     githubWhatsAppSender
	token      string
	recipients []string
	allowed    map[string]struct{}
	mu         sync.Mutex
	deliveries map[string]time.Time
}

func NewGitHubIssueNotificationHandler(sender githubWhatsAppSender, token, recipients, allowedRepositories string) *GitHubIssueNotificationHandler {
	allowed := make(map[string]struct{})
	for _, repository := range splitNotificationRecipients(allowedRepositories) {
		allowed[repository] = struct{}{}
	}
	return &GitHubIssueNotificationHandler{
		sender:     sender,
		token:      strings.TrimSpace(token),
		recipients: splitNotificationRecipients(recipients),
		allowed:    allowed,
		deliveries: make(map[string]time.Time),
	}
}

type githubIssueNotificationPayload struct {
	Action      string `json:"action"`
	Repository  string `json:"repository"`
	IssueNumber int    `json:"issue_number"`
	Title       string `json:"title"`
	IssueURL    string `json:"issue_url"`
	DeliveryID  string `json:"delivery_id"`
}

// Handle receives a small, purpose-built payload from GitHub Actions. The
// bearer token is separate from GOWA credentials and is never forwarded.
func (h *GitHubIssueNotificationHandler) Handle(c echo.Context) error {
	if h == nil || h.sender == nil || h.token == "" || len(h.recipients) == 0 || len(h.allowed) == 0 {
		return appresponse.ServiceUnavailable(c, "GitHub WhatsApp notification is not configured")
	}

	if !validBearerToken(h.token, c.Request().Header.Get("Authorization")) {
		return appresponse.Unauthorized(c, "Invalid GitHub webhook token")
	}

	body, err := io.ReadAll(io.LimitReader(c.Request().Body, 64<<10))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid webhook body")
	}
	var payload githubIssueNotificationPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return appresponse.BadRequest(c, "Invalid GitHub issue payload")
	}
	payload.Action = strings.TrimSpace(payload.Action)
	payload.Repository = strings.TrimSpace(payload.Repository)
	payload.Title = strings.TrimSpace(payload.Title)
	payload.IssueURL = strings.TrimSpace(payload.IssueURL)
	if payload.Action == "" || payload.Title == "" || payload.IssueNumber <= 0 {
		return appresponse.BadRequest(c, "GitHub issue action, number, and title are required")
	}
	if payload.Repository == "" {
		return appresponse.BadRequest(c, "GitHub repository is required")
	}
	if payload.DeliveryID == "" {
		return appresponse.BadRequest(c, "GitHub delivery ID is required")
	}
	if _, ok := h.allowed[payload.Repository]; !ok {
		return appresponse.Forbidden(c, "GitHub repository is not allowed for this relay")
	}
	if h.isDuplicate(payload.DeliveryID) {
		return appresponse.Success(c, map[string]interface{}{"duplicate": true, "delivery_id": payload.DeliveryID})
	}

	message := formatGitHubIssueMessage(payload)
	sent := 0
	failed := 0
	for _, recipient := range h.recipients {
		if _, sendErr := h.sender.SendTo(c.Request().Context(), recipient, message); sendErr != nil {
			failed++
			continue
		}
		sent++
	}
	if sent == 0 {
		h.releaseDelivery(payload.DeliveryID)
		return appresponse.ServiceUnavailable(c, "GitHub WhatsApp notification could not be sent")
	}

	return appresponse.Success(c, map[string]int{
		"attempted": len(h.recipients),
		"sent":      sent,
		"failed":    failed,
	})
}

func (h *GitHubIssueNotificationHandler) isDuplicate(deliveryID string) bool {
	now := time.Now().UTC()
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, createdAt := range h.deliveries {
		if now.Sub(createdAt) > 24*time.Hour {
			delete(h.deliveries, id)
		}
	}
	if _, exists := h.deliveries[deliveryID]; exists {
		return true
	}
	h.deliveries[deliveryID] = now
	return false
}

func (h *GitHubIssueNotificationHandler) releaseDelivery(deliveryID string) {
	h.mu.Lock()
	delete(h.deliveries, deliveryID)
	h.mu.Unlock()
}

func validBearerToken(expected, authorization string) bool {
	const prefix = "Bearer "
	if !strings.HasPrefix(authorization, prefix) {
		return false
	}
	provided := strings.TrimSpace(strings.TrimPrefix(authorization, prefix))
	if provided == "" || len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func splitNotificationRecipients(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || unicode.IsSpace(r)
	})
	recipients := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			recipients = append(recipients, value)
		}
	}
	return recipients
}

func formatGitHubIssueMessage(payload githubIssueNotificationPayload) string {
	var builder strings.Builder
	builder.WriteString("GuestFlow GitHub\n")
	builder.WriteString("Aksi: ")
	builder.WriteString(payload.Action)
	if payload.Repository != "" {
		builder.WriteString("\nRepo: ")
		builder.WriteString(payload.Repository)
	}
	builder.WriteString("\nIssue #")
	builder.WriteString(strconv.Itoa(payload.IssueNumber))
	builder.WriteString(": ")
	builder.WriteString(payload.Title)
	if payload.IssueURL != "" {
		builder.WriteString("\n")
		builder.WriteString(payload.IssueURL)
	}
	return builder.String()
}
