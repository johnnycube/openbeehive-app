-- 0006_hive_photo.sql
-- A representative photo per hive. Stored as a data URL in the local-first
-- flow (synced as a scalar, per-field LWW); a blob key can be used instead
-- once blob upload is wired into the sync path.

ALTER TABLE hive ADD COLUMN photo TEXT NOT NULL DEFAULT '';
