-- +goose Up
-- +goose StatementBegin

-- Keep legacy duplicate settings usable while giving each tenant its own
-- durable GOWA session identifier. The first tenant keeps the old value;
-- later duplicates are moved to a deterministic tenant-scoped ID and must be
-- paired again by that tenant owner.
WITH ranked AS (
    SELECT
        id,
        settings #>> '{integrations,whatsapp,device_id}' AS device_id,
        ROW_NUMBER() OVER (
            PARTITION BY settings #>> '{integrations,whatsapp,device_id}'
            ORDER BY created_at, id
        ) AS row_number
    FROM tenants
    WHERE deleted_at IS NULL
      AND NULLIF(settings #>> '{integrations,whatsapp,device_id}', '') IS NOT NULL
), duplicates AS (
    SELECT id
    FROM ranked
    WHERE row_number > 1
)
UPDATE tenants AS t
SET settings = jsonb_set(
        COALESCE(t.settings, '{}'::jsonb),
        '{integrations,whatsapp,device_id}',
        to_jsonb('guestflow-' || replace(t.id::text, '-', '')),
        true
    ),
    updated_at = NOW()
FROM duplicates AS d
WHERE t.id = d.id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_whatsapp_device_id
    ON tenants ((settings #>> '{integrations,whatsapp,device_id}'))
    WHERE deleted_at IS NULL
      AND NULLIF(settings #>> '{integrations,whatsapp,device_id}', '') IS NOT NULL;

-- +goose StatementEnd
