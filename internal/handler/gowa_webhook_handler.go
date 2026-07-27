package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"guestflow/internal/domain"
	"guestflow/internal/repository"
	appresponse "guestflow/pkg/response"

	"github.com/labstack/echo/v4"
)

// GOWAWebhookHandler applies GOWA delivery/read acknowledgements to messages.
type GOWAWebhookHandler struct {
	commRepo *repository.CommunicationRepository
	secret   string
}

func NewGOWAWebhookHandler(commRepo *repository.CommunicationRepository, secret string) *GOWAWebhookHandler {
	return &GOWAWebhookHandler{commRepo: commRepo, secret: strings.TrimSpace(secret)}
}

type gowaWebhookPayload struct {
	Event     string `json:"event"`
	Timestamp string `json:"timestamp"`
	Payload   struct {
		IDs         []string `json:"ids"`
		ReceiptType string   `json:"receipt_type"`
	} `json:"payload"`
}

// Handle receives GOWA's signed webhook payload. It deliberately responds
// quickly and ignores unknown message IDs so retries cannot create failures.
func (h *GOWAWebhookHandler) Handle(c echo.Context) error {
	if h == nil || h.commRepo == nil || h.secret == "" {
		return appresponse.ServiceUnavailable(c, "GOWA webhook is not configured")
	}

	body, err := io.ReadAll(io.LimitReader(c.Request().Body, 2<<20))
	if err != nil {
		return appresponse.BadRequest(c, "Invalid webhook body")
	}
	if !verifyGOWASignature(h.secret, body, c.Request().Header.Get("X-Hub-Signature-256")) {
		return appresponse.Unauthorized(c, "Invalid webhook signature")
	}

	var payload gowaWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return appresponse.BadRequest(c, "Invalid webhook payload")
	}
	if payload.Event != "message.ack" {
		return appresponse.Success(c, map[string]int{"updated": 0})
	}
	status := strings.ToLower(strings.TrimSpace(payload.Payload.ReceiptType))
	if status != domain.MessageStatusDelivered && status != domain.MessageStatusRead {
		return appresponse.Success(c, map[string]int{"updated": 0})
	}

	ackAt := time.Now().UTC()
	if parsed, parseErr := time.Parse(time.RFC3339Nano, payload.Timestamp); parseErr == nil {
		ackAt = parsed.UTC()
	}
	updated := 0
	for _, externalID := range payload.Payload.IDs {
		externalID = strings.TrimSpace(externalID)
		if externalID == "" {
			continue
		}
		message, findErr := h.commRepo.GetMessageByExternalID(c.Request().Context(), externalID)
		if findErr != nil {
			if errors.Is(findErr, domain.ErrMessageNotFound) {
				continue
			}
			return appresponse.InternalError(c, "Failed to find message for webhook")
		}
		if message.Status == domain.MessageStatusRead || (message.Status == domain.MessageStatusDelivered && status == domain.MessageStatusDelivered) || message.Status == domain.MessageStatusFailed {
			continue
		}

		deliveredAt := message.DeliveredAt
		readAt := message.ReadAt
		if deliveredAt == nil {
			deliveredAt = &ackAt
		}
		if status == domain.MessageStatusRead {
			readAt = &ackAt
		}
		sentAt := message.SentAt
		externalIDCopy := externalID
		if err := h.commRepo.UpdateMessageStatus(c.Request().Context(), message.TenantID, message.ID, status, sentAt, deliveredAt, readAt, message.FailedAt, message.ErrorMessage, &externalIDCopy, message.ProviderHTTPStatus, message.Cost); err != nil {
			return appresponse.InternalError(c, "Failed to update message status")
		}
		updated++
	}

	return appresponse.Success(c, map[string]int{"updated": updated})
}

func verifyGOWASignature(secret string, body []byte, header string) bool {
	header = strings.TrimSpace(header)
	if !strings.HasPrefix(header, "sha256=") {
		return false
	}
	want, err := hex.DecodeString(strings.TrimPrefix(header, "sha256="))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hmac.Equal(want, mac.Sum(nil))
}
