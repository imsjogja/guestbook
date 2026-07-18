-- +goose Up
-- +goose StatementBegin
-- Ensure every existing tenant has the standard RSVP reminder templates.
-- The NOT EXISTS guards make this generator safe to run more than once.

INSERT INTO communication_templates (
    id, tenant_id, name, channel, type, subject, body, variables,
    is_active, is_system, description, language, created_at, updated_at
)
SELECT
    gen_random_uuid(),
    t.id,
    'Pengingat RSVP WhatsApp',
    'whatsapp',
    'rsvp_followup',
    NULL,
    'Halo {{guest_name}},

Kami menunggu konfirmasi kehadiran Anda untuk acara {{event_name}} pada {{event_date}} pukul {{event_time}}.

Mohon konfirmasi melalui tautan berikut:
{{rsvp_link}}

Terima kasih.',
    '["guest_name", "event_name", "event_date", "event_time", "rsvp_link"]'::jsonb,
    TRUE,
    TRUE,
    'Template pengingat konfirmasi RSVP melalui WhatsApp.',
    'id',
    NOW(),
    NOW()
FROM tenants t
WHERE t.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM communication_templates ct
      WHERE ct.tenant_id = t.id
        AND ct.name = 'Pengingat RSVP WhatsApp'
        AND ct.channel = 'whatsapp'
        AND ct.type = 'rsvp_followup'
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
    'Pengingat RSVP Email',
    'email',
    'rsvp_followup',
    'Pengingat Konfirmasi Kehadiran {{event_name}}',
    '<!doctype html><html><body><p>Halo {{guest_name}},</p><p>Kami menunggu konfirmasi kehadiran Anda untuk acara <strong>{{event_name}}</strong>.</p><p>Tanggal: {{event_date}}<br>Waktu: {{event_time}}</p><p><a href="{{rsvp_link}}">Konfirmasi kehadiran Anda di sini</a></p><p>Terima kasih.</p></body></html>',
    '["guest_name", "event_name", "event_date", "event_time", "rsvp_link"]'::jsonb,
    TRUE,
    TRUE,
    'Template pengingat konfirmasi RSVP melalui email.',
    'id',
    NOW(),
    NOW()
FROM tenants t
WHERE t.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM communication_templates ct
      WHERE ct.tenant_id = t.id
        AND ct.name = 'Pengingat RSVP Email'
        AND ct.channel = 'email'
        AND ct.type = 'rsvp_followup'
        AND ct.is_system = TRUE
        AND ct.deleted_at IS NULL
  );

-- Seed invitation and reminder defaults together for tenants created after this migration.
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
    ),
    (
        gen_random_uuid(),
        NEW.id,
        'Pengingat RSVP WhatsApp',
        'whatsapp',
        'rsvp_followup',
        NULL,
        'Halo {{guest_name}},

Kami menunggu konfirmasi kehadiran Anda untuk acara {{event_name}} pada {{event_date}} pukul {{event_time}}.

Mohon konfirmasi melalui tautan berikut:
{{rsvp_link}}

Terima kasih.',
        '["guest_name", "event_name", "event_date", "event_time", "rsvp_link"]'::jsonb,
        TRUE,
        TRUE,
        'Template pengingat konfirmasi RSVP melalui WhatsApp.',
        'id',
        NOW(),
        NOW()
    ),
    (
        gen_random_uuid(),
        NEW.id,
        'Pengingat RSVP Email',
        'email',
        'rsvp_followup',
        'Pengingat Konfirmasi Kehadiran {{event_name}}',
        '<!doctype html><html><body><p>Halo {{guest_name}},</p><p>Kami menunggu konfirmasi kehadiran Anda untuk acara <strong>{{event_name}}</strong>.</p><p>Tanggal: {{event_date}}<br>Waktu: {{event_time}}</p><p><a href="{{rsvp_link}}">Konfirmasi kehadiran Anda di sini</a></p><p>Terima kasih.</p></body></html>',
        '["guest_name", "event_name", "event_date", "event_time", "rsvp_link"]'::jsonb,
        TRUE,
        TRUE,
        'Template pengingat konfirmasi RSVP melalui email.',
        'id',
        NOW(),
        NOW()
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
