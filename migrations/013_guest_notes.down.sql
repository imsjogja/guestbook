-- Migration: Drop guest_notes table
-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_guest_notes_pinned;
DROP INDEX IF EXISTS idx_guest_notes_guest;
DROP TABLE IF EXISTS guest_notes;

-- +goose StatementEnd
