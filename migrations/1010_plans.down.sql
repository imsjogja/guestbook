-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS plans;
-- +goose StatementEnd
