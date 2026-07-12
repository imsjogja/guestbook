-- Migration: Create communication tables (templates, campaigns, messages)
-- +goose Up
-- +goose StatementBegin

-- Communication templates table
CREATE TABLE IF NOT EXISTS communication_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('whatsapp', 'email', 'sms')),
    type VARCHAR(50) NOT NULL,
    subject VARCHAR(500),
    body TEXT NOT NULL,
    variables JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT TRUE,
    is_system BOOLEAN DEFAULT FALSE,
    description TEXT,
    language VARCHAR(10) DEFAULT 'id',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Communication campaigns table
CREATE TABLE IF NOT EXISTS communication_campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    template_id UUID NOT NULL REFERENCES communication_templates(id) ON DELETE CASCADE,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('whatsapp', 'email', 'sms')),
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'scheduled', 'sending', 'completed', 'cancelled')),
    recipient_filter JSONB DEFAULT '{}',
    scheduled_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    total_recipients INTEGER NOT NULL DEFAULT 0,
    sent_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Communication messages table
CREATE TABLE IF NOT EXISTS communication_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    campaign_id UUID REFERENCES communication_campaigns(id) ON DELETE SET NULL,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    guest_id UUID NOT NULL REFERENCES guests(id) ON DELETE CASCADE,
    invitation_id UUID REFERENCES invitations(id) ON DELETE SET NULL,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('whatsapp', 'email', 'sms')),
    type VARCHAR(50) NOT NULL,
    subject VARCHAR(500),
    body TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'queued' CHECK (status IN ('draft', 'queued', 'sent', 'delivered', 'read', 'failed', 'cancelled')),
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    error_message TEXT,
    external_id VARCHAR(255),
    cost DECIMAL(10, 4),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for communication_templates
CREATE INDEX IF NOT EXISTS idx_comm_templates_tenant ON communication_templates(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_comm_templates_channel ON communication_templates(tenant_id, channel, deleted_at);
CREATE INDEX IF NOT EXISTS idx_comm_templates_type ON communication_templates(tenant_id, type, deleted_at);
CREATE INDEX IF NOT EXISTS idx_comm_templates_active ON communication_templates(tenant_id, is_active, deleted_at);
CREATE INDEX IF NOT EXISTS idx_comm_templates_channel_type ON communication_templates(tenant_id, channel, type, is_active, deleted_at);

-- Indexes for communication_campaigns
CREATE INDEX IF NOT EXISTS idx_comm_campaigns_tenant ON communication_campaigns(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_comm_campaigns_event ON communication_campaigns(tenant_id, event_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_comm_campaigns_status ON communication_campaigns(tenant_id, event_id, status, deleted_at);
CREATE INDEX IF NOT EXISTS idx_comm_campaigns_template ON communication_campaigns(tenant_id, template_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_comm_campaigns_scheduled ON communication_campaigns(tenant_id, event_id, status, scheduled_at) WHERE status = 'scheduled';

-- Indexes for communication_messages
CREATE INDEX IF NOT EXISTS idx_comm_messages_tenant ON communication_messages(tenant_id, event_id);
CREATE INDEX IF NOT EXISTS idx_comm_messages_campaign ON communication_messages(campaign_id);
CREATE INDEX IF NOT EXISTS idx_comm_messages_guest ON communication_messages(tenant_id, event_id, guest_id);
CREATE INDEX IF NOT EXISTS idx_comm_messages_status ON communication_messages(tenant_id, event_id, status);
CREATE INDEX IF NOT EXISTS idx_comm_messages_external ON communication_messages(external_id);
CREATE INDEX IF NOT EXISTS idx_comm_messages_created ON communication_messages(tenant_id, event_id, created_at DESC);

-- +goose StatementEnd
