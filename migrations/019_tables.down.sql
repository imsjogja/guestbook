-- Migration: Drop tables and seat_assignments tables
-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS seat_assignments CASCADE;
DROP TABLE IF EXISTS tables CASCADE;

-- +goose StatementEnd
