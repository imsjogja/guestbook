-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS email_verification_tokens;
-- +goose StatementEnd
