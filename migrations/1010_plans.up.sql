-- +goose Up
-- +goose StatementBegin
-- Migration 1010: Plans & Subscriptions
-- Defines available subscription plans and tracks tenant subscriptions.

-- Plans table: defines each pricing tier and its limits
CREATE TABLE IF NOT EXISTS plans (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(50)  NOT NULL,         -- internal key: 'starter', 'pro', 'enterprise'
    display_name  VARCHAR(100) NOT NULL,          -- user-facing: 'Starter', 'Pro', 'Enterprise'
    billing_cycle VARCHAR(20)  NOT NULL DEFAULT 'monthly', -- 'monthly', 'yearly'
    price_idr     INTEGER      NOT NULL,          -- price in Indonesian Rupiah
    -- Resource limits (NULL = unlimited)
    max_guests              INTEGER,
    max_events              INTEGER,
    max_team_members        INTEGER,
    max_campaigns_per_month INTEGER,
    max_csv_import_rows     INTEGER,
    -- Boolean feature flags stored in JSONB for flexibility
    features      JSONB        NOT NULL DEFAULT '{}',
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    sort_order    INTEGER      NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_plans_name_cycle ON plans(name, billing_cycle) WHERE is_active = TRUE;

-- Subscriptions table: tracks the active subscription per tenant
CREATE TABLE IF NOT EXISTS subscriptions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    plan_id                 UUID         NOT NULL REFERENCES plans(id),
    status                  VARCHAR(20)  NOT NULL DEFAULT 'active', -- 'active', 'expired', 'cancelled', 'pending'
    billing_cycle           VARCHAR(20)  NOT NULL DEFAULT 'monthly', -- 'monthly', 'yearly'
    started_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at              TIMESTAMPTZ,           -- NULL = manual/lifetime
    cancelled_at            TIMESTAMPTZ,
    midtrans_order_id       VARCHAR(255),
    midtrans_transaction_id VARCHAR(255),
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_id ON subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status) WHERE status = 'active';

-- Trigger for updated_at
DROP TRIGGER IF EXISTS update_plans_updated_at ON plans;
CREATE TRIGGER update_plans_updated_at
    BEFORE UPDATE ON plans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_subscriptions_updated_at ON subscriptions;
CREATE TRIGGER update_subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ─── Seed default plans ────────────────────────────────────────────────────────

-- STARTER - MONTHLY
INSERT INTO plans (name, display_name, billing_cycle, price_idr, max_guests, max_events, max_team_members, max_campaigns_per_month, max_csv_import_rows, features, sort_order)
VALUES (
    'starter', 'Starter', 'monthly', 199000,
    500, 1, 3, 3, 500,
    '{
        "whatsapp_campaign": false,
        "custom_template": false,
        "webhook": false,
        "advanced_reports": false,
        "remove_branding": false,
        "priority_support": false
    }',
    10
) ON CONFLICT DO NOTHING;

-- STARTER - YEARLY (diskon ~17%, equiv ~Rp 166rb/bulan)
INSERT INTO plans (name, display_name, billing_cycle, price_idr, max_guests, max_events, max_team_members, max_campaigns_per_month, max_csv_import_rows, features, sort_order)
VALUES (
    'starter', 'Starter', 'yearly', 1990000,
    500, 1, 3, 3, 500,
    '{
        "whatsapp_campaign": false,
        "custom_template": false,
        "webhook": false,
        "advanced_reports": false,
        "remove_branding": false,
        "priority_support": false
    }',
    11
) ON CONFLICT DO NOTHING;

-- PRO - MONTHLY
INSERT INTO plans (name, display_name, billing_cycle, price_idr, max_guests, max_events, max_team_members, max_campaigns_per_month, max_csv_import_rows, features, sort_order)
VALUES (
    'pro', 'Pro', 'monthly', 499000,
    2000, 3, 10, NULL, NULL,
    '{
        "whatsapp_campaign": true,
        "custom_template": true,
        "webhook": true,
        "advanced_reports": true,
        "remove_branding": false,
        "priority_support": false
    }',
    20
) ON CONFLICT DO NOTHING;

-- PRO - YEARLY (diskon ~17%, equiv ~Rp 415rb/bulan)
INSERT INTO plans (name, display_name, billing_cycle, price_idr, max_guests, max_events, max_team_members, max_campaigns_per_month, max_csv_import_rows, features, sort_order)
VALUES (
    'pro', 'Pro', 'yearly', 4990000,
    2000, 3, 10, NULL, NULL,
    '{
        "whatsapp_campaign": true,
        "custom_template": true,
        "webhook": true,
        "advanced_reports": true,
        "remove_branding": false,
        "priority_support": false
    }',
    21
) ON CONFLICT DO NOTHING;

-- ENTERPRISE - MONTHLY
INSERT INTO plans (name, display_name, billing_cycle, price_idr, max_guests, max_events, max_team_members, max_campaigns_per_month, max_csv_import_rows, features, sort_order)
VALUES (
    'enterprise', 'Enterprise', 'monthly', 1199000,
    NULL, NULL, NULL, NULL, NULL,
    '{
        "whatsapp_campaign": true,
        "custom_template": true,
        "webhook": true,
        "advanced_reports": true,
        "remove_branding": true,
        "priority_support": true
    }',
    30
) ON CONFLICT DO NOTHING;

-- ENTERPRISE - YEARLY (diskon ~17%, equiv ~Rp 999rb/bulan)
INSERT INTO plans (name, display_name, billing_cycle, price_idr, max_guests, max_events, max_team_members, max_campaigns_per_month, max_csv_import_rows, features, sort_order)
VALUES (
    'enterprise', 'Enterprise', 'yearly', 11990000,
    NULL, NULL, NULL, NULL, NULL,
    '{
        "whatsapp_campaign": true,
        "custom_template": true,
        "webhook": true,
        "advanced_reports": true,
        "remove_branding": true,
        "priority_support": true
    }',
    31
) ON CONFLICT DO NOTHING;

-- +goose StatementEnd
