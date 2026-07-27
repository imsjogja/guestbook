-- +goose Up
-- +goose StatementBegin
-- Remove retired provider settings and tenant-managed WhatsApp credentials.
-- GOWA is now the only WhatsApp transport; tenant isolation is represented
-- only by enabled and device_id.
UPDATE tenants
SET settings = jsonb_set(
        COALESCE(settings, '{}'::jsonb),
        '{integrations,whatsapp}',
        (settings #> '{integrations,whatsapp}')
            - 'provider'
            - 'api_url'
            - 'username'
            - 'password'
            - 'account_token'
            - 'sender_token'
            - 'phone_number_id'
            - 'access_token'
            - 'webhook_verify_token',
        true
    ),
    updated_at = NOW()
WHERE settings #> '{integrations,whatsapp}' IS NOT NULL;
-- +goose StatementEnd
