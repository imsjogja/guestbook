-- Migration: Drop guests table
-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_guests_email;
DROP INDEX IF EXISTS idx_guests_phone;
DROP INDEX IF EXISTS idx_guests_active;
DROP INDEX IF EXISTS idx_guests_segment;
DROP INDEX IF EXISTS idx_guests_type;
DROP INDEX IF EXISTS idx_guests_search;
DROP INDEX IF EXISTS idx_guests_tenant;

DROP TABLE IF EXISTS guests;

-- +goose StatementEnd
