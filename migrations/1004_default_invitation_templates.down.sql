-- +goose Down
-- +goose StatementBegin
DELETE FROM communication_templates
WHERE is_system = TRUE
  AND type = 'invitation'
  AND name IN ('Undangan Standar WhatsApp', 'Undangan Standar Email');

DROP TRIGGER IF EXISTS guestflow_seed_default_communication_templates_trigger ON tenants;
DROP FUNCTION IF EXISTS guestflow_seed_default_communication_templates();
-- +goose StatementEnd
