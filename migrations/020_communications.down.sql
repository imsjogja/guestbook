-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_comm_templates_tenant;
DROP INDEX IF EXISTS idx_comm_templates_channel;
DROP INDEX IF EXISTS idx_comm_templates_type;
DROP INDEX IF EXISTS idx_comm_templates_active;
DROP INDEX IF EXISTS idx_comm_templates_channel_type;
DROP INDEX IF EXISTS idx_comm_campaigns_tenant;
DROP INDEX IF EXISTS idx_comm_campaigns_event;
DROP INDEX IF EXISTS idx_comm_campaigns_status;
DROP INDEX IF EXISTS idx_comm_campaigns_template;
DROP INDEX IF EXISTS idx_comm_campaigns_scheduled;
DROP INDEX IF EXISTS idx_comm_messages_tenant;
DROP INDEX IF EXISTS idx_comm_messages_campaign;
DROP INDEX IF EXISTS idx_comm_messages_guest;
DROP INDEX IF EXISTS idx_comm_messages_status;
DROP INDEX IF EXISTS idx_comm_messages_external;
DROP INDEX IF EXISTS idx_comm_messages_created;

DROP TABLE IF EXISTS communication_messages;
DROP TABLE IF EXISTS communication_campaigns;
DROP TABLE IF EXISTS communication_templates;

-- +goose StatementEnd
