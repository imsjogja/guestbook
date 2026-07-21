-- +goose Up
-- +goose StatementBegin
-- Migration 1011: Payments audit trail

CREATE TABLE IF NOT EXISTS payments (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subscription_id         UUID         REFERENCES subscriptions(id),
    plan_id                 UUID         NOT NULL REFERENCES plans(id),
    midtrans_order_id       VARCHAR(255) NOT NULL UNIQUE,
    midtrans_transaction_id VARCHAR(255),
    amount_idr              INTEGER      NOT NULL,
    billing_cycle           VARCHAR(20)  NOT NULL DEFAULT 'monthly',
    status                  VARCHAR(30)  NOT NULL DEFAULT 'pending',
    -- 'pending', 'success', 'failed', 'expired', 'refunded', 'cancelled'
    payment_method          VARCHAR(50),
    -- e.g. 'gopay', 'bank_transfer', 'credit_card', 'qris'
    va_number               VARCHAR(50),
    -- VA number if applicable
    paid_at                 TIMESTAMPTZ,
    expired_at              TIMESTAMPTZ,
    raw_notification        JSONB,
    -- full payload from Midtrans notification
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payments_tenant_id ON payments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_payments_midtrans_order ON payments(midtrans_order_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);

DROP TRIGGER IF EXISTS update_payments_updated_at ON payments;
CREATE TRIGGER update_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose StatementEnd
