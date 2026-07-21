-- +goose Up
-- +goose StatementBegin
-- GuestFlow - Seed Data for Development
--
-- Run after all migrations to populate the database with:
-- - System roles and permissions
-- - Demo tenant and user
-- - Sample event with sessions
-- - Sample guests
-- - Sample templates
--
-- DO NOT RUN IN PRODUCTION

-- Seed system roles
INSERT INTO roles (id, name, display_name, description, permissions, is_system, tenant_id, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'tenant_owner', 'Tenant Owner', 'Full access to tenant resources. Can manage billing, team, and all settings.',
     ARRAY['guest:read','guest:write','guest:delete','guest:import','guest:export',
           'event:read','event:write','event:delete','invitation:read','invitation:write','invitation:send',
           'rsvp:read','rsvp:write','checkin:read','checkin:write','seating:read','seating:write',
           'report:read','report:export','communication:read','communication:write','communication:send',
           'billing:read','billing:write','team:read','team:write','team:invite',
           'settings:read','settings:write','audit:read'], true, NULL, NOW(), NOW()),

    (gen_random_uuid(), 'event_manager', 'Event Manager', 'Can manage events, guests, invitations, RSVPs, check-ins, and seating.',
     ARRAY['guest:read','guest:write','guest:delete','guest:import','guest:export',
           'event:read','event:write','event:delete','invitation:read','invitation:write','invitation:send',
           'rsvp:read','rsvp:write','checkin:read','checkin:write','seating:read','seating:write',
           'report:read','report:export','communication:read','communication:write','communication:send',
           'team:read','settings:read'], true, NULL, NOW(), NOW()),

    (gen_random_uuid(), 'rsvp_officer', 'RSVP Officer', 'Monitors RSVPs and communicates with guests.',
     ARRAY['guest:read','guest:write','rsvp:read','rsvp:write',
           'communication:read','communication:write','communication:send',
           'report:read'], true, NULL, NOW(), NOW()),

    (gen_random_uuid(), 'registration_officer', 'Registration Officer', 'Scans QR codes and processes check-ins. Sees minimal guest data.',
     ARRAY['checkin:read','checkin:write','guest:read'], true, NULL, NOW(), NOW()),

    (gen_random_uuid(), 'usher', 'Usher', 'Guides guests. Sees arrival status and seating info only.',
     ARRAY['checkin:read','seating:read'], true, NULL, NOW(), NOW()),

    (gen_random_uuid(), 'gift_officer', 'Gift Officer', 'Records gifts and souvenirs. No access to financial data.',
     ARRAY['guest:read'], true, NULL, NOW(), NOW()),

    (gen_random_uuid(), 'viewer', 'Viewer', 'Read-only access to dashboards and reports.',
     ARRAY['report:read','event:read','guest:read'], true, NULL, NOW(), NOW());

-- Seed demo tenant
INSERT INTO tenants (id, name, slug, description, primary_color, status, created_at, updated_at)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'Demo Wedding Organizer',
    'demo-wo',
    'Demo tenant for testing GuestFlow platform',
    '#8B5CF6',
    'active',
    NOW(), NOW()
);

-- Seed demo user (password: 'password123' - bcrypt hashed)
-- Hash: $2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK
INSERT INTO users (id, email, password_hash, full_name, phone, status, email_verified_at, created_at, updated_at)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12',
    'demo@guestflow.id',
    '$2a$12$coSXHcPzsYI8U/ATJCGLdOoteKOz/o0gYhzibamOdQLWLC9tY9oXK',
    'Demo Event Manager',
    '+6281234567890',
    'active',
    NOW(),
    NOW(), NOW()
);

-- Assign user as tenant owner
INSERT INTO tenant_users (id, tenant_id, user_id, role, invited_at, joined_at, status, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12',
    'tenant_owner',
    NOW(), NOW(), 'active',
    NOW(), NOW()
);

-- Seed sample wedding event
INSERT INTO events (id, tenant_id, name, type, description, status, start_date, end_date, rsvp_deadline,
    capacity, target_invites, target_attendance, dress_code, privacy_notice, guest_policy,
    settings, created_by, created_at, updated_at)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'Pernikahan Andi & Rina',
    'wedding',
    'Dengan penuh suka cita, kami mengundang Anda untuk hadir dan memberikan doa restu di acara pernikahan kami.',
    'published',
    '2026-08-15 10:00:00+07',
    '2026-08-15 22:00:00+07',
    '2026-08-08 23:59:00+07',
    500, 400, 350,
    'Batik / Kebaya',
    'Data pribadi Anda akan digunakan hanya untuk keperluan acara ini dan tidak akan dibagikan ke pihak ketiga.',
    'Mohon konfirmasi kehadiran sebelum batas waktu RSVP. Tamu yang tidak mengkonfirmasi mungkin tidak dapat kami akomodasi.',
    '{"theme": "elegant", "primary_color": "#8B5CF6"}'::jsonb,
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12',
    NOW(), NOW()
);

-- Seed event sessions
INSERT INTO event_sessions (id, event_id, name, description, start_time, end_time, capacity, sort_order, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13',
     'Akad Nikah', 'Prosesi akad nikah untuk keluarga terdekat.',
     '2026-08-15 10:00:00+07', '2026-08-15 11:30:00+07', 100, 0, NOW(), NOW()),
    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13',
     'Resepsi', 'Resepsi pernikahan terbuka untuk seluruh tamu undangan.',
     '2026-08-15 13:00:00+07', '2026-08-15 17:00:00+07', 400, 1, NOW(), NOW()),
    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13',
     'After Party', 'Perayaan santai untuk teman-teman dekat.',
     '2026-08-15 19:00:00+07', '2026-08-15 22:00:00+07', 100, 2, NOW(), NOW());

-- Seed event location
INSERT INTO event_locations (id, tenant_id, name, address, city, maps_url, instructions, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'Gedung Graha Bhakti Budaya',
    'Jl. Sumenep No. 6, Menteng',
    'Jakarta Pusat',
    'https://maps.google.com/?q=Gedung+Graha+Bhakti+Budaya',
    'Parkir tersedia di area depan gedung. Masuk melalui pintu utama.',
    NOW(), NOW()
);

-- Seed sample guests
INSERT INTO guests (id, tenant_id, full_name, nickname, phone, email, city, country, language, guest_type, segment, institution, relationship, notes, consent_communication, is_active, created_by, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     'Budi Santoso', 'Budi', '+6281111111111', 'budi@email.com', 'Jakarta', 'Indonesia', 'id',
     'family', 'Keluarga Pengantin Pria', NULL, 'Paman Andi', NULL, true, true,
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', NOW(), NOW()),

    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     'Siti Aminah', 'Siti', '+6281222222222', 'siti@email.com', 'Jakarta', 'Indonesia', 'id',
     'family', 'Keluarga Pengantin Wanita', NULL, 'Bibi Rina', NULL, true, true,
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', NOW(), NOW()),

    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     'Dr. Ahmad Wijaya', 'Ahmad', '+6281333333333', 'ahmad@kantor.com', 'Bandung', 'Indonesia', 'id',
     'vip', 'Rekan Kerja', 'PT Maju Jaya', 'Atasan Andi', 'Vegetarian', true, true,
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', NOW(), NOW()),

    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     'Dewi Kusuma', 'Dewi', '+6281444444444', 'dewi@kantor.com', 'Jakarta', 'Indonesia', 'id',
     'colleague', 'Rekan Kerja', 'PT Maju Jaya', 'Rekan Andi', NULL, true, true,
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', NOW(), NOW()),

    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     'Hendra Gunawan', 'Hendra', '+6281555555555', 'hendra@email.com', 'Surabaya', 'Indonesia', 'id',
     'friend', 'Teman Kuliah', NULL, 'Teman SMA Andi', NULL, true, true,
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', NOW(), NOW()),

    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     'Maya Sari', 'Maya', '+6281666666666', 'maya@email.com', 'Jakarta', 'Indonesia', 'id',
     'friend', 'Teman Kuliah', NULL, 'Teman kuliah Rina', 'Alergi kacang', true, true,
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', NOW(), NOW()),

    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     'Ir. Bambang Soeparno', 'Pak Bambang', '+6281777777777', 'bambang@pemda.go.id', 'Jakarta', 'Indonesia', 'id',
     'government', 'Pejabat', 'Dinas Pariwisata DKI', 'Pihak berwenang', 'Kursi roda', true, true,
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', NOW(), NOW()),

    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     'Ratna Dewi', 'Ratna', '+6281888888888', 'ratna@email.com', 'Yogyakarta', 'Indonesia', 'id',
     'vip', 'Saudara', NULL, 'Kakak Rina', NULL, true, true,
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', NOW(), NOW());

-- Seed communication templates
INSERT INTO communication_templates (id, tenant_id, name, channel, type, subject, body, variables, is_active, is_system, description, language, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     'Undangan Pernikahan', 'whatsapp', 'invitation', NULL,
     'Assalamualaikum {{guest_name}},

Dengan hormat, kami sampaikan undangan pernikahan:

{{event_name}}
📅 {{event_date}}
📍 {{event_location}}

Konfirmasi kehadiran: {{rsvp_link}}

Merupakan suatu kehormatan dan kebahagiaan bagi kami apabila Bapak/Ibu/Saudara/i berkenan hadir untuk memberikan doa restu.

Hormat kami,
{{host_names}}',
     to_jsonb(ARRAY['guest_name','event_name','event_date','event_location','rsvp_link','host_names']),
     true, true, 'Template undangan pernikahan via WhatsApp', 'id',
     NOW(), NOW()),

    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     'Reminder RSVP', 'whatsapp', 'reminder', NULL,
     'Halo {{guest_name}},

Ini pengingat bahwa batas konfirmasi kehadiran untuk {{event_name}} adalah {{rsvp_deadline}}.

Mohon konfirmasi melalui: {{rsvp_link}}

Terima kasih!',
     to_jsonb(ARRAY['guest_name','event_name','rsvp_deadline','rsvp_link']),
     true, true, 'Template pengingat RSVP', 'id',
     NOW(), NOW()),

    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     'Konfirmasi Kehadiran', 'email', 'confirmation',
     'Konfirmasi Kehadiran - {{event_name}}',
     '<html><body>
<h2>Terima Kasih!</h2>
<p>Halo {{guest_name}},</p>
<p>Terima kasih telah mengkonfirmasi kehadiran Anda untuk:</p>
<h3>{{event_name}}</h3>
<p>📅 {{event_date}}<br>
📍 {{event_location}}</p>
<p>QR Code check-in Anda terlampir.</p>
<p>Sampai jumpa di acara!</p>
</body></html>',
     to_jsonb(ARRAY['guest_name','event_name','event_date','event_location']),
     true, true, 'Template konfirmasi kehadiran via email', 'id',
     NOW(), NOW());

-- Update event with location reference
UPDATE events
SET primary_location_id = (SELECT id FROM event_locations LIMIT 1)
WHERE id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13';
-- +goose StatementEnd
