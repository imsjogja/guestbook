-- +goose Down
-- +goose StatementBegin

DELETE FROM event_guests
WHERE event_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13'
  AND tenant_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'
  AND source = 'manual';

-- +goose StatementEnd
