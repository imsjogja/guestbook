-- Migration: Drop checkins, checkin_gates, and checkin_devices tables
-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS checkins CASCADE;
DROP TABLE IF EXISTS checkin_devices CASCADE;
DROP TABLE IF EXISTS checkin_gates CASCADE;

-- +goose StatementEnd
