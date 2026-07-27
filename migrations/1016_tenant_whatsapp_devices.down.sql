-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_tenants_whatsapp_device_id;

-- +goose StatementEnd
