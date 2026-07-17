-- 0005_stockkarte.sql
-- Richer inspection ("Stockkarte") fields: colony state, a behaviour rating and
-- the activities carried out during the visit (feeding, frame management,
-- drone-frame cutting, supering, weighing).

ALTER TABLE inspection ADD COLUMN brood_frames    INTEGER NOT NULL DEFAULT 0;
ALTER TABLE inspection ADD COLUMN calmness        INTEGER NOT NULL DEFAULT 0;
ALTER TABLE inspection ADD COLUMN fed_kg          REAL    NOT NULL DEFAULT 0;
ALTER TABLE inspection ADD COLUMN frames_added    INTEGER NOT NULL DEFAULT 0;
ALTER TABLE inspection ADD COLUMN frames_removed  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE inspection ADD COLUMN drone_frame_cut BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE inspection ADD COLUMN super_added     BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE inspection ADD COLUMN weight_kg       REAL    NOT NULL DEFAULT 0;
