-- +goose Up
-- +goose StatementBegin

ALTER TABLE communication_messages
    ADD COLUMN IF NOT EXISTS provider_http_status SMALLINT;

-- +goose StatementEnd
