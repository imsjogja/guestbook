-- Migration: Create households and household_members tables
-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS households (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    address TEXT,
    city VARCHAR(100),
    max_pax INTEGER,
    notes TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_households_tenant ON households(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_households_name ON households(tenant_id, name);

CREATE TABLE IF NOT EXISTS household_members (
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    guest_id UUID NOT NULL REFERENCES guests(id) ON DELETE CASCADE,
    is_primary BOOLEAN DEFAULT FALSE,
    role VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (household_id, guest_id)
);

CREATE INDEX IF NOT EXISTS idx_household_members_guest ON household_members(guest_id);
CREATE INDEX IF NOT EXISTS idx_household_members_primary ON household_members(household_id, is_primary) WHERE is_primary = TRUE;

-- +goose StatementEnd
