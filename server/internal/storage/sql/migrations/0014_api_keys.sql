-- 0014_api_keys.sql
-- Long-lived API keys for scripts and integrations (hive scales, exports).
-- A key is bound to one user and one tenant; the plaintext is shown once and
-- only its SHA-256 hash is stored. key_prefix keeps the first characters for
-- display so a user can tell keys apart in Settings. scope is 'write' or
-- 'read' (read-only keys may call only Get*/List*/Pull/Subscribe); expires_at
-- is NULL for keys that never expire.
CREATE TABLE IF NOT EXISTS api_key (
  id              TEXT PRIMARY KEY,
  user_id         VARCHAR(64) NOT NULL,
  organization_id VARCHAR(64) NOT NULL,
  name            TEXT NOT NULL DEFAULT '',
  key_prefix      TEXT NOT NULL DEFAULT '',
  key_hash        VARCHAR(64) NOT NULL UNIQUE,
  scope           TEXT NOT NULL DEFAULT 'write',
  created_at      TIMESTAMP NOT NULL,
  expires_at      TIMESTAMP,
  last_used_at    TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_api_key_user ON api_key (user_id);
