-- +goose Up
-- +goose StatementBegin

ALTER TABLE events
    ADD COLUMN IF NOT EXISTS self_checkin_token VARCHAR(64)
    DEFAULT replace(gen_random_uuid()::text, '-', '');

UPDATE events
SET self_checkin_token = replace(gen_random_uuid()::text, '-', '')
WHERE self_checkin_token IS NULL;

ALTER TABLE events
    ALTER COLUMN self_checkin_token SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_events_self_checkin_token
    ON events(self_checkin_token) WHERE deleted_at IS NULL;

-- +goose StatementEnd
