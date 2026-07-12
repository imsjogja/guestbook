-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_webhook_configs_tenant;
DROP INDEX IF EXISTS idx_webhook_configs_event;
DROP INDEX IF EXISTS idx_webhook_configs_active;
DROP INDEX IF EXISTS idx_webhook_logs_config;
DROP INDEX IF EXISTS idx_webhook_logs_event;
DROP INDEX IF EXISTS idx_webhook_logs_success;
DROP INDEX IF EXISTS idx_webhook_logs_triggered;

DROP TABLE IF EXISTS webhook_delivery_logs;
DROP TABLE IF EXISTS webhook_configs;

-- +goose StatementEnd
