-- +goose Down
-- +goose StatementBegin

ALTER TABLE communication_messages
    DROP COLUMN IF EXISTS provider_http_status;

-- +goose StatementEnd
