-- +goose Up
-- +goose StatementBegin

-- invitations table: stores guest invitations with opaque token hashes
CREATE TABLE IF NOT EXISTS invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    guest_id UUID NOT NULL REFERENCES guests(id) ON DELETE CASCADE,
    token VARCHAR(255), -- Raw token (only set temporarily during creation, cleared after)
    token_hash VARCHAR(64) NOT NULL, -- SHA-256 hash of the token for lookup
    url VARCHAR(500) NOT NULL, -- Public invitation URL
    max_pax INTEGER NOT NULL DEFAULT 1,
    adults INTEGER NOT NULL DEFAULT 0,
    children INTEGER NOT NULL DEFAULT 0,
    plus_one_allowed BOOLEAN NOT NULL DEFAULT false,
    plus_one_required BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    sent_at TIMESTAMPTZ,
    opened_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id),
    revoke_reason TEXT,
    expires_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_invitations_tenant ON invitations(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_invitations_event ON invitations(event_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_invitations_guest ON invitations(guest_id);
CREATE INDEX IF NOT EXISTS idx_invitations_token_hash ON invitations(token_hash);
CREATE INDEX IF NOT EXISTS idx_invitations_status ON invitations(status);
CREATE INDEX IF NOT EXISTS idx_invitations_event_status ON invitations(event_id, status, deleted_at);

-- Unique constraint: one active invitation per guest per event
CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_unique_active ON invitations(event_id, guest_id) WHERE deleted_at IS NULL;

-- credential_usage_log table: tracks scans and usage of invitation credentials
CREATE TABLE IF NOT EXISTS credential_usage_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invitation_id UUID NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    guest_id UUID NOT NULL REFERENCES guests(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL, -- checkin, rsvp, opened
    device_id VARCHAR(255),
    gate_id UUID,
    officer_id UUID REFERENCES users(id),
    ip_address INET,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_credential_usage_invitation ON credential_usage_log(invitation_id);
CREATE INDEX IF NOT EXISTS idx_credential_usage_event ON credential_usage_log(event_id);
CREATE INDEX IF NOT EXISTS idx_credential_usage_type ON credential_usage_log(type);
CREATE INDEX IF NOT EXISTS idx_credential_usage_created ON credential_usage_log(created_at);

-- +goose StatementEnd
