-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS auth_email_tokens;
-- +goose StatementEnd
