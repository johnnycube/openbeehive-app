-- 0009_climate_treatment.sql
-- Climate readings on inspections (temperature/humidity, in-hive and outside),
-- and the richer, syncable Treatment (Bestandsbuch) shape.

-- Climate columns are nullable: NULL = not measured (distinct from 0 °C).
ALTER TABLE inspection ADD COLUMN temp_hive REAL;
ALTER TABLE inspection ADD COLUMN temp_outside REAL;
ALTER TABLE inspection ADD COLUMN humidity_hive REAL;
ALTER TABLE inspection ADD COLUMN humidity_outside REAL;

-- Rebuild the treatment table to the richer shape and make it sync-capable
-- (field_hlc + deleted). The old shape was never synced and carried no data.
DROP TABLE IF EXISTS treatment;
CREATE TABLE IF NOT EXISTS treatment (
  id                TEXT PRIMARY KEY,
  organization_id   TEXT NOT NULL,
  apiary_id         TEXT NOT NULL DEFAULT '',
  hive_id           TEXT NOT NULL DEFAULT '',
  queen_id          TEXT NOT NULL DEFAULT '',
  date              TIMESTAMP,
  product           TEXT NOT NULL DEFAULT '',
  active_ingredient TEXT NOT NULL DEFAULT '',
  dose              TEXT NOT NULL DEFAULT '',
  method            TEXT NOT NULL DEFAULT '',
  batch_number      TEXT NOT NULL DEFAULT '',
  withdrawal_until  TIMESTAMP,
  reason            TEXT NOT NULL DEFAULT '',
  note              TEXT NOT NULL DEFAULT '',
  field_hlc         TEXT NOT NULL DEFAULT '{}',
  deleted           BOOLEAN NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_treatment_hive ON treatment (hive_id, date);
