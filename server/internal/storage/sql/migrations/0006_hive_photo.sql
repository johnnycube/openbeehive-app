-- 0006_hive_photo.sql
-- A representative photo per hive, stored as a data URL (synced as a scalar,
-- per-field LWW).

ALTER TABLE hive ADD COLUMN photo TEXT NOT NULL DEFAULT '';
