-- 0004_history.sql
-- Full history + statistics foundation.

-- Queen: end of reign -> reign = [introduced_at, replaced_at).
-- Old queens are kept as rows (active = 0), never deleted.
ALTER TABLE queen ADD COLUMN replaced_at TIMESTAMP;

-- Placement history: where a hive lived and when (migratory beekeeping).
-- Open interval = current apiary (end_at IS NULL).
CREATE TABLE IF NOT EXISTS placement (
  id              TEXT PRIMARY KEY,
  organization_id VARCHAR(64) NOT NULL,
  hive_id        VARCHAR(64) NOT NULL,
  apiary_id     VARCHAR(64) NOT NULL,
  start_at             TIMESTAMP NOT NULL,
  end_at             TIMESTAMP,
  field_hlc       TEXT NOT NULL DEFAULT '{}',
  deleted         BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_bs_hive ON placement (hive_id, start_at);

-- Harvest is now per hive, with frozen context.
ALTER TABLE harvest ADD COLUMN hive_id TEXT;
ALTER TABLE harvest ADD COLUMN queen_id TEXT;
ALTER TABLE harvest ADD COLUMN field_hlc TEXT NOT NULL DEFAULT '{}';
ALTER TABLE harvest ADD COLUMN deleted BOOLEAN NOT NULL DEFAULT FALSE;

-- Universal event log = immutable history AND fact table.
-- Each row carries the dimension keys (apiary/hive/queen/time)
-- frozen at event time -> statistics without joins, permanently correct.
CREATE TABLE IF NOT EXISTS event (
  id              TEXT PRIMARY KEY,
  organization_id VARCHAR(64) NOT NULL,
  scope_id        VARCHAR(64) NOT NULL,   -- Apiary-id (sharing/Pull)
  type             INTEGER NOT NULL,       -- see EventType
  date           TIMESTAMP NOT NULL,
  apiary_id     VARCHAR(64),            -- frozen
  hive_id        VARCHAR(64),            -- frozen
  queen_id     VARCHAR(64),            -- frozen (reigning at the time)
  ref_entity      TEXT,                   -- Detaildatensatz, z.B. "harvest"
  ref_id          TEXT,
  title           TEXT NOT NULL DEFAULT '',
  amount_kg        REAL NOT NULL DEFAULT 0, -- schnelle Ertragsstatistik
  detail          TEXT NOT NULL DEFAULT '', -- JSON
  author_id       TEXT NOT NULL DEFAULT '',
  field_hlc       TEXT NOT NULL DEFAULT '{}',
  deleted         BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_event_scope ON event (scope_id, date);
CREATE INDEX IF NOT EXISTS idx_event_hive ON event (hive_id, date);
CREATE INDEX IF NOT EXISTS idx_event_queen ON event (queen_id);
CREATE INDEX IF NOT EXISTS idx_event_apiary ON event (apiary_id, date);
