-- +goose Down
-- +goose StatementBegin
-- Remove the system RSVP reminder templates and restore the invitation-only
-- seed trigger from migration 1004.

DELETE FROM communication_templates
WHERE is_system = TRUE
  AND type = 'rsvp_followup'
  AND name IN ('Pengingat RSVP WhatsApp', 'Pengingat RSVP Email');

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
-- +goose StatementEnd
