-- Migration: Create venue_zones table
-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS venue_zones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_venue_zones_tenant_event ON venue_zones(tenant_id, event_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_venue_zones_sort ON venue_zones(tenant_id, event_id, sort_order);

-- +goose StatementEnd
