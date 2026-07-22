-- +goose Down
-- +goose StatementBegin

UPDATE guest_gifts SET amount = 1 WHERE amount IS NULL;

ALTER TABLE guest_gifts
    DROP CONSTRAINT IF EXISTS guest_gifts_amount_check;

ALTER TABLE guest_gifts
    ADD CONSTRAINT guest_gifts_amount_check CHECK (amount > 0);

ALTER TABLE guest_gifts
    ALTER COLUMN amount SET NOT NULL;

-- +goose StatementEnd
