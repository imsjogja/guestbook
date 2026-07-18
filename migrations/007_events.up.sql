-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    cover_url VARCHAR(500),
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ,
    rsvp_deadline TIMESTAMPTZ,
    capacity INTEGER,
    target_invites INTEGER,
    target_attendance INTEGER,
    primary_location_id UUID,
    dress_code VARCHAR(100),
    privacy_notice TEXT,
    guest_policy TEXT,
    settings JSONB DEFAULT '{}',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_events_tenant ON events(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_events_status ON events(status);
CREATE INDEX IF NOT EXISTS idx_events_dates ON events(start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_events_tenant_status ON events(tenant_id, status, deleted_at);
-- +goose StatementEnd
