-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_event_sessions_sort;
DROP INDEX IF EXISTS idx_event_sessions_time;
DROP INDEX IF EXISTS idx_event_sessions_event;
DROP TABLE IF EXISTS event_sessions;
-- +goose StatementEnd
