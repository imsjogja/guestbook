-- Migration: Drop guest_tags and guest_tag_assignments tables
-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_guest_tag_assignments_tag;
DROP TABLE IF EXISTS guest_tag_assignments;

DROP INDEX IF EXISTS idx_guest_tags_tenant;
DROP TABLE IF EXISTS guest_tags;

-- +goose StatementEnd
