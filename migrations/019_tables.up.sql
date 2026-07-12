-- Migration: Create tables and seat_assignments tables
-- +goose Up
-- +goose StatementBegin

-- Tables (seating tables)
CREATE TABLE IF NOT EXISTS tables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    zone_id UUID REFERENCES venue_zones(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    capacity INTEGER NOT NULL CHECK (capacity > 0 AND capacity <= 999),
    shape VARCHAR(20) DEFAULT 'round' CHECK (shape IN ('round', 'rectangular', 'square', 'oval', 'u_shape')),
    position_x DECIMAL(8, 2),
    position_y DECIMAL(8, 2),
    is_locked BOOLEAN DEFAULT FALSE,
    accessibility BOOLEAN DEFAULT FALSE,
    vip_only BOOLEAN DEFAULT FALSE,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Seat assignments (links guests to tables)
CREATE TABLE IF NOT EXISTS seat_assignments (
    table_id UUID NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
    guest_id UUID NOT NULL REFERENCES guests(id) ON DELETE CASCADE,
    seat_number INTEGER,
    assigned_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (table_id, guest_id)
);

-- Indexes for tables
CREATE INDEX IF NOT EXISTS idx_tables_tenant_event ON tables(tenant_id, event_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_tables_zone ON tables(tenant_id, event_id, zone_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_tables_name ON tables(tenant_id, event_id, name);
CREATE INDEX IF NOT EXISTS idx_tables_accessibility ON tables(tenant_id, event_id, accessibility) WHERE accessibility = TRUE AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tables_vip ON tables(tenant_id, event_id, vip_only) WHERE vip_only = TRUE AND deleted_at IS NULL;

-- Indexes for seat_assignments
CREATE INDEX IF NOT EXISTS idx_seat_assignments_table ON seat_assignments(table_id);
CREATE INDEX IF NOT EXISTS idx_seat_assignments_guest ON seat_assignments(guest_id);

-- +goose StatementEnd
