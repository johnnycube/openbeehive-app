-- 0003_field_lww.sql
-- Per-field LWW: every row gets a "field clock" (JSON: field -> hlc).
-- Set fields (e.g. inspection.photo_keys) now store OR-Set JSON
-- instead of a plain list; their merge is by union, not LWW.

ALTER TABLE apiary   ADD COLUMN field_hlc TEXT NOT NULL DEFAULT '{}';
ALTER TABLE hive      ADD COLUMN field_hlc TEXT NOT NULL DEFAULT '{}';
ALTER TABLE queen   ADD COLUMN field_hlc TEXT NOT NULL DEFAULT '{}';
ALTER TABLE inspection ADD COLUMN field_hlc TEXT NOT NULL DEFAULT '{}';
ALTER TABLE task    ADD COLUMN field_hlc TEXT NOT NULL DEFAULT '{}';
