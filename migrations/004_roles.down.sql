-- +goose Down
-- +goose StatementBegin
-- Migration 004: Rollback - Drop roles table

DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;
DROP TABLE IF EXISTS roles;
-- +goose StatementEnd
