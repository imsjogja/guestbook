-- +goose Down
-- +goose StatementBegin

DELETE FROM tenant_users
WHERE tenant_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'
  AND user_id IN (
      SELECT id FROM users WHERE email IN (
          'owner@guestflow.id',
          'manager@guestflow.id',
          'rsvp@guestflow.id',
          'registration@guestflow.id',
          'usher@guestflow.id',
          'gift@guestflow.id',
          'viewer@guestflow.id'
      )
  );

DELETE FROM users
WHERE email IN (
    'owner@guestflow.id',
    'manager@guestflow.id',
    'rsvp@guestflow.id',
    'registration@guestflow.id',
    'usher@guestflow.id',
    'gift@guestflow.id',
    'viewer@guestflow.id'
);

-- +goose StatementEnd
