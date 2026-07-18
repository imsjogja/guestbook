-- +goose Up
-- +goose StatementBegin

ALTER TABLE invitations
    ADD COLUMN IF NOT EXISTS failed_reason TEXT;

-- +goose StatementEnd
