-- +goose Up
-- +goose StatementBegin
-- Migration 006: Create refresh_tokens table
-- Stores refresh tokens for JWT authentication with token rotation support.
-- Tokens are stored as SHA-256 hashes (never store raw tokens).
-- Revoked tokens are kept for audit purposes rather than hard-deleted.

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id),
    device_info TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for looking up tokens by hash (used during refresh)
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash
    ON refresh_tokens(token_hash)
    WHERE revoked_at IS NULL;

-- Index for looking up a user's active tokens
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user
    ON refresh_tokens(user_id)
    WHERE revoked_at IS NULL;

-- Index for cleanup of expired tokens
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires
    ON refresh_tokens(expires_at)
    WHERE revoked_at IS NULL;
-- +goose StatementEnd
