-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_events_self_checkin_token;
ALTER TABLE events DROP COLUMN IF EXISTS self_checkin_token;

-- +goose StatementEnd
