-- Migration: Drop venue_zones table
-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS venue_zones CASCADE;

-- +goose StatementEnd
