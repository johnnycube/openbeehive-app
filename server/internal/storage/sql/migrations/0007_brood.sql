-- 0007_brood.sql
-- Brood-stage observations on an inspection: age (in days) of the youngest
-- larva seen — used to time swarm-control interventions — and whether capped
-- (sealed) brood was present.

ALTER TABLE inspection ADD COLUMN youngest_larva INTEGER NOT NULL DEFAULT 0;
ALTER TABLE inspection ADD COLUMN covered_larva  BOOLEAN NOT NULL DEFAULT 0;
