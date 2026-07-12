-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_rsvp_unique_invitation;
DROP INDEX IF EXISTS idx_rsvp_responded_at;
DROP INDEX IF EXISTS idx_rsvp_event_status;
DROP INDEX IF EXISTS idx_rsvp_status;
DROP INDEX IF EXISTS idx_rsvp_guest;
DROP INDEX IF EXISTS idx_rsvp_invitation;
DROP INDEX IF EXISTS idx_rsvp_event;
DROP INDEX IF EXISTS idx_rsvp_tenant;
DROP TABLE IF EXISTS rsvp_responses;

-- +goose StatementEnd
