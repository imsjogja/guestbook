-- Migration: Drop households and household_members tables
-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_household_members_primary;
DROP INDEX IF EXISTS idx_household_members_guest;
DROP TABLE IF EXISTS household_members;

DROP INDEX IF EXISTS idx_households_name;
DROP INDEX IF EXISTS idx_households_tenant;
DROP TABLE IF EXISTS households;

-- +goose StatementEnd
