-- Backfill the demo event roster so the seeded workspace is usable end to end.
-- +goose Up
-- +goose StatementBegin

INSERT INTO event_guests (
    tenant_id, event_id, guest_id, source, created_by, created_at, updated_at
)
SELECT
    e.tenant_id,
    e.id,
    g.id,
    'manual',
    e.created_by,
    NOW(),
    NOW()
FROM events e
JOIN guests g ON g.tenant_id = e.tenant_id AND g.deleted_at IS NULL
WHERE e.id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13'
  AND e.tenant_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'
  AND NOT EXISTS (
      SELECT 1
      FROM event_guests eg
      WHERE eg.event_id = e.id
        AND eg.guest_id = g.id
        AND eg.deleted_at IS NULL
  );

-- +goose StatementEnd
