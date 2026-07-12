-- Migration: Create guests table with indexes for the Guest Management module
-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS guests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    nickname VARCHAR(255),
    phone VARCHAR(50),
    email VARCHAR(255),
    address TEXT,
    city VARCHAR(100),
    country VARCHAR(100) DEFAULT 'Indonesia',
    language VARCHAR(10) DEFAULT 'id',
    guest_type VARCHAR(50) NOT NULL,
    segment VARCHAR(100),
    institution VARCHAR(255),
    title VARCHAR(100),
    relationship VARCHAR(100),
    pic VARCHAR(255),
    accessibility_needs TEXT,
    dietary_restrictions TEXT,
    allergies TEXT,
    notes TEXT,
    consent_communication BOOLEAN DEFAULT FALSE,
    consent_version VARCHAR(50),
    source VARCHAR(50) DEFAULT 'manual',
    is_active BOOLEAN DEFAULT TRUE,
    created_by UUID NOT NULL REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_guests_tenant ON guests(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_guests_search ON guests(tenant_id) INCLUDE (full_name, phone, email);
CREATE INDEX IF NOT EXISTS idx_guests_type ON guests(tenant_id, guest_type);
CREATE INDEX IF NOT EXISTS idx_guests_segment ON guests(tenant_id, segment);
CREATE INDEX IF NOT EXISTS idx_guests_active ON guests(tenant_id, is_active);
CREATE INDEX IF NOT EXISTS idx_guests_phone ON guests(tenant_id, phone) WHERE phone IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_guests_email ON guests(tenant_id, email) WHERE email IS NOT NULL;

-- +goose StatementEnd
