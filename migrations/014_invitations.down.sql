-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_credential_usage_created;
DROP INDEX IF EXISTS idx_credential_usage_type;
DROP INDEX IF EXISTS idx_credential_usage_event;
DROP INDEX IF EXISTS idx_credential_usage_invitation;
DROP TABLE IF EXISTS credential_usage_log;

DROP INDEX IF EXISTS idx_invitations_unique_active;
DROP INDEX IF EXISTS idx_invitations_event_status;
DROP INDEX IF EXISTS idx_invitations_status;
DROP INDEX IF EXISTS idx_invitations_token_hash;
DROP INDEX IF EXISTS idx_invitations_guest;
DROP INDEX IF EXISTS idx_invitations_event;
DROP INDEX IF EXISTS idx_invitations_tenant;
DROP TABLE IF EXISTS invitations;

-- +goose StatementEnd
