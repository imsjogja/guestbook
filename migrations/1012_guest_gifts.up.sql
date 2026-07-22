-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS guest_gifts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    guest_id UUID NOT NULL REFERENCES guests(id) ON DELETE CASCADE,
    event_guest_id UUID NOT NULL REFERENCES event_guests(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL CHECK (amount > 0),
    gift_type VARCHAR(20) NOT NULL DEFAULT 'cash' CHECK (gift_type IN ('cash', 'transfer', 'goods', 'other')),
    notes TEXT,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    recorded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_guest_gifts_unique_active
    ON guest_gifts(tenant_id, event_id, guest_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_guest_gifts_event
    ON guest_gifts(tenant_id, event_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_guest_gifts_event_guest
    ON guest_gifts(event_guest_id, deleted_at);

DROP TRIGGER IF EXISTS update_guest_gifts_updated_at ON guest_gifts;
CREATE TRIGGER update_guest_gifts_updated_at
    BEFORE UPDATE ON guest_gifts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose StatementEnd
