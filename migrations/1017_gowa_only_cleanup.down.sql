-- The removed provider credentials are intentionally not restored.
-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
