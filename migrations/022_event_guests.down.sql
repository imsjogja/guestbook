-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_seat_assignments_event_guest;
DROP INDEX IF EXISTS idx_comm_messages_event_guest;
DROP INDEX IF EXISTS idx_checkins_event_guest;
DROP INDEX IF EXISTS idx_rsvp_event_guest;
DROP INDEX IF EXISTS idx_invitations_event_guest;
DROP INDEX IF EXISTS idx_event_guests_event_status;

ALTER TABLE seat_assignments DROP COLUMN IF EXISTS event_guest_id;
ALTER TABLE communication_messages DROP COLUMN IF EXISTS event_guest_id;
ALTER TABLE checkins DROP COLUMN IF EXISTS event_guest_id;
ALTER TABLE rsvp_responses DROP COLUMN IF EXISTS event_guest_id;
ALTER TABLE invitations DROP COLUMN IF EXISTS event_guest_id;

DROP INDEX IF EXISTS idx_event_guests_guest;
DROP INDEX IF EXISTS idx_event_guests_tenant_event;
DROP INDEX IF EXISTS idx_event_guests_unique_active;
DROP TABLE IF EXISTS event_guests;

-- +goose StatementEnd
