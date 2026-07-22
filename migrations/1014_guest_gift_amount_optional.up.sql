-- +goose Up
-- +goose StatementBegin

ALTER TABLE guest_gifts
    ALTER COLUMN amount DROP NOT NULL;

ALTER TABLE guest_gifts
    DROP CONSTRAINT IF EXISTS guest_gifts_amount_check;

ALTER TABLE guest_gifts
    ADD CONSTRAINT guest_gifts_amount_check CHECK (amount IS NULL OR amount > 0);

-- +goose StatementEnd
