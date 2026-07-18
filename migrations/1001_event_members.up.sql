-- +goose Up
-- +goose StatementBegin
-- Event-scoped staff access. Tenant owners and event managers inherit access
-- from tenant_users and do not need rows in this table.

CREATE TABLE IF NOT EXISTS event_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('rsvp_officer', 'registration_officer', 'usher', 'gift_officer', 'viewer')),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    invited_by UUID REFERENCES users(id),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(event_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_event_members_tenant_event
    ON event_members(tenant_id, event_id, status);
CREATE INDEX IF NOT EXISTS idx_event_members_user
    ON event_members(tenant_id, user_id, status);

DROP TRIGGER IF EXISTS update_event_members_updated_at ON event_members;
CREATE TRIGGER update_event_members_updated_at
    BEFORE UPDATE ON event_members
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Preserve access for existing event-scoped tenant members on events that
-- already exist. New events require an explicit assignment.
INSERT INTO event_members (tenant_id, event_id, user_id, role, invited_by, assigned_at, created_at, updated_at)
SELECT e.tenant_id, e.id, tu.user_id, tu.role, tu.invited_by, NOW(), NOW(), NOW()
FROM events e
JOIN tenant_users tu ON tu.tenant_id = e.tenant_id
WHERE e.deleted_at IS NULL
  AND tu.status = 'active'
  AND tu.role IN ('rsvp_officer', 'registration_officer', 'usher', 'gift_officer', 'viewer')
ON CONFLICT (event_id, user_id) DO NOTHING;

-- +goose StatementEnd
