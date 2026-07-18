-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS event_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    address TEXT,
    city VARCHAR(100),
    maps_url VARCHAR(500),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    instructions TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_event_locations_tenant ON event_locations(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_event_locations_name ON event_locations(tenant_id, name);
-- +goose StatementEnd
