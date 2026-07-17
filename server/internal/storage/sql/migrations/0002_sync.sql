-- 0002_sync.sql
-- Makes the schema offline/sync capable: HLC per row, tombstones,
-- a change log as sync feed and apiary sharing.

-- HLC + tombstone on every synced entity.
ALTER TABLE apiary   ADD COLUMN updated_hlc TEXT NOT NULL DEFAULT '';
ALTER TABLE apiary   ADD COLUMN deleted BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE hive      ADD COLUMN updated_hlc TEXT NOT NULL DEFAULT '';
ALTER TABLE hive      ADD COLUMN deleted BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE queen   ADD COLUMN updated_hlc TEXT NOT NULL DEFAULT '';
ALTER TABLE queen   ADD COLUMN deleted BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE inspection ADD COLUMN updated_hlc TEXT NOT NULL DEFAULT '';
ALTER TABLE inspection ADD COLUMN deleted BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE task    ADD COLUMN updated_hlc TEXT NOT NULL DEFAULT '';
ALTER TABLE task    ADD COLUMN deleted BOOLEAN NOT NULL DEFAULT 0;

-- Receive sequence: portable application-level counter (one row, atomically incremented).
CREATE TABLE IF NOT EXISTS seq_counter (
  name TEXT PRIMARY KEY,
  val  BIGINT NOT NULL DEFAULT 0
);
INSERT INTO seq_counter (name, val) VALUES ('change', 0);

-- The sync feed. seq = server-assigned receive order (cursor).
CREATE TABLE IF NOT EXISTS change_log (
  id        TEXT PRIMARY KEY,
  seq       BIGINT NOT NULL,
  scope_id  TEXT NOT NULL,
  entity    TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  op        INTEGER NOT NULL,         -- 1 upsert, 2 delete
  payload   TEXT NOT NULL DEFAULT '', -- JSON snapshot of the row
  hlc       TEXT NOT NULL,            -- origin HLC (for LWW)
  author_id TEXT NOT NULL,
  org_id    TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_change_seq ON change_log (seq);
CREATE INDEX IF NOT EXISTS idx_change_scope_seq ON change_log (scope_id, seq);

-- Apiary-sharing: wer darf welchen Apiary sehen/bearbeiten.
CREATE TABLE IF NOT EXISTS apiary_share (
  apiary_id TEXT NOT NULL,
  benutzer_id TEXT NOT NULL,
  role       TEXT NOT NULL DEFAULT 'imker', -- viewer | imker | owner
  created_at  TIMESTAMP NOT NULL,
  PRIMARY KEY (apiary_id, benutzer_id)
);
CREATE INDEX IF NOT EXISTS idx_share_user ON apiary_share (benutzer_id);
