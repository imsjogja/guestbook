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
func (c *Client) Send(ctx context.Context, to, message string) (string, error) {
	if !c.Configured() {
		return "", ErrNotConfigured
	}
	phone, err := NormalizePhone(to)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(message) == "" {
		return "", errors.New("whatsapp message is empty")
	}

	payload, err := json.Marshal(map[string]string{"to": phone, "message": message})
	if err != nil {
		return "", fmt.Errorf("marshal whatsapp request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.APIURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create whatsapp request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.AccountToken)
	req.Header.Set("X-Sender-Token", c.cfg.SenderToken)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send whatsapp request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 64*1024))
	if err != nil {
		return "", fmt.Errorf("read whatsapp response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("whatsapp provider returned %s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	return providerMessageID(body), nil
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
