-- Migration: Create guest_tags and guest_tag_assignments tables
-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS guest_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7),
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_guest_tags_tenant ON guest_tags(tenant_id);

CREATE TABLE IF NOT EXISTS guest_tag_assignments (
    guest_id UUID NOT NULL REFERENCES guests(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES guest_tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (guest_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_guest_tag_assignments_tag ON guest_tag_assignments(tag_id);

-- +goose StatementEnd
