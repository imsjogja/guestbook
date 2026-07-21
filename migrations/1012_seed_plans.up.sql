-- +goose Up
-- +goose StatementBegin
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

-- STARTER - YEARLY
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

-- PRO - YEARLY
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
    'enterprise', 'Enterprise', 'monthly', 1299000,
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

-- ENTERPRISE - YEARLY
INSERT INTO plans (name, display_name, billing_cycle, price_idr, max_guests, max_events, max_team_members, max_campaigns_per_month, max_csv_import_rows, features, sort_order)
VALUES (
    'enterprise', 'Enterprise', 'yearly', 12990000,
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
