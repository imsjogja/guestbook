-- Migration: Create the event guest roster and link existing operational records.
-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS event_guests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    guest_id UUID NOT NULL REFERENCES guests(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'cancelled')),
    source VARCHAR(20) NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'import', 'invitation', 'rsvp', 'checkin', 'seating', 'communication', 'copy_event', 'walk_in')),
    max_pax INTEGER NOT NULL DEFAULT 1 CHECK (max_pax > 0),
    adults INTEGER NOT NULL DEFAULT 1 CHECK (adults >= 0),
    children INTEGER NOT NULL DEFAULT 0 CHECK (children >= 0),
    plus_one_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    notes TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_event_guests_unique_active
    ON event_guests(event_id, guest_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_event_guests_tenant_event
    ON event_guests(tenant_id, event_id, status, deleted_at);
CREATE INDEX IF NOT EXISTS idx_event_guests_guest
    ON event_guests(tenant_id, guest_id, deleted_at);

-- Backfill the roster from existing event-scoped records. The event creator is
-- used as the actor because older records do not consistently retain one.
INSERT INTO event_guests (
    tenant_id, event_id, guest_id, source, created_by, created_at, updated_at
)
SELECT DISTINCT ON (c.event_id, c.guest_id)
    c.tenant_id, c.event_id, c.guest_id, c.source, e.created_by, NOW(), NOW()
FROM (
    SELECT tenant_id, event_id, guest_id, 'invitation' AS source
    FROM invitations WHERE deleted_at IS NULL
    UNION ALL
    SELECT tenant_id, event_id, guest_id, 'rsvp' AS source
    FROM rsvp_responses
    UNION ALL
    SELECT tenant_id, event_id, guest_id, 'checkin' AS source
    FROM checkins WHERE deleted_at IS NULL
    UNION ALL
    SELECT tenant_id, event_id, guest_id, 'communication' AS source
    FROM communication_messages
    UNION ALL
    SELECT t.tenant_id, t.event_id, sa.guest_id, 'seating' AS source
    FROM seat_assignments sa
    JOIN tables t ON t.id = sa.table_id
    WHERE t.deleted_at IS NULL
) c
JOIN events e ON e.id = c.event_id AND e.tenant_id = c.tenant_id AND e.deleted_at IS NULL
JOIN guests g ON g.id = c.guest_id AND g.tenant_id = c.tenant_id AND g.deleted_at IS NULL
WHERE NOT EXISTS (
    SELECT 1 FROM event_guests eg
    WHERE eg.event_id = c.event_id AND eg.guest_id = c.guest_id AND eg.deleted_at IS NULL
)
ORDER BY c.event_id, c.guest_id,
    CASE c.source
        WHEN 'invitation' THEN 1
        WHEN 'rsvp' THEN 2
        WHEN 'checkin' THEN 3
        WHEN 'seating' THEN 4
        ELSE 5
    END;

-- Nullable links let the application adopt the roster without invalidating
-- historical rows. Subsequent migrations can make these links mandatory.
ALTER TABLE invitations ADD COLUMN IF NOT EXISTS event_guest_id UUID REFERENCES event_guests(id) ON DELETE SET NULL;
ALTER TABLE rsvp_responses ADD COLUMN IF NOT EXISTS event_guest_id UUID REFERENCES event_guests(id) ON DELETE SET NULL;
ALTER TABLE checkins ADD COLUMN IF NOT EXISTS event_guest_id UUID REFERENCES event_guests(id) ON DELETE SET NULL;
ALTER TABLE communication_messages ADD COLUMN IF NOT EXISTS event_guest_id UUID REFERENCES event_guests(id) ON DELETE SET NULL;
ALTER TABLE seat_assignments ADD COLUMN IF NOT EXISTS event_guest_id UUID REFERENCES event_guests(id) ON DELETE SET NULL;

UPDATE invitations i
SET event_guest_id = eg.id
FROM event_guests eg
WHERE i.event_guest_id IS NULL AND eg.event_id = i.event_id AND eg.guest_id = i.guest_id AND eg.deleted_at IS NULL;

UPDATE rsvp_responses r
SET event_guest_id = eg.id
FROM event_guests eg
WHERE r.event_guest_id IS NULL AND eg.event_id = r.event_id AND eg.guest_id = r.guest_id AND eg.deleted_at IS NULL;

UPDATE checkins c
SET event_guest_id = eg.id
FROM event_guests eg
WHERE c.event_guest_id IS NULL AND eg.event_id = c.event_id AND eg.guest_id = c.guest_id AND eg.deleted_at IS NULL;

UPDATE communication_messages m
SET event_guest_id = eg.id
FROM event_guests eg
WHERE m.event_guest_id IS NULL AND eg.event_id = m.event_id AND eg.guest_id = m.guest_id AND eg.deleted_at IS NULL;

UPDATE seat_assignments sa
SET event_guest_id = eg.id
FROM tables t, event_guests eg
WHERE sa.event_guest_id IS NULL
  AND t.id = sa.table_id
  AND eg.event_id = t.event_id
  AND eg.guest_id = sa.guest_id
  AND eg.deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_event_guests_event_status ON event_guests(event_id, status, deleted_at);
CREATE INDEX IF NOT EXISTS idx_invitations_event_guest ON invitations(event_guest_id);
CREATE INDEX IF NOT EXISTS idx_rsvp_event_guest ON rsvp_responses(event_guest_id);
CREATE INDEX IF NOT EXISTS idx_checkins_event_guest ON checkins(event_guest_id);
CREATE INDEX IF NOT EXISTS idx_comm_messages_event_guest ON communication_messages(event_guest_id);
CREATE INDEX IF NOT EXISTS idx_seat_assignments_event_guest ON seat_assignments(event_guest_id);

-- +goose StatementEnd
