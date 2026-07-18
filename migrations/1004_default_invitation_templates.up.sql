-- +goose Up
-- +goose StatementBegin
-- Ensure every existing tenant has the standard invitation templates.
-- The NOT EXISTS guards make this generator safe to run more than once.

INSERT INTO communication_templates (
    id, tenant_id, name, channel, type, subject, body, variables,
    is_active, is_system, description, language, created_at, updated_at
)
SELECT
    gen_random_uuid(),
    t.id,
    'Undangan Standar WhatsApp',
    'whatsapp',
    'invitation',
    NULL,
    'Halo {{guest_name}},

Anda kami undang ke acara {{event_name}} pada {{event_date}} pukul {{event_time}}.

Konfirmasi kehadiran dan lihat detail undangan:
{{rsvp_link}}

Terima kasih.',
    '["guest_name", "event_name", "event_date", "event_time", "rsvp_link"]'::jsonb,
    TRUE,
    TRUE,
    'Template undangan standar melalui WhatsApp.',
    'id',
    NOW(),
    NOW()
FROM tenants t
WHERE t.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM communication_templates ct
      WHERE ct.tenant_id = t.id
        AND ct.name = 'Undangan Standar WhatsApp'
        AND ct.channel = 'whatsapp'
        AND ct.type = 'invitation'
        AND ct.is_system = TRUE
        AND ct.deleted_at IS NULL
  );

INSERT INTO communication_templates (
    id, tenant_id, name, channel, type, subject, body, variables,
    is_active, is_system, description, language, created_at, updated_at
)
SELECT
    gen_random_uuid(),
    t.id,
    'Undangan Standar Email',
    'email',
    'invitation',
    'Undangan {{event_name}} untuk {{guest_name}}',
    '<!doctype html><html><body><p>Halo {{guest_name}},</p><p>Dengan hormat, kami mengundang Anda ke acara <strong>{{event_name}}</strong>.</p><p>Tanggal: {{event_date}}<br>Waktu: {{event_time}}</p><p><a href="{{rsvp_link}}">Lihat undangan dan konfirmasi kehadiran</a></p><p>Terima kasih.</p></body></html>',
    '["guest_name", "event_name", "event_date", "event_time", "rsvp_link"]'::jsonb,
    TRUE,
    TRUE,
    'Template undangan standar melalui email.',
    'id',
    NOW(),
    NOW()
FROM tenants t
WHERE t.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM communication_templates ct
      WHERE ct.tenant_id = t.id
        AND ct.name = 'Undangan Standar Email'
        AND ct.channel = 'email'
        AND ct.type = 'invitation'
        AND ct.is_system = TRUE
        AND ct.deleted_at IS NULL
  );

-- Automatically provision the same defaults for every tenant created after this migration.
CREATE OR REPLACE FUNCTION guestflow_seed_default_communication_templates()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO communication_templates (
        id, tenant_id, name, channel, type, subject, body, variables,
        is_active, is_system, description, language, created_at, updated_at
    ) VALUES
    (
        gen_random_uuid(),
        NEW.id,
        'Undangan Standar WhatsApp',
        'whatsapp',
        'invitation',
        NULL,
        'Halo {{guest_name}},

Anda kami undang ke acara {{event_name}} pada {{event_date}} pukul {{event_time}}.

Konfirmasi kehadiran dan lihat detail undangan:
{{rsvp_link}}

Terima kasih.',
        '["guest_name", "event_name", "event_date", "event_time", "rsvp_link"]'::jsonb,
        TRUE,
        TRUE,
        'Template undangan standar melalui WhatsApp.',
        'id',
        NOW(),
        NOW()
    ),
    (
        gen_random_uuid(),
        NEW.id,
        'Undangan Standar Email',
        'email',
        'invitation',
        'Undangan {{event_name}} untuk {{guest_name}}',
        '<!doctype html><html><body><p>Halo {{guest_name}},</p><p>Dengan hormat, kami mengundang Anda ke acara <strong>{{event_name}}</strong>.</p><p>Tanggal: {{event_date}}<br>Waktu: {{event_time}}</p><p><a href="{{rsvp_link}}">Lihat undangan dan konfirmasi kehadiran</a></p><p>Terima kasih.</p></body></html>',
        '["guest_name", "event_name", "event_date", "event_time", "rsvp_link"]'::jsonb,
        TRUE,
        TRUE,
        'Template undangan standar melalui email.',
        'id',
        NOW(),
        NOW()
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS guestflow_seed_default_communication_templates_trigger ON tenants;
CREATE TRIGGER guestflow_seed_default_communication_templates_trigger
AFTER INSERT ON tenants
FOR EACH ROW
EXECUTE FUNCTION guestflow_seed_default_communication_templates();
-- +goose StatementEnd
