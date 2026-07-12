-- Migration: Create guest_notes table
-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS guest_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    guest_id UUID NOT NULL REFERENCES guests(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    is_pinned BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_guest_notes_guest ON guest_notes(guest_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_guest_notes_pinned ON guest_notes(guest_id, is_pinned) WHERE is_pinned = TRUE;

-- +goose StatementEnd
