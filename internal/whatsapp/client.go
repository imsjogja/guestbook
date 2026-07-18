// Package whatsapp contains the provider adapter used for WhatsApp delivery.
package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"guestflow/internal/config"
)

var (
	ErrNotConfigured = errors.New("whatsapp provider is not configured")
	ErrPhoneMissing  = errors.New("guest WhatsApp number is empty")
	ErrInvalidPhone  = errors.New("guest WhatsApp number is invalid")
)

// SendReceipt records the provider acknowledgement for one send attempt.
// The provider's explicit ok field determines acceptance; delivery and read
// receipts require a separate provider callback.
type SendReceipt struct {
	ExternalID  string
	HTTPStatus  int
	SenderID    string
	AttemptedAt *time.Time
	SentAt      *time.Time
}

// ProviderError preserves the provider HTTP status for an auditable failure.
type ProviderError struct {
	StatusCode int
	Message    string
}

func (e *ProviderError) Error() string {
	return e.Message
}

type providerSendResponse struct {
	OK          *bool  `json:"ok"`
	Error       string `json:"error"`
	SenderID    string `json:"senderId"`
	To          string `json:"to"`
	AttemptedAt string `json:"attemptedAt"`
	SentAt      string `json:"sentAt"`
}

var phoneDigits = regexp.MustCompile(`[^0-9]`)

// Client sends WhatsApp messages through the configured provider.
type Client struct {
	cfg        config.WhatsAppConfig
	httpClient *http.Client
}

// NewClient creates a Blastr-compatible WhatsApp client.
func NewClient(cfg config.WhatsAppConfig) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Configured reports whether the provider can accept a delivery request.
func (c *Client) Configured() bool {
	return c != nil && c.cfg.Enabled && strings.TrimSpace(c.cfg.APIURL) != "" &&
		strings.TrimSpace(c.cfg.AccountToken) != "" && strings.TrimSpace(c.cfg.SenderToken) != ""
}

// NormalizePhone converts common Indonesian phone formats into provider format.
func NormalizePhone(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", ErrPhoneMissing
	}

	value = phoneDigits.ReplaceAllString(value, "")
	if strings.HasPrefix(value, "0") {
		value = "62" + strings.TrimPrefix(value, "0")
	}
	if len(value) < 10 || len(value) > 15 || !strings.HasPrefix(value, "62") {
		return "", ErrInvalidPhone
	}
	return value, nil
}

// Send posts a plain text message to the Blastr public API.
func (c *Client) Send(ctx context.Context, to, message string) (SendReceipt, error) {
	if !c.Configured() {
		return SendReceipt{}, ErrNotConfigured
	}
	phone, err := NormalizePhone(to)
	if err != nil {
		return SendReceipt{}, err
	}
	if strings.TrimSpace(message) == "" {
		return SendReceipt{}, errors.New("whatsapp message is empty")
	}

	payload, err := json.Marshal(map[string]string{"to": phone, "message": message})
	if err != nil {
		return SendReceipt{}, fmt.Errorf("marshal whatsapp request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.APIURL, bytes.NewReader(payload))
	if err != nil {
		return SendReceipt{}, fmt.Errorf("create whatsapp request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.AccountToken)
	req.Header.Set("X-Sender-Token", c.cfg.SenderToken)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return SendReceipt{}, fmt.Errorf("send whatsapp request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 64*1024))
	if err != nil {
		return SendReceipt{}, fmt.Errorf("read whatsapp response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return SendReceipt{HTTPStatus: res.StatusCode}, &ProviderError{
			StatusCode: res.StatusCode,
			Message:    providerErrorMessage(body, res.Status),
		}
	}

	providerResponse, parseErr := parseProviderResponse(body)
	if parseErr != nil {
		return SendReceipt{HTTPStatus: res.StatusCode}, &ProviderError{
			StatusCode: res.StatusCode,
			Message:    parseErr.Error(),
		}
	}
	if providerResponse.OK != nil && !*providerResponse.OK {
		message := strings.TrimSpace(providerResponse.Error)
		if message == "" {
			message = "whatsapp provider rejected the message"
		}
		return SendReceipt{HTTPStatus: res.StatusCode}, &ProviderError{
			StatusCode: res.StatusCode,
			Message:    message,
		}
	}

	return SendReceipt{
		ExternalID:  providerMessageID(body),
		HTTPStatus:  res.StatusCode,
		SenderID:    providerResponse.SenderID,
		AttemptedAt: parseProviderTime(providerResponse.AttemptedAt),
		SentAt:      parseProviderTime(providerResponse.SentAt),
	}, nil
}

func parseProviderResponse(body []byte) (providerSendResponse, error) {
	var response providerSendResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return providerSendResponse{}, fmt.Errorf("decode whatsapp provider response: %w", err)
	}
	return response, nil
}

func parseProviderTime(value string) *time.Time {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil
	}
	return &parsed
}

func providerErrorMessage(body []byte, status string) string {
	var response struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &response) == nil && strings.TrimSpace(response.Error) != "" {
		return response.Error
	}
	return fmt.Sprintf("whatsapp provider returned %s: %s", status, strings.TrimSpace(string(body)))
}

func providerMessageID(body []byte) string {
	var response map[string]any
	if json.Unmarshal(body, &response) != nil {
		return ""
	}
	for _, key := range []string{"id", "message_id", "messageId", "external_id", "externalId"} {
		if value, ok := response[key].(string); ok {
			return value
		}
	}
	if data, ok := response["data"].(map[string]any); ok {
		for _, key := range []string{"id", "message_id", "messageId", "external_id", "externalId"} {
			if value, ok := data[key].(string); ok {
				return value
			}
		}
	}
	return ""
}
