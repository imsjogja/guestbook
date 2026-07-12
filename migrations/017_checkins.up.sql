-- Migration: Create checkins, checkin_gates, and checkin_devices tables
-- +goose Up
-- +goose StatementBegin

-- Check-ins table
CREATE TABLE IF NOT EXISTS checkins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    session_id UUID REFERENCES event_sessions(id) ON DELETE SET NULL,
    guest_id UUID NOT NULL REFERENCES guests(id) ON DELETE CASCADE,
    invitation_id UUID,
    credential_id UUID,
    method VARCHAR(20) NOT NULL CHECK (method IN ('qr_scan', 'manual_search', 'walk_in', 'kiosk')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('success', 'duplicate', 'invalid', 'revoked', 'wrong_event', 'expired')),
    device_id VARCHAR(100),
    gate_id UUID,
    officer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actual_pax INTEGER NOT NULL DEFAULT 1,
    adults INTEGER NOT NULL DEFAULT 1,
    children INTEGER NOT NULL DEFAULT 0,
    override_reason TEXT,
    approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ip_address INET,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    notes TEXT,
    offline_synced BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Check-in gates table
CREATE TABLE IF NOT EXISTS checkin_gates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL,
    location VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, event_id, code)
);

-- Check-in devices table
CREATE TABLE IF NOT EXISTS checkin_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    device_code VARCHAR(100) NOT NULL,
    device_type VARCHAR(20) NOT NULL CHECK (device_type IN ('tablet', 'phone', 'kiosk')),
    officer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    last_sync_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, event_id, device_code)
);

-- Indexes for checkins
CREATE INDEX IF NOT EXISTS idx_checkins_tenant_event ON checkins(tenant_id, event_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_checkins_guest ON checkins(tenant_id, event_id, guest_id, status, deleted_at);
CREATE INDEX IF NOT EXISTS idx_checkins_method ON checkins(tenant_id, event_id, method, deleted_at);
CREATE INDEX IF NOT EXISTS idx_checkins_status ON checkins(tenant_id, event_id, status, deleted_at);
CREATE INDEX IF NOT EXISTS idx_checkins_gate ON checkins(tenant_id, event_id, gate_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_checkins_created ON checkins(tenant_id, event_id, created_at DESC, deleted_at);
CREATE INDEX IF NOT EXISTS idx_checkins_duplicate ON checkins(tenant_id, event_id, guest_id) WHERE status = 'success' AND deleted_at IS NULL;

-- Indexes for checkin_gates
CREATE INDEX IF NOT EXISTS idx_checkin_gates_tenant_event ON checkin_gates(tenant_id, event_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_checkin_gates_active ON checkin_gates(tenant_id, event_id, is_active);

-- Indexes for checkin_devices
CREATE INDEX IF NOT EXISTS idx_checkin_devices_tenant_event ON checkin_devices(tenant_id, event_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_checkin_devices_active ON checkin_devices(tenant_id, event_id, is_active);

-- +goose StatementEnd
