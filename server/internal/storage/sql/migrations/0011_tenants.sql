-- 0011_tenants.sql
-- Tenant invites: a tenant admin (member role 'owner') can invite users by email.
-- The organization (tenant) and member tables already exist (0001_init).
CREATE TABLE IF NOT EXISTS invite (
  id              TEXT PRIMARY KEY,
  organization_id TEXT NOT NULL,
  email           TEXT NOT NULL,
  role            TEXT NOT NULL DEFAULT 'member',
  token           TEXT NOT NULL,
  created_at      TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_invite_token ON invite (token);
