package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"guestflow/internal/audit"
	"guestflow/internal/config"
	"guestflow/internal/domain"
	"guestflow/internal/repository"
	"guestflow/internal/whatsapp"

	"github.com/google/uuid"
)

const whatsappIntegrationSettingsKey = "whatsapp"

var ErrWhatsAppIntegrationInvalid = errors.New("invalid WhatsApp integration settings")
var ErrWhatsAppDeviceConflict = errors.New("WhatsApp device is already assigned to another tenant")
var ErrWhatsAppAuthInvalid = errors.New("GOWA Basic Auth was rejected")
var ErrWhatsAppNotReady = errors.New("WhatsApp is not connected")

// WhatsAppIntegrationUpdateRequest contains tenant-level WhatsApp changes.
type WhatsAppIntegrationUpdateRequest struct {
	Enabled  *bool  `json:"enabled"`
	DeviceID string `json:"device_id,omitempty"`
}

// WhatsAppIntegrationStatus is the tenant-safe WhatsApp connection status.
type WhatsAppIntegrationStatus struct {
	Enabled    bool                     `json:"enabled"`
	Configured bool                     `json:"configured"`
	DeviceID   string                   `json:"device_id,omitempty"`
	Connection WhatsAppConnectionStatus `json:"connection"`
}

// WhatsAppConnectionStatus is the live device state reported by GOWA.
type WhatsAppConnectionStatus struct {
	State       string `json:"state"`
	Connected   bool   `json:"connected"`
	LoggedIn    bool   `json:"logged_in"`
	JID         string `json:"jid,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Error       string `json:"error,omitempty"`
}

// WhatsAppPairingStatus contains the short-lived QR pairing session details.
type WhatsAppPairingStatus struct {
	DeviceID   string `json:"device_id"`
	QRURL      string `json:"qr_url"`
	QRDuration int    `json:"qr_duration,omitempty"`
}

// WhatsAppTestSendResult is returned by the connection test and is not stored
// as a guest communication message.
type WhatsAppTestSendResult struct {
	To             string     `json:"to"`
	ExternalID     string     `json:"external_id,omitempty"`
	ProviderStatus int        `json:"provider_status,omitempty"`
	SentAt         *time.Time `json:"sent_at,omitempty"`
}

// WhatsAppConfigProvider lets communication flows resolve tenant settings
// without coupling them to the HTTP handler.
type WhatsAppConfigProvider interface {
	ResolveWhatsAppConfig(ctx context.Context, tenantID uuid.UUID) (config.WhatsAppConfig, error)
}

// WhatsAppIntegrationService persists tenant device settings and applies them
// to the in-memory GOWA client immediately.
type WhatsAppIntegrationService struct {
	tenantRepo *repository.TenantRepository
	client     *whatsapp.Client
	fallback   config.WhatsAppConfig
	audit      *audit.Service
	mu         sync.RWMutex
	qrPaths    map[uuid.UUID]string
	httpClient *http.Client
}

func NewWhatsAppIntegrationService(
	tenantRepo *repository.TenantRepository,
	client *whatsapp.Client,
	fallback config.WhatsAppConfig,
	auditService *audit.Service,
) *WhatsAppIntegrationService {
	return &WhatsAppIntegrationService{
		tenantRepo: tenantRepo,
		client:     client,
		fallback:   fallback,
		audit:      auditService,
		qrPaths:    make(map[uuid.UUID]string),
		httpClient: &http.Client{Timeout: 8 * time.Second},
	}
}

// ResolveWhatsAppConfig loads the tenant override and falls back to the
// process environment for tenants that have not configured an override.
func (s *WhatsAppIntegrationService) ResolveWhatsAppConfig(ctx context.Context, tenantID uuid.UUID) (config.WhatsAppConfig, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return config.WhatsAppConfig{}, fmt.Errorf("load WhatsApp integration: %w", err)
	}

	cfg := s.fallback
	wa, exists := readWhatsAppSettings(tenant.Settings)
	tenantDeviceID := ""
	if exists {
		if enabled, ok := wa["enabled"].(bool); ok {
			cfg.Enabled = enabled
		}
		if deviceID, ok := wa["device_id"].(string); ok && strings.TrimSpace(deviceID) != "" {
			tenantDeviceID = strings.TrimSpace(deviceID)
			cfg.GOWADeviceID = tenantDeviceID
		}
	}
	if tenantDeviceID == "" {
		// Environment credentials may be shared by the process, but every
		// tenant gets an isolated GOWA session/device slot.
		cfg.GOWADeviceID = defaultTenantDeviceID(tenantID)
	}
	if strings.TrimSpace(cfg.GOWADeviceID) != "" {
		inUse, checkErr := s.tenantRepo.WhatsAppDeviceIDInUse(ctx, cfg.GOWADeviceID, tenantID)
		if checkErr != nil {
			return config.WhatsAppConfig{}, checkErr
		}
		if inUse {
			return config.WhatsAppConfig{}, fmt.Errorf("%w: %s", ErrWhatsAppDeviceConflict, cfg.GOWADeviceID)
		}
	}
	return cfg, nil
}

// GetStatus returns tenant-safe GOWA connection state.
func (s *WhatsAppIntegrationService) GetStatus(ctx context.Context, tenantID uuid.UUID) (*WhatsAppIntegrationStatus, error) {
	cfg, err := s.ResolveWhatsAppConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	status := &WhatsAppIntegrationStatus{
		Enabled:    cfg.Enabled,
		Configured: cfg.Enabled && configuredForStatus(cfg),
		DeviceID:   cfg.GOWADeviceID,
	}
	status.Connection = s.getConnectionStatus(ctx, cfg)
	return status, nil
}

// Update saves tenant WhatsApp settings and applies them without restarting
// the process. Server URL and authentication remain platform-managed.
func (s *WhatsAppIntegrationService) Update(ctx context.Context, tenantID, userID uuid.UUID, req WhatsAppIntegrationUpdateRequest) (*WhatsAppIntegrationStatus, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load tenant for WhatsApp integration: %w", err)
	}
	settings := cloneSettings(tenant.Settings)
	integrations, _ := settings["integrations"].(map[string]interface{})
	if integrations == nil {
		integrations = make(map[string]interface{})
	}
	wa, _ := integrations[whatsappIntegrationSettingsKey].(map[string]interface{})
	if wa == nil {
		wa = make(map[string]interface{})
	}
	if req.Enabled != nil {
		wa["enabled"] = *req.Enabled
	}
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		if existing, ok := wa["device_id"].(string); ok {
			deviceID = strings.TrimSpace(existing)
		}
	}
	if deviceID == "" {
		deviceID = defaultTenantDeviceID(tenantID)
	}
	if err := validateWhatsAppDeviceID(deviceID); err != nil {
		return nil, err
	}
	inUse, err := s.tenantRepo.WhatsAppDeviceIDInUse(ctx, deviceID, tenantID)
	if err != nil {
		return nil, err
	}
	if inUse {
		return nil, fmt.Errorf("%w: %s", ErrWhatsAppDeviceConflict, deviceID)
	}
	wa["device_id"] = deviceID
	// Remove settings from the retired provider and tenant-managed GOWA
	// credentials whenever this integration is saved.
	for _, key := range []string{"provider", "api_url", "username", "password", "account_token", "sender_token", "phone_number_id", "access_token", "webhook_verify_token"} {
		delete(wa, key)
	}
	wa["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	integrations[whatsappIntegrationSettingsKey] = wa
	settings["integrations"] = integrations
	tenant.Settings = settings
	tenant.Touch()
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("save WhatsApp integration: %w", err)
	}

	cfg, err := s.ResolveWhatsAppConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	s.client.SetTenantConfig(tenantID, cfg)
	if s.audit != nil {
		_ = s.audit.LogWithUser(ctx, userID, tenantID, domain.AuditActionUpdate, domain.EntityTypeSetting, tenantID, nil, map[string]interface{}{
			"integration": "whatsapp",
			"enabled":     cfg.Enabled,
			"source":      "tenant",
		})
	}
	return s.GetStatus(ctx, tenantID)
}

// StartPairing creates the configured device slot when necessary and starts a
// GOWA QR pairing session. The QR image itself is served by the authenticated
// GuestFlow endpoint so the browser never needs direct GOWA access.
func (s *WhatsAppIntegrationService) StartPairing(ctx context.Context, tenantID uuid.UUID) (*WhatsAppPairingStatus, error) {
	cfg, err := s.ResolveWhatsAppConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled || !configuredForStatus(cfg) {
		return nil, fmt.Errorf("%w: GOWA is not configured", ErrWhatsAppIntegrationInvalid)
	}
	if err := s.ensureGOWADevice(ctx, cfg); err != nil {
		return nil, err
	}

	// Use the app-level login endpoint for compatibility with GOWA v8.x.
	// Device-scoped requests select the configured session via X-Device-Id.
	body, statusCode, err := s.doGOWA(ctx, cfg, http.MethodGet, "/app/login", nil)
	if err != nil {
		return nil, err
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return nil, gowaResponseError(body, statusCode)
	}
	var response struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Results struct {
			QRLink     string `json:"qr_link"`
			QRDuration int    `json:"qr_duration"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode GOWA pairing response: %w", err)
	}
	if response.Code != "" && !strings.EqualFold(response.Code, "SUCCESS") {
		return nil, fmt.Errorf("GOWA pairing rejected: %s", firstNonEmpty(response.Message, response.Code))
	}
	qrURL, err := url.Parse(response.Results.QRLink)
	if err != nil || qrURL.Path == "" {
		return nil, errors.New("GOWA did not return a valid pairing QR")
	}
	s.mu.Lock()
	s.qrPaths[tenantID] = qrURL.RequestURI()
	s.mu.Unlock()
	return &WhatsAppPairingStatus{DeviceID: cfg.GOWADeviceID, QRDuration: response.Results.QRDuration}, nil
}

// GetPairingQR proxies the current GOWA QR image through the backend.
func (s *WhatsAppIntegrationService) GetPairingQR(ctx context.Context, tenantID uuid.UUID) ([]byte, string, error) {
	cfg, err := s.ResolveWhatsAppConfig(ctx, tenantID)
	if err != nil {
		return nil, "", err
	}
	s.mu.RLock()
	path := s.qrPaths[tenantID]
	s.mu.RUnlock()
	if path == "" {
		return nil, "", errors.New("GOWA pairing QR is not available; start pairing first")
	}
	body, statusCode, err := s.doGOWA(ctx, cfg, http.MethodGet, path, nil)
	if err != nil {
		return nil, "", err
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return nil, "", gowaResponseError(body, statusCode)
	}
	return body, "image/png", nil
}

// Logout disconnects the tenant's WhatsApp session while retaining its
// isolated device slot for a future pairing.
func (s *WhatsAppIntegrationService) Logout(ctx context.Context, tenantID uuid.UUID) (*WhatsAppIntegrationStatus, error) {
	cfg, err := s.ResolveWhatsAppConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled || !configuredForStatus(cfg) {
		return nil, ErrWhatsAppIntegrationInvalid
	}
	path := "/devices/" + url.PathEscape(cfg.GOWADeviceID) + "/logout"
	body, statusCode, err := s.doGOWA(ctx, cfg, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNotFound {
		return s.GetStatus(ctx, tenantID)
	}
	if statusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("%w: GOWA returned HTTP 401", ErrWhatsAppAuthInvalid)
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return nil, gowaResponseError(body, statusCode)
	}
	return s.GetStatus(ctx, tenantID)
}

// SendTest sends a small, explicit connection test without creating an
// invitation or communication history row.
func (s *WhatsAppIntegrationService) SendTest(ctx context.Context, tenantID uuid.UUID, to, message string) (*WhatsAppTestSendResult, error) {
	cfg, err := s.ResolveWhatsAppConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled || !configuredForStatus(cfg) {
		return nil, ErrWhatsAppIntegrationInvalid
	}
	connection := s.getConnectionStatus(ctx, cfg)
	if !connection.LoggedIn {
		return nil, fmt.Errorf("%w: %s", ErrWhatsAppNotReady, connection.State)
	}
	s.client.SetTenantConfig(tenantID, cfg)
	receipt, err := s.client.SendFor(ctx, tenantID, to, message)
	if err != nil {
		return nil, err
	}
	return &WhatsAppTestSendResult{
		To:             to,
		ExternalID:     receipt.ExternalID,
		ProviderStatus: receipt.HTTPStatus,
		SentAt:         receipt.SentAt,
	}, nil
}

func (s *WhatsAppIntegrationService) getConnectionStatus(ctx context.Context, cfg config.WhatsAppConfig) WhatsAppConnectionStatus {
	if !cfg.Enabled {
		return WhatsAppConnectionStatus{State: "disabled"}
	}
	if !configuredForStatus(cfg) {
		return WhatsAppConnectionStatus{State: "not_configured"}
	}
	if err := s.ensureGOWADevice(ctx, cfg); err != nil {
		if errors.Is(err, ErrWhatsAppAuthInvalid) {
			return WhatsAppConnectionStatus{State: "unauthorized", Error: "WhatsApp belum dapat diautentikasi oleh platform"}
		}
		return WhatsAppConnectionStatus{State: "unavailable", Error: err.Error()}
	}
	body, statusCode, err := s.doGOWA(ctx, cfg, http.MethodGet, "/app/status", nil)
	if err != nil {
		return WhatsAppConnectionStatus{State: "unavailable", Error: err.Error()}
	}
	var response struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Results struct {
			IsConnected bool   `json:"is_connected"`
			IsLoggedIn  bool   `json:"is_logged_in"`
			JID         string `json:"jid"`
		} `json:"results"`
	}
	if json.Unmarshal(body, &response) != nil {
		return WhatsAppConnectionStatus{State: "unavailable", Error: "invalid GOWA status response"}
	}
	if statusCode == http.StatusNotFound || !strings.EqualFold(response.Code, "SUCCESS") {
		return WhatsAppConnectionStatus{State: "not_registered", Error: firstNonEmpty(response.Message, response.Code)}
	}
	state := "disconnected"
	if response.Results.IsLoggedIn {
		state = "logged_in"
	} else if response.Results.IsConnected {
		state = "connected"
	}
	return WhatsAppConnectionStatus{
		State:       state,
		Connected:   response.Results.IsConnected,
		LoggedIn:    response.Results.IsLoggedIn,
		JID:         response.Results.JID,
		PhoneNumber: phoneNumberFromJID(response.Results.JID),
	}
}

func (s *WhatsAppIntegrationService) ensureGOWADevice(ctx context.Context, cfg config.WhatsAppConfig) error {
	path := "/devices/" + url.PathEscape(cfg.GOWADeviceID)
	body, statusCode, err := s.doGOWA(ctx, cfg, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
		return nil
	}
	if !isGOWADeviceNotFound(body, statusCode) {
		if statusCode == http.StatusUnauthorized {
			return fmt.Errorf("%w: GOWA returned HTTP 401", ErrWhatsAppAuthInvalid)
		}
		return gowaResponseError(body, statusCode)
	}
	payload, _ := json.Marshal(map[string]string{"device_id": cfg.GOWADeviceID})
	body, statusCode, err = s.doGOWA(ctx, cfg, http.MethodPost, "/devices", payload)
	if err != nil {
		return err
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		if statusCode == http.StatusUnauthorized {
			return fmt.Errorf("%w: GOWA returned HTTP 401", ErrWhatsAppAuthInvalid)
		}
		return gowaResponseError(body, statusCode)
	}
	return nil
}

func isGOWADeviceNotFound(body []byte, statusCode int) bool {
	if statusCode == http.StatusNotFound {
		return true
	}
	var response struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &response) != nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(response.Message))
	return strings.Contains(message, "device") && strings.Contains(message, "not found")
}

func (s *WhatsAppIntegrationService) doGOWA(ctx context.Context, cfg config.WhatsAppConfig, method, path string, body []byte) ([]byte, int, error) {
	baseURL := strings.TrimRight(cfg.GOWAAPIURL, "/")
	if baseURL == "" {
		return nil, 0, errors.New("GOWA API URL is empty")
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("create GOWA request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cfg.GOWADeviceID != "" {
		req.Header.Set("X-Device-Id", cfg.GOWADeviceID)
	}
	if cfg.GOWAUsername != "" || cfg.GOWAPassword != "" {
		req.SetBasicAuth(cfg.GOWAUsername, cfg.GOWAPassword)
	}
	response, err := s.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request GOWA: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return nil, response.StatusCode, fmt.Errorf("read GOWA response: %w", err)
	}
	return data, response.StatusCode, nil
}

func gowaResponseError(body []byte, statusCode int) error {
	var response struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &response) == nil && (response.Code != "" || response.Message != "") {
		return fmt.Errorf("GOWA returned %s: %s", firstNonEmpty(response.Code, "ERROR"), firstNonEmpty(response.Message, "request failed"))
	}
	return fmt.Errorf("GOWA returned HTTP %d", statusCode)
}

func readWhatsAppSettings(settings domain.JSONMap) (map[string]interface{}, bool) {
	integrations, ok := settings["integrations"].(map[string]interface{})
	if !ok {
		return nil, false
	}
	wa, ok := integrations[whatsappIntegrationSettingsKey].(map[string]interface{})
	return wa, ok
}

func cloneSettings(settings domain.JSONMap) domain.JSONMap {
	if settings == nil {
		return make(domain.JSONMap)
	}
	cloned := make(domain.JSONMap, len(settings))
	for key, value := range settings {
		cloned[key] = value
	}
	return cloned
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func phoneNumberFromJID(jid string) string {
	value := strings.TrimSpace(jid)
	if value == "" {
		return ""
	}
	if at := strings.IndexByte(value, '@'); at >= 0 {
		value = value[:at]
	}
	if colon := strings.IndexByte(value, ':'); colon >= 0 {
		value = value[:colon]
	}
	return value
}

func configuredForStatus(cfg config.WhatsAppConfig) bool {
	return strings.TrimSpace(cfg.GOWAAPIURL) != "" && strings.TrimSpace(cfg.GOWADeviceID) != "" && ((cfg.GOWAUsername == "") == (cfg.GOWAPassword == ""))
}

func defaultTenantDeviceID(tenantID uuid.UUID) string {
	return "guestflow-" + strings.ReplaceAll(tenantID.String(), "-", "")
}

func validateWhatsAppDeviceID(deviceID string) error {
	if len(deviceID) < 3 || len(deviceID) > 64 {
		return fmt.Errorf("%w: device_id must be between 3 and 64 characters", ErrWhatsAppIntegrationInvalid)
	}
	for _, char := range deviceID {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' && char != '_' && char != '.' {
			return fmt.Errorf("%w: device_id contains unsupported characters", ErrWhatsAppIntegrationInvalid)
		}
	}
	return nil
}
