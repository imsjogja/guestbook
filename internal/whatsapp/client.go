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
// A successful HTTP response means the provider accepted the request; delivery
// and read receipts require a separate provider callback.
type SendReceipt struct {
	ExternalID string
	HTTPStatus int
}

// ProviderError preserves the provider HTTP status for an auditable failure.
type ProviderError struct {
	StatusCode int
	Message    string
}

func (e *ProviderError) Error() string {
	return e.Message
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
			Message:    fmt.Sprintf("whatsapp provider returned %s: %s", res.Status, strings.TrimSpace(string(body))),
		}
	}

	return SendReceipt{
		ExternalID: providerMessageID(body),
		HTTPStatus: res.StatusCode,
	}, nil
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
