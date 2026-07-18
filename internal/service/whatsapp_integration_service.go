package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"guestflow/internal/audit"
	"guestflow/internal/config"
	"guestflow/internal/domain"
	"guestflow/internal/repository"
	"guestflow/internal/security"
	"guestflow/internal/whatsapp"

	"github.com/google/uuid"
)

const whatsappIntegrationSettingsKey = "whatsapp"

var ErrWhatsAppIntegrationInvalid = errors.New("invalid WhatsApp integration settings")

// WhatsAppIntegrationUpdateRequest contains optional changes to the provider.
// Empty tokens intentionally preserve the existing encrypted values.
type WhatsAppIntegrationUpdateRequest struct {
	Enabled           *bool  `json:"enabled"`
	APIURL            string `json:"api_url,omitempty"`
	AccountToken      string `json:"account_token,omitempty"`
	SenderToken       string `json:"sender_token,omitempty"`
	ClearAccountToken bool   `json:"clear_account_token,omitempty"`
	ClearSenderToken  bool   `json:"clear_sender_token,omitempty"`
}

// WhatsAppIntegrationStatus is safe to return to the browser. Token values
// are represented only by masked strings and configured flags.
type WhatsAppIntegrationStatus struct {
	Enabled            bool       `json:"enabled"`
	Configured         bool       `json:"configured"`
	APIURL             string     `json:"api_url"`
	AccountTokenSet    bool       `json:"account_token_set"`
	AccountTokenMasked string     `json:"account_token_masked,omitempty"`
	SenderTokenSet     bool       `json:"sender_token_set"`
	SenderTokenMasked  string     `json:"sender_token_masked,omitempty"`
	Source             string     `json:"source"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty"`
}

// WhatsAppConfigProvider lets communication flows resolve tenant settings
// without coupling them to the HTTP handler.
type WhatsAppConfigProvider interface {
	ResolveWhatsAppConfig(ctx context.Context, tenantID uuid.UUID) (config.WhatsAppConfig, error)
}

// WhatsAppIntegrationService persists encrypted tenant credentials and applies
// them to the in-memory provider client immediately.
type WhatsAppIntegrationService struct {
	tenantRepo *repository.TenantRepository
	client     *whatsapp.Client
	fallback   config.WhatsAppConfig
	secret     string
	audit      *audit.Service
}

func NewWhatsAppIntegrationService(
	tenantRepo *repository.TenantRepository,
	client *whatsapp.Client,
	fallback config.WhatsAppConfig,
	encryptionSecret string,
	auditService *audit.Service,
) *WhatsAppIntegrationService {
	return &WhatsAppIntegrationService{
		tenantRepo: tenantRepo,
		client:     client,
		fallback:   fallback,
		secret:     encryptionSecret,
		audit:      auditService,
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
	if !exists {
		return cfg, nil
	}
	if enabled, ok := wa["enabled"].(bool); ok {
		cfg.Enabled = enabled
	}
	if apiURL, ok := wa["api_url"].(string); ok && strings.TrimSpace(apiURL) != "" {
		cfg.APIURL = apiURL
	}
	if encrypted, ok := wa["account_token"].(string); ok && encrypted != "" {
		cfg.AccountToken, err = security.DecryptSecret(s.secret, encrypted)
		if err != nil {
			return config.WhatsAppConfig{}, fmt.Errorf("decrypt WhatsApp account token: %w", err)
		}
	}
	if encrypted, ok := wa["sender_token"].(string); ok && encrypted != "" {
		cfg.SenderToken, err = security.DecryptSecret(s.secret, encrypted)
		if err != nil {
			return config.WhatsAppConfig{}, fmt.Errorf("decrypt WhatsApp sender token: %w", err)
		}
	}
	return cfg, nil
}

// GetStatus returns provider state without exposing credentials.
func (s *WhatsAppIntegrationService) GetStatus(ctx context.Context, tenantID uuid.UUID) (*WhatsAppIntegrationStatus, error) {
	cfg, err := s.ResolveWhatsAppConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load WhatsApp integration status: %w", err)
	}
	wa, tenantConfigured := readWhatsAppSettings(tenant.Settings)
	status := &WhatsAppIntegrationStatus{
		Enabled:         cfg.Enabled,
		Configured:      cfg.Enabled && strings.TrimSpace(cfg.APIURL) != "" && strings.TrimSpace(cfg.AccountToken) != "" && strings.TrimSpace(cfg.SenderToken) != "",
		APIURL:          cfg.APIURL,
		AccountTokenSet: strings.TrimSpace(cfg.AccountToken) != "",
		SenderTokenSet:  strings.TrimSpace(cfg.SenderToken) != "",
		Source:          "environment",
	}
	status.AccountTokenMasked = security.MaskSecret(cfg.AccountToken)
	status.SenderTokenMasked = security.MaskSecret(cfg.SenderToken)
	if tenantConfigured {
		status.Source = "tenant"
		if updated, ok := wa["updated_at"].(string); ok {
			if parsed, parseErr := time.Parse(time.RFC3339, updated); parseErr == nil {
				status.UpdatedAt = &parsed
			}
		}
	}
	return status, nil
}

// Update saves encrypted credentials and applies them without restarting the
// process. The next send after a process restart resolves the same DB values.
func (s *WhatsAppIntegrationService) Update(ctx context.Context, tenantID, userID uuid.UUID, req WhatsAppIntegrationUpdateRequest) (*WhatsAppIntegrationStatus, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load tenant for WhatsApp integration: %w", err)
	}
	if strings.TrimSpace(s.secret) == "" {
		return nil, fmt.Errorf("%w: encryption is not configured", ErrWhatsAppIntegrationInvalid)
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
	if strings.TrimSpace(req.APIURL) != "" {
		wa["api_url"] = strings.TrimSpace(req.APIURL)
	}
	if req.ClearAccountToken {
		delete(wa, "account_token")
	} else if strings.TrimSpace(req.AccountToken) != "" {
		encrypted, encryptErr := security.EncryptSecret(s.secret, strings.TrimSpace(req.AccountToken))
		if encryptErr != nil {
			return nil, fmt.Errorf("encrypt WhatsApp account token: %w", encryptErr)
		}
		wa["account_token"] = encrypted
	}
	if req.ClearSenderToken {
		delete(wa, "sender_token")
	} else if strings.TrimSpace(req.SenderToken) != "" {
		encrypted, encryptErr := security.EncryptSecret(s.secret, strings.TrimSpace(req.SenderToken))
		if encryptErr != nil {
			return nil, fmt.Errorf("encrypt WhatsApp sender token: %w", encryptErr)
		}
		wa["sender_token"] = encrypted
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
