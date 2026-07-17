-- 0001_init.sql
-- Portables Schema: nur TEXT/INTEGER/REAL/TIMESTAMP/BOOLEAN, String-UUIDs.
-- Dialect specifics are adjusted by Store.translate().

CREATE TABLE IF NOT EXISTS organization (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  plan        TEXT NOT NULL DEFAULT 'hobby',   -- hobby | profi
  created_at  TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS user (
  id           TEXT PRIMARY KEY,
  email        TEXT NOT NULL UNIQUE,
  name         TEXT NOT NULL DEFAULT '',
  oidc_subject TEXT NOT NULL,                  -- provider:sub
  created_at   TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS member (
  organization_id TEXT NOT NULL,
  benutzer_id     TEXT NOT NULL,
  role           TEXT NOT NULL DEFAULT 'owner', -- owner | imker | viewer
  PRIMARY KEY (organization_id, benutzer_id)
);

CREATE TABLE IF NOT EXISTS apiary (
  id              TEXT PRIMARY KEY,
  organization_id TEXT NOT NULL,
  name            TEXT NOT NULL,
  address            TEXT NOT NULL DEFAULT '',
  lat             REAL NOT NULL DEFAULT 0,
  lng             REAL NOT NULL DEFAULT 0,
  note           TEXT NOT NULL DEFAULT '',
  created_at      TIMESTAMP NOT NULL,
  updated_at      TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_apiary_org ON apiary (organization_id);

CREATE TABLE IF NOT EXISTS hive (
  id              TEXT PRIMARY KEY,
  organization_id TEXT NOT NULL,
  apiary_id     TEXT NOT NULL,
  name            TEXT NOT NULL,
  type             INTEGER NOT NULL DEFAULT 0,
  status          INTEGER NOT NULL DEFAULT 1,
  boxes          INTEGER NOT NULL DEFAULT 1,
  colony_origin   TEXT NOT NULL DEFAULT '',
  note           TEXT NOT NULL DEFAULT '',
  qr_code         TEXT NOT NULL DEFAULT '',
  created_at      TIMESTAMP NOT NULL,
  updated_at      TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_hive_org ON hive (organization_id);
CREATE INDEX IF NOT EXISTS idx_placement ON hive (apiary_id);

CREATE TABLE IF NOT EXISTS queen (
  id              TEXT PRIMARY KEY,
  organization_id TEXT NOT NULL,
  hive_id        TEXT NOT NULL,
  year            INTEGER NOT NULL,
  marking      INTEGER NOT NULL DEFAULT 0,
  origin        TEXT NOT NULL DEFAULT '',
  breeder_number     TEXT NOT NULL DEFAULT '',
  introduced_at   TIMESTAMP,
  active           BOOLEAN NOT NULL DEFAULT 1,
  note           TEXT NOT NULL DEFAULT '',
  created_at      TIMESTAMP NOT NULL,
  updated_at      TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_queen_hive ON queen (hive_id);

CREATE TABLE IF NOT EXISTS inspection (
  id               TEXT PRIMARY KEY,
  organization_id  TEXT NOT NULL,
  hive_id         TEXT NOT NULL,
  date            TIMESTAMP NOT NULL,
  weather           TEXT NOT NULL DEFAULT '',
  queen_seen BOOLEAN NOT NULL DEFAULT 0,
  eggs_seen     BOOLEAN NOT NULL DEFAULT 0,
  temperament         INTEGER NOT NULL DEFAULT 0,
  frames      INTEGER NOT NULL DEFAULT 0,
  stores           INTEGER NOT NULL DEFAULT 0,
  queen_cells     INTEGER NOT NULL DEFAULT 0,
  varroa    TEXT NOT NULL DEFAULT '',
  honey_kg         REAL NOT NULL DEFAULT 0,
  note            TEXT NOT NULL DEFAULT '',
  photo_keys        TEXT NOT NULL DEFAULT '',   -- JSON-Array
  created_at       TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_inspection_hive ON inspection (hive_id);

CREATE TABLE IF NOT EXISTS task (
  id              TEXT PRIMARY KEY,
  organization_id TEXT NOT NULL,
  title           TEXT NOT NULL,
  hive_id        TEXT,
  apiary_id     TEXT,
  due_at      TIMESTAMP,
  done        BOOLEAN NOT NULL DEFAULT 0,
  priority      INTEGER NOT NULL DEFAULT 2,
  note           TEXT NOT NULL DEFAULT '',
  recurrence    TEXT NOT NULL DEFAULT '',
  assigned_to   TEXT NOT NULL DEFAULT '',
  created_at      TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_task_org ON task (organization_id);

CREATE TABLE IF NOT EXISTS treatment (
  id              TEXT PRIMARY KEY,
  organization_id TEXT NOT NULL,
  hive_id        TEXT NOT NULL,
  date           TIMESTAMP NOT NULL,
  product          TEXT NOT NULL DEFAULT '',
  batch          TEXT NOT NULL DEFAULT '',
  dosage       TEXT NOT NULL DEFAULT '',
  withdrawal_days  INTEGER NOT NULL DEFAULT 0,
  applied_by        TEXT NOT NULL DEFAULT '',
  note           TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS harvest (
  id              TEXT PRIMARY KEY,
  organization_id TEXT NOT NULL,
  apiary_id     TEXT NOT NULL,
  date           TIMESTAMP NOT NULL,
  variety           TEXT NOT NULL DEFAULT '',
  amount_kg        REAL NOT NULL DEFAULT 0,
  water_content    REAL NOT NULL DEFAULT 0,
  batch_number   TEXT NOT NULL DEFAULT '',
  best_before             TIMESTAMP,
  note           TEXT NOT NULL DEFAULT ''
);
