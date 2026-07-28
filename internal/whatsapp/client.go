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
	"sync"
	"time"

	"guestflow/internal/config"

	"github.com/google/uuid"
)

var (
	ErrNotConfigured = errors.New("whatsapp provider is not configured")
	ErrPhoneMissing  = errors.New("guest WhatsApp number is empty")
	ErrInvalidPhone  = errors.New("guest WhatsApp number is invalid")
	ErrInvalidTarget = errors.New("WhatsApp recipient is invalid")
)

// SendReceipt records the provider acknowledgement for one send attempt.
// Delivery and read receipts are applied later by the GOWA webhook.
type SendReceipt struct {
	ExternalID  string
	HTTPStatus  int
	AttemptedAt *time.Time
	SentAt      *time.Time
}

// ProviderError preserves the provider HTTP status for an auditable failure.
type ProviderError struct {
	StatusCode int
	Message    string
}

func (e *ProviderError) Error() string { return e.Message }

var phoneDigits = regexp.MustCompile(`[^0-9]`)
var groupJID = regexp.MustCompile(`^[0-9][0-9-]{5,}@g\.us$`)

// Client sends WhatsApp messages through GOWA.
type Client struct {
	mu         sync.RWMutex
	cfg        config.WhatsAppConfig
	tenantCfg  map[uuid.UUID]config.WhatsAppConfig
	httpClient *http.Client
}

func NewClient(cfg config.WhatsAppConfig) *Client {
	return &Client{
		cfg:        cfg,
		tenantCfg:  make(map[uuid.UUID]config.WhatsAppConfig),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Configured() bool { return configured(c.configForTenant(uuid.Nil)) }

func (c *Client) SetTenantConfig(tenantID uuid.UUID, cfg config.WhatsAppConfig) {
	if c == nil || tenantID == uuid.Nil {
		return
	}
	c.mu.Lock()
	c.tenantCfg[tenantID] = cfg
	c.mu.Unlock()
}

func (c *Client) ConfiguredFor(tenantID uuid.UUID) bool {
	return configured(c.configForTenant(tenantID))
}

// NormalizePhone converts common Indonesian phone formats into GOWA format.
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

func (c *Client) Send(ctx context.Context, to, message string) (SendReceipt, error) {
	return c.sendWithConfig(ctx, c.configForTenant(uuid.Nil), to, message, false)
}

func (c *Client) SendFor(ctx context.Context, tenantID uuid.UUID, to, message string) (SendReceipt, error) {
	return c.sendWithConfig(ctx, c.configForTenant(tenantID), to, message, false)
}

// SendTo sends to a WhatsApp phone number or group JID. It is intended for
// platform-level integrations such as development notifications. Guest
// delivery should continue using SendFor, which only accepts phone numbers.
func (c *Client) SendTo(ctx context.Context, to, message string) (SendReceipt, error) {
	return c.sendWithConfig(ctx, c.configForTenant(uuid.Nil), to, message, true)
}

func (c *Client) sendWithConfig(ctx context.Context, cfg config.WhatsAppConfig, to, message string, allowGroup bool) (SendReceipt, error) {
	if !configured(cfg) {
		return SendReceipt{}, ErrNotConfigured
	}
	target, err := normalizeTarget(to, allowGroup)
	if err != nil {
		return SendReceipt{}, err
	}
	if strings.TrimSpace(message) == "" {
		return SendReceipt{}, errors.New("whatsapp message is empty")
	}

	return c.sendGOWA(ctx, cfg, target, message)
}

func normalizeTarget(raw string, allowGroup bool) (string, error) {
	value := strings.TrimSpace(raw)
	if allowGroup && strings.Contains(value, "@") {
		if groupJID.MatchString(value) {
			return value, nil
		}
		return "", ErrInvalidTarget
	}
	return NormalizePhone(value)
}

func (c *Client) sendGOWA(ctx context.Context, cfg config.WhatsAppConfig, phone, message string) (SendReceipt, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.GOWAAPIURL), "/")
	endpoint := baseURL
	if !strings.HasSuffix(endpoint, "/send/message") {
		endpoint += "/send/message"
	}
	payload, err := json.Marshal(map[string]string{"phone": phone, "message": message})
	if err != nil {
		return SendReceipt{}, fmt.Errorf("marshal GOWA request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return SendReceipt{}, fmt.Errorf("create GOWA request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-Id", cfg.GOWADeviceID)
	if cfg.GOWAUsername != "" || cfg.GOWAPassword != "" {
		req.SetBasicAuth(cfg.GOWAUsername, cfg.GOWAPassword)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return SendReceipt{}, fmt.Errorf("send GOWA request: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 64*1024))
	if err != nil {
		return SendReceipt{}, fmt.Errorf("read GOWA response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return SendReceipt{HTTPStatus: res.StatusCode}, &ProviderError{StatusCode: res.StatusCode, Message: providerErrorMessage(body, res.Status)}
	}

	response, err := parseGOWAResponse(body)
	if err != nil {
		return SendReceipt{HTTPStatus: res.StatusCode}, &ProviderError{StatusCode: res.StatusCode, Message: err.Error()}
	}
	if response.Code != "" && !strings.EqualFold(response.Code, "SUCCESS") {
		message := strings.TrimSpace(response.Message)
		if message == "" {
			message = "GOWA rejected the message"
		}
		return SendReceipt{HTTPStatus: res.StatusCode}, &ProviderError{StatusCode: res.StatusCode, Message: message}
	}
	now := time.Now().UTC()
	return SendReceipt{ExternalID: response.Results.MessageID, HTTPStatus: res.StatusCode, AttemptedAt: &now, SentAt: &now}, nil
}

type gowaResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Results struct {
		MessageID string `json:"message_id"`
	} `json:"results"`
}

func parseGOWAResponse(body []byte) (gowaResponse, error) {
	var response gowaResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return gowaResponse{}, fmt.Errorf("decode GOWA response: %w", err)
	}
	return response, nil
}

func (c *Client) configForTenant(tenantID uuid.UUID) config.WhatsAppConfig {
	if c == nil {
		return config.WhatsAppConfig{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if tenantID != uuid.Nil {
		if cfg, ok := c.tenantCfg[tenantID]; ok {
			return cfg
		}
	}
	return c.cfg
}

func configured(cfg config.WhatsAppConfig) bool {
	if !cfg.Enabled {
		return false
	}
	return strings.TrimSpace(cfg.GOWAAPIURL) != "" && strings.TrimSpace(cfg.GOWADeviceID) != "" && ((cfg.GOWAUsername == "") == (cfg.GOWAPassword == ""))
}

func providerErrorMessage(body []byte, status string) string {
	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &response) == nil {
		if strings.TrimSpace(response.Error) != "" {
			return response.Error
		}
		if strings.TrimSpace(response.Message) != "" {
			return response.Message
		}
	}
	return fmt.Sprintf("whatsapp provider returned %s: %s", status, strings.TrimSpace(string(body)))
}
