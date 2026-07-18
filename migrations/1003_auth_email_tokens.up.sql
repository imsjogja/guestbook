-- +goose Up
-- +goose StatementBegin
-- One-time tokens for password reset and passwordless login.
CREATE TABLE IF NOT EXISTS auth_email_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose VARCHAR(32) NOT NULL CHECK (purpose IN ('password_reset', 'magic_login')),
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_email_tokens_active
    ON auth_email_tokens(user_id, purpose)
    WHERE used_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_auth_email_tokens_expiry
    ON auth_email_tokens(expires_at)
    WHERE used_at IS NULL;
-- +goose StatementEnd
