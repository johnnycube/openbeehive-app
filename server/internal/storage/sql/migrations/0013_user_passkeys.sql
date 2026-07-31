-- Passkeys bound to registered accounts. The legacy webauthn_user /
-- webauthn_credential silo predates the invite system: its register endpoint
-- created accounts for anonymous visitors and its logins minted owner
-- sessions, bypassing invite-only registration entirely. Credentials there
-- were never linked to users rows, so there is nothing to migrate — affected
-- installs re-enroll from Settings after signing in normally.
DROP TABLE IF EXISTS webauthn_credential;
DROP TABLE IF EXISTS webauthn_user;

CREATE TABLE IF NOT EXISTS user_passkey (
  id           TEXT PRIMARY KEY,          -- credential id (raw, base64url)
  user_id      TEXT NOT NULL,             -- users.id
  name         TEXT NOT NULL DEFAULT '',  -- user-facing label ("MacBook", "YubiKey")
  cred         TEXT NOT NULL,             -- go-webauthn Credential JSON
  created_at   TIMESTAMP NOT NULL,
  last_used_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_user_passkey_user ON user_passkey (user_id);
