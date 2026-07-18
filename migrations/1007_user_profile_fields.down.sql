-- +goose Down
-- +goose StatementBegin

ALTER TABLE users DROP COLUMN IF EXISTS position;
ALTER TABLE users DROP COLUMN IF EXISTS bio;

-- +goose StatementEnd
