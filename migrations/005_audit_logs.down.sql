-- +goose Down
-- +goose StatementBegin
-- Migration 005: Rollback - Drop audit_logs table

DROP TABLE IF EXISTS audit_logs;
-- +goose StatementEnd
