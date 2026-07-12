-- Migration 003: Create tenant_users table
-- Links users to tenants with a specific role. A user can belong to
-- multiple tenants with different roles in each.

CREATE TABLE IF NOT EXISTS tenant_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'event_manager',
    -- tenant_owner, event_manager, rsvp_officer, registration_officer, usher, gift_officer, viewer
    invited_by UUID REFERENCES users(id),
    invited_at TIMESTAMPTZ,
    joined_at TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'active', -- active, pending, inactive
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, user_id)
);

-- Index for looking up a user's tenants
CREATE INDEX IF NOT EXISTS idx_tenant_users_user_id ON tenant_users(user_id);

-- Index for looking up a tenant's members
CREATE INDEX IF NOT EXISTS idx_tenant_users_tenant_id ON tenant_users(tenant_id);

-- Index for role-based queries
CREATE INDEX IF NOT EXISTS idx_tenant_users_role ON tenant_users(tenant_id, role) WHERE status = 'active';

-- Trigger to automatically update updated_at timestamp
CREATE TRIGGER update_tenant_users_updated_at
    BEFORE UPDATE ON tenant_users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
