// Local database on the device. Web: SQLite-WASM on OPFS (persistent).
// Native (Capacitor): same interface, different implementation.
//
// The UI never talks to this class directly, only via lib/local/repo.ts.

export interface LocalDB {
  exec(sql: string, params?: unknown[]): Promise<void>;
  all<T = Record<string, unknown>>(sql: string, params?: unknown[]): Promise<T[]>;
  get<T = Record<string, unknown>>(sql: string, params?: unknown[]): Promise<T | undefined>;
}

// --- Web implementation: SQLite-WASM runs in a dedicated worker (db.worker.ts)
// because OPFS sync-access handles are only available off the main thread.
// This module proxies exec/all/get to that worker over postMessage. ---
import DbWorker from './db.worker?worker';

let dbPromise: Promise<LocalDB> | null = null;

export function getDB(): Promise<LocalDB> {
  if (!dbPromise) dbPromise = init();
  return dbPromise;
}

async function init(): Promise<LocalDB> {
  const worker = new DbWorker();
  let seq = 0;
  const pending = new Map<number, { resolve: (v: any) => void; reject: (e: Error) => void }>();

  worker.onmessage = (e: MessageEvent) => {
    const { id, ok, rows, error } = e.data;
    const p = pending.get(id);
    if (!p) return;
    pending.delete(id);
    ok ? p.resolve(rows) : p.reject(new Error(error));
  };

  function call(method: 'open' | 'exec' | 'all', sql: string, params: unknown[], name?: string): Promise<any> {
    return new Promise((resolve, reject) => {
      const id = ++seq;
      pending.set(id, { resolve, reject });
      worker.postMessage({ id, method, sql, params, name });
    });
  }

  // Per-tenant local store: open a database file keyed to the active tenant so
  // switching tenants never mixes data on the same device. Falls back to the
  // single-user "local" store when no tenant is set.
  const orgId = (typeof localStorage !== 'undefined' && localStorage.getItem('obh.orgId')) || 'local';
  const dbFile = `openbeehive-${orgId.replace(/[^a-zA-Z0-9_-]/g, '_')}.sqlite3`;
  await call('open', '', [], dbFile);

  const impl: LocalDB = {
    async exec(sql, params = []) {
      await call('exec', sql, params);
    },
    async all(sql, params = []) {
      return (await call('all', sql, params)) ?? [];
    },
    async get(sql, params = []) {
      const rows = (await impl.all(sql, params)) as any[];
      return rows[0];
    }
  };

  await migrateLocal(impl);
  return impl;
}

// Locals Spiegel-Schema. field_hlc = field-clock (Per-field-LWW),
// photo_keys holds OR-Set JSON. outbox = changes not yet pushed.
async function migrateLocal(db: LocalDB) {
  await db.exec(`
    CREATE TABLE IF NOT EXISTS apiary (
      id TEXT PRIMARY KEY, organization_id TEXT, name TEXT, address TEXT,
      lat REAL, lng REAL, note TEXT, created_at TEXT, updated_at TEXT,
      field_hlc TEXT DEFAULT '{}', deleted INTEGER DEFAULT 0);
    CREATE TABLE IF NOT EXISTS hive (
      id TEXT PRIMARY KEY, organization_id TEXT, apiary_id TEXT, name TEXT,
      type INTEGER, status INTEGER, boxes INTEGER, colony_origin TEXT, note TEXT,
      qr_code TEXT, photo TEXT, created_at TEXT, updated_at TEXT, field_hlc TEXT DEFAULT '{}', deleted INTEGER DEFAULT 0);
    CREATE TABLE IF NOT EXISTS queen (
      id TEXT PRIMARY KEY, organization_id TEXT, hive_id TEXT, year INTEGER,
      marking INTEGER, origin TEXT, breeder_number TEXT, introduced_at TEXT,
      replaced_at TEXT, active INTEGER, note TEXT, created_at TEXT, updated_at TEXT,
      field_hlc TEXT DEFAULT '{}', deleted INTEGER DEFAULT 0);
    CREATE TABLE IF NOT EXISTS inspection (
      id TEXT PRIMARY KEY, organization_id TEXT, hive_id TEXT, date TEXT,
      weather TEXT, queen_seen INTEGER, eggs_seen INTEGER, temperament INTEGER,
      frames INTEGER, stores INTEGER, queen_cells INTEGER, varroa TEXT,
      honey_kg REAL, note TEXT,
      -- Stockkarte fields: colony, behaviour and the activities done on the visit.
      brood_frames INTEGER, calmness INTEGER, fed_kg REAL,
      frames_added INTEGER, frames_removed INTEGER, drone_frame_cut INTEGER,
      super_added INTEGER, weight_kg REAL,
      youngest_larva INTEGER, covered_larva INTEGER,
      -- Climate readings: temperature (°C) and humidity (%), inside the hive and outside.
      temp_hive REAL, temp_outside REAL, humidity_hive REAL, humidity_outside REAL,
      photo_keys TEXT DEFAULT '{}', created_at TEXT, field_hlc TEXT DEFAULT '{}', deleted INTEGER DEFAULT 0);
    CREATE TABLE IF NOT EXISTS task (
      id TEXT PRIMARY KEY, organization_id TEXT, title TEXT, hive_id TEXT,
      apiary_id TEXT, due_at TEXT, done INTEGER, priority INTEGER,
      note TEXT, recurrence TEXT, assigned_to TEXT, created_at TEXT,
      field_hlc TEXT DEFAULT '{}', deleted INTEGER DEFAULT 0);

    -- history / fact table
    CREATE TABLE IF NOT EXISTS placement (
      id TEXT PRIMARY KEY, organization_id TEXT, hive_id TEXT, apiary_id TEXT,
      start_at TEXT, end_at TEXT, field_hlc TEXT DEFAULT '{}', deleted INTEGER DEFAULT 0);
    CREATE TABLE IF NOT EXISTS harvest (
      id TEXT PRIMARY KEY, organization_id TEXT, apiary_id TEXT, hive_id TEXT,
      queen_id TEXT, date TEXT, variety TEXT, amount_kg REAL, water_content REAL,
      batch_number TEXT, best_before TEXT, note TEXT, field_hlc TEXT DEFAULT '{}', deleted INTEGER DEFAULT 0);
    -- Treatments (Bestandsbuch): varroa / disease treatments, audit-ready.
    CREATE TABLE IF NOT EXISTS treatment (
      id TEXT PRIMARY KEY, organization_id TEXT, apiary_id TEXT, hive_id TEXT, queen_id TEXT,
      date TEXT, product TEXT, active_ingredient TEXT, dose TEXT, method TEXT,
      batch_number TEXT, withdrawal_until TEXT, reason TEXT, note TEXT,
      field_hlc TEXT DEFAULT '{}', deleted INTEGER DEFAULT 0);
    CREATE TABLE IF NOT EXISTS event (
      id TEXT PRIMARY KEY, organization_id TEXT, scope_id TEXT, type INTEGER, date TEXT,
      apiary_id TEXT, hive_id TEXT, queen_id TEXT, ref_entity TEXT, ref_id TEXT,
      title TEXT, amount_kg REAL DEFAULT 0, detail TEXT, author_id TEXT,
      field_hlc TEXT DEFAULT '{}', deleted INTEGER DEFAULT 0);
    CREATE INDEX IF NOT EXISTS idx_event_hive ON event (hive_id, date);
    CREATE INDEX IF NOT EXISTS idx_event_apiary ON event (apiary_id, date);

    -- Outbox: local changes not yet pushed (partial field deltas).
    CREATE TABLE IF NOT EXISTS outbox (
      id TEXT PRIMARY KEY, entity TEXT, entity_id TEXT, scope_id TEXT,
      op INTEGER, payload TEXT, hlc TEXT, author_id TEXT);

    CREATE TABLE IF NOT EXISTS sync_meta (k TEXT PRIMARY KEY, v TEXT);
  `);

  // Schema evolution for databases created by an earlier app version:
  // CREATE TABLE IF NOT EXISTS never adds columns to an existing table, so
  // reconcile any columns added after the initial schema. Without this, writes
  // that reference a new column fail on an old local DB (e.g. saving a visit).
  await ensureColumns(db, 'hive', [['photo', 'TEXT']]);
  await ensureColumns(db, 'inspection', [
    ['brood_frames', 'INTEGER'], ['calmness', 'INTEGER'], ['fed_kg', 'REAL'],
    ['frames_added', 'INTEGER'], ['frames_removed', 'INTEGER'], ['drone_frame_cut', 'INTEGER'],
    ['super_added', 'INTEGER'], ['weight_kg', 'REAL'],
    ['youngest_larva', 'INTEGER'], ['covered_larva', 'INTEGER'],
    ['temp_hive', 'REAL'], ['temp_outside', 'REAL'],
    ['humidity_hive', 'REAL'], ['humidity_outside', 'REAL']
  ]);
}

// Add any missing columns to an existing table (idempotent; SQLite ALTER TABLE
// ADD COLUMN errors if the column already exists, so we check first).
async function ensureColumns(db: LocalDB, table: string, cols: [string, string][]) {
  const info = await db.all<{ name: string }>(`PRAGMA table_info(${table})`);
  const existing = new Set(info.map((r) => r.name));
  for (const [name, type] of cols) {
    if (!existing.has(name)) await db.exec(`ALTER TABLE ${table} ADD COLUMN ${name} ${type}`);
  }
}
