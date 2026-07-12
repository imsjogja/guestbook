-- Migration 002: Rollback - Drop tenants table

DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
DROP TABLE IF EXISTS tenants;
