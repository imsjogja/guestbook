-- +goose Up
-- +goose StatementBegin
-- Restore the demo workspace accounts and active memberships after accidental
-- deletion. These credentials are for local/demo use only.

INSERT INTO users (
    id, email, password_hash, full_name, status, email_verified_at, created_at, updated_at, deleted_at
)
VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'demo@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo Event Manager', 'active', NOW(), NOW(), NOW(), NULL),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a21', 'owner@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo Tenant Owner', 'active', NOW(), NOW(), NOW(), NULL),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'manager@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo Event Manager', 'active', NOW(), NOW(), NOW(), NULL),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a23', 'rsvp@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo RSVP Officer', 'active', NOW(), NOW(), NOW(), NULL),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a24', 'registration@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo Registration Officer', 'active', NOW(), NOW(), NOW(), NULL),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a25', 'usher@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo Usher', 'active', NOW(), NOW(), NOW(), NULL),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a26', 'gift@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo Gift Officer', 'active', NOW(), NOW(), NOW(), NULL),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a27', 'viewer@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo Viewer', 'active', NOW(), NOW(), NOW(), NULL)
ON CONFLICT (id) DO UPDATE SET
    email = EXCLUDED.email,
    password_hash = EXCLUDED.password_hash,
    full_name = EXCLUDED.full_name,
    status = EXCLUDED.status,
    email_verified_at = EXCLUDED.email_verified_at,
    deleted_at = NULL,
    updated_at = NOW();

WITH demo_members(email, role) AS (
    VALUES
        ('demo@guestflow.id', 'tenant_owner'),
        ('owner@guestflow.id', 'tenant_owner'),
        ('manager@guestflow.id', 'event_manager'),
        ('rsvp@guestflow.id', 'rsvp_officer'),
        ('registration@guestflow.id', 'registration_officer'),
        ('usher@guestflow.id', 'usher'),
        ('gift@guestflow.id', 'gift_officer'),
        ('viewer@guestflow.id', 'viewer')
)
INSERT INTO tenant_users (
    id, tenant_id, user_id, role, invited_by, invited_at, joined_at, status, created_at, updated_at
)
SELECT
    gen_random_uuid(),
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    u.id,
    dm.role,
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12',
    NOW(), NOW(), 'active', NOW(), NOW()
FROM demo_members dm
JOIN users u ON u.email = dm.email
ON CONFLICT (tenant_id, user_id) DO UPDATE SET
    role = EXCLUDED.role,
    invited_by = EXCLUDED.invited_by,
    invited_at = EXCLUDED.invited_at,
    joined_at = EXCLUDED.joined_at,
    status = EXCLUDED.status,
    updated_at = NOW();

-- +goose StatementEnd
