-- +goose Down
-- +goose StatementBegin

ALTER TABLE invitations
    DROP COLUMN IF EXISTS failed_reason;

-- +goose StatementEnd
