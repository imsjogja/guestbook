-- +goose Down
-- +goose StatementBegin
-- Migration 001: Rollback - Drop users table

DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
