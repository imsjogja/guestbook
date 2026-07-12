-- Migration: Create webhook configuration and delivery log tables
-- +goose Up
-- +goose StatementBegin

-- Webhook configurations table
CREATE TABLE IF NOT EXISTS webhook_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_id UUID REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL, -- whatsapp, email, sms
    webhook_url VARCHAR(500) NOT NULL,
    secret_key VARCHAR(255),
    auth_type VARCHAR(20) DEFAULT 'none' CHECK (auth_type IN ('none', 'bearer', 'basic', 'hmac')),
    auth_config JSONB DEFAULT '{}',
    events JSONB DEFAULT '["*"]', -- Array of event types to subscribe to
    is_active BOOLEAN DEFAULT TRUE,
    headers JSONB DEFAULT '{}',
    retry_count INTEGER DEFAULT 3,
    timeout_ms INTEGER DEFAULT 30000,
    last_triggered_at TIMESTAMPTZ,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Webhook delivery logs table
CREATE TABLE IF NOT EXISTS webhook_delivery_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_config_id UUID NOT NULL REFERENCES webhook_configs(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    response_status INTEGER,
    response_body TEXT,
    response_headers JSONB DEFAULT '{}',
    error_message TEXT,
    attempt_number INTEGER NOT NULL DEFAULT 1,
    duration_ms INTEGER,
    is_success BOOLEAN DEFAULT FALSE,
    triggered_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Indexes for webhook_configs
CREATE INDEX IF NOT EXISTS idx_webhook_configs_tenant ON webhook_configs(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_webhook_configs_event ON webhook_configs(tenant_id, event_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_webhook_configs_active ON webhook_configs(tenant_id, is_active, deleted_at);

-- Indexes for webhook_delivery_logs
CREATE INDEX IF NOT EXISTS idx_webhook_logs_config ON webhook_delivery_logs(webhook_config_id);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_event ON webhook_delivery_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_success ON webhook_delivery_logs(webhook_config_id, is_success);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_triggered ON webhook_delivery_logs(triggered_at DESC);

-- +goose StatementEnd
