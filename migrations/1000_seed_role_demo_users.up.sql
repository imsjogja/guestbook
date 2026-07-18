-- +goose Up
-- +goose StatementBegin
-- Seed one demo account for every tenant role.
-- All accounts use password: password123. These accounts are for development/demo use only.

INSERT INTO users (
    id, email, password_hash, full_name, status, email_verified_at, created_at, updated_at
)
VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a21', 'owner@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo Tenant Owner', 'active', NOW(), NOW(), NOW()),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'manager@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo Event Manager', 'active', NOW(), NOW(), NOW()),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a23', 'rsvp@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo RSVP Officer', 'active', NOW(), NOW(), NOW()),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a24', 'registration@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo Registration Officer', 'active', NOW(), NOW(), NOW()),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a25', 'usher@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo Usher', 'active', NOW(), NOW(), NOW()),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a26', 'gift@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo Gift Officer', 'active', NOW(), NOW(), NOW()),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a27', 'viewer@guestflow.id',
     '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
     'Demo Viewer', 'active', NOW(), NOW(), NOW())
ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    full_name = EXCLUDED.full_name,
    status = 'active',
    email_verified_at = COALESCE(users.email_verified_at, EXCLUDED.email_verified_at),
    deleted_at = NULL,
    updated_at = NOW();

WITH role_accounts(email, role) AS (
    VALUES
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
    ra.role,
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12',
    NOW(),
    NOW(),
    'active',
    NOW(),
    NOW()
FROM role_accounts ra
JOIN users u ON u.email = ra.email
ON CONFLICT (tenant_id, user_id) DO UPDATE SET
    role = EXCLUDED.role,
    status = 'active',
    joined_at = COALESCE(tenant_users.joined_at, EXCLUDED.joined_at),
    updated_at = NOW();

-- +goose StatementEnd
