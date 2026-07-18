-- +goose Down
-- +goose StatementBegin

DELETE FROM tenant_users
WHERE tenant_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'
  AND user_id IN (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a21',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a23',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a24',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a25',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a26',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a27'
  );

-- Keep user records intact on rollback to avoid deleting accounts that may
-- have acquired unrelated audit or event data after this repair.

-- +goose StatementEnd
