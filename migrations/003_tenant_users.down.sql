-- Migration 003: Rollback - Drop tenant_users table

DROP TRIGGER IF EXISTS update_tenant_users_updated_at ON tenant_users;
DROP TABLE IF EXISTS tenant_users;
