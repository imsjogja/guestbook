-- +goose Up
-- +goose StatementBegin
-- Migration 004: Create roles table
-- Defines roles with associated permissions. System roles have no tenant_id
-- and are available globally. Tenant-specific roles can be created per tenant.

CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    permissions TEXT[] DEFAULT '{}',
    is_system BOOLEAN DEFAULT FALSE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- Index for system roles lookup
CREATE INDEX IF NOT EXISTS idx_roles_system ON roles(is_system) WHERE is_system = TRUE;

-- Index for tenant roles lookup
CREATE INDEX IF NOT EXISTS idx_roles_tenant ON roles(tenant_id) WHERE tenant_id IS NOT NULL;

-- Insert default system roles
INSERT INTO roles (name, display_name, description, permissions, is_system) VALUES
    ('tenant_owner', 'Tenant Owner', 'Full access to tenant resources', ARRAY[
        'tenant.*', 'event.*', 'guest.*', 'rsvp.*', 'checkin.*',
        'invitation.*', 'seating.*', 'report.*', 'user.*', 'setting.*'
    ], TRUE),
    ('event_manager', 'Event Manager', 'Manage events and all related resources', ARRAY[
        'event.*', 'guest.*', 'rsvp.*', 'checkin.*',
        'invitation.*', 'seating.*', 'report.*', 'user.view'
    ], TRUE),
    ('rsvp_officer', 'RSVP Officer', 'Manage RSVPs and guest responses', ARRAY[
        'rsvp.*', 'guest.view', 'guest.update_rsvp', 'report.rsvp'
    ], TRUE),
    ('registration_officer', 'Registration Officer', 'Handle check-ins and walk-in registration', ARRAY[
        'checkin.*', 'guest.view', 'guest.create_walkin', 'report.checkin'
    ], TRUE),
    ('usher', 'Usher', 'View guest lists and assist with seating', ARRAY[
        'guest.view', 'seating.view', 'checkin.create'
    ], TRUE),
    ('gift_officer', 'Gift Officer', 'Record and track gift envelopes', ARRAY[
        'gift.*', 'guest.view', 'report.gift'
    ], TRUE),
    ('viewer', 'Viewer', 'Read-only access to guest lists and reports', ARRAY[
        'guest.view', 'report.view', 'event.view', 'rsvp.view', 'checkin.view'
    ], TRUE)
ON CONFLICT DO NOTHING;

-- Trigger to automatically update updated_at timestamp
DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;
CREATE TRIGGER update_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd
