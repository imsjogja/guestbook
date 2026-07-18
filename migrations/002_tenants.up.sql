-- +goose Up
-- +goose StatementBegin
-- Migration 002: Create tenants table
-- Tenants represent organizations (e.g., couples, event planners) that
-- contain events, guests, and other resources.

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    logo_url VARCHAR(500),
    primary_color VARCHAR(7) DEFAULT '#0d6efd',
    settings JSONB DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'trial', -- trial, active, suspended, cancelled
    trial_ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Index for slug lookups (subdomain resolution)
CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug) WHERE deleted_at IS NULL;

-- Index for status filtering
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;

-- Trigger to automatically update updated_at timestamp
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
CREATE TRIGGER update_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd
