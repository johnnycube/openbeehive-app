-- 0008_webauthn.sql
-- Passkey (WebAuthn) users and their stored credentials. The credential is
-- kept as JSON (the go-webauthn Credential), keyed for lookup by its id.

CREATE TABLE IF NOT EXISTS webauthn_user (
  id           TEXT PRIMARY KEY,
  name         TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS webauthn_credential (
  id      TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  cred    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_wa_cred_user ON webauthn_credential (user_id);
