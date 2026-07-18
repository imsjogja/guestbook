-- +goose Up
-- +goose StatementBegin

-- rsvp_responses table: stores guest RSVP submissions
CREATE TABLE IF NOT EXISTS rsvp_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    invitation_id UUID NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    guest_id UUID NOT NULL REFERENCES guests(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    attending_pax INTEGER NOT NULL DEFAULT 0,
    adults INTEGER NOT NULL DEFAULT 0,
    children INTEGER NOT NULL DEFAULT 0,
    menu_choice VARCHAR(255),
    allergies TEXT,
    accessibility_needs TEXT,
    transportation VARCHAR(255),
    notes TEXT,
    responded_at TIMESTAMPTZ,
    edited_at TIMESTAMPTZ,
    edited_by UUID REFERENCES users(id),
    ip_address INET,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for efficient RSVP lookups
CREATE INDEX IF NOT EXISTS idx_rsvp_tenant ON rsvp_responses(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rsvp_event ON rsvp_responses(event_id);
CREATE INDEX IF NOT EXISTS idx_rsvp_invitation ON rsvp_responses(invitation_id);
CREATE INDEX IF NOT EXISTS idx_rsvp_guest ON rsvp_responses(guest_id);
CREATE INDEX IF NOT EXISTS idx_rsvp_status ON rsvp_responses(status);
CREATE INDEX IF NOT EXISTS idx_rsvp_event_status ON rsvp_responses(event_id, status);
CREATE INDEX IF NOT EXISTS idx_rsvp_responded_at ON rsvp_responses(responded_at);

-- Unique constraint: one RSVP per invitation
CREATE UNIQUE INDEX IF NOT EXISTS idx_rsvp_unique_invitation ON rsvp_responses(invitation_id);

-- +goose StatementEnd
