-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_event_locations_name;
DROP INDEX IF EXISTS idx_event_locations_tenant;
DROP TABLE IF EXISTS event_locations;
-- +goose StatementEnd
