// The sync engine. Runs in the background: pushes the outbox up (Push),
// pulls foreign changes (Pull) and applies them via last-writer-wins.
// Works offline (queued) and syncs automatically once connected.

import { getDB } from './db';
import { hlc } from './hlc';
import { parseFieldClock, accept, parseORSet, orAdd, orRemove } from './merge';
import { bumpData } from './live';
import { syncClient } from '$lib/client'; // generated Connect client (SyncService)

const SYNCED_TABLES = ['apiary', 'hive', 'queen', 'inspection', 'task', 'placement', 'harvest', 'event'];
const SET_COLS: Record<string, string[]> = { inspection: ['photo_keys'] };

async function getCursor(): Promise<string> {
  const db = await getDB();
  const r = await db.get<{ v: string }>(`SELECT v FROM sync_meta WHERE k = 'cursor'`);
  return r?.v ?? '';
}
async function setCursor(c: string) {
  const db = await getDB();
  await db.exec(
    `INSERT INTO sync_meta (k, v) VALUES ('cursor', ?)
     ON CONFLICT(k) DO UPDATE SET v = excluded.v`, [c]);
}

// --- Push: local outbox to the server ---
export async function push() {
  const db = await getDB();
  const pending = await db.all<any>(
    `SELECT * FROM outbox ORDER BY hlc ASC LIMIT 200`);
  if (pending.length === 0) return;

  const changes = pending.map((p) => ({
    entity: p.entity, entityId: p.entity_id, scopeId: p.scope_id,
    op: p.op, payloadJson: p.payload, hlc: p.hlc, authorId: p.author_id
  }));

  const res = await syncClient.push({ changes });

  // Remove successfully transmitted entries from the outbox.
  for (const p of pending) {
    await db.exec(`DELETE FROM outbox WHERE id = ?`, [p.id]);
  }
  // Conflicts: the server has a newer version -> it comes back via Pull anyway.
  if (res.serverCursor) await setCursor(res.serverCursor);
}

// --- Pull: apply remote changes ---
export async function pull() {
  const db = await getDB();
  let cursor = await getCursor();
  let hasMore = true;

  let applied = 0;
  while (hasMore) {
    const res = await syncClient.pull({ cursor, limit: 200 });

    for (const ch of res.changes) {
      hlc.recv(ch.hlc);
      await applyRemote(ch);
      applied++;
    }
    cursor = res.nextCursor;
    await setCursor(cursor);
    hasMore = res.hasMore;
  }
  // Notify open pages so their queries re-run with the freshly pulled data.
  if (applied > 0) bumpData();
}

// Per-field LWW + OR-Set: only newer fields are applied, set fields
// per union gemerged.
async function applyRemote(ch: any) {
  if (!SYNCED_TABLES.includes(ch.entity)) return;
  const db = await getDB();
  const cur = await db.get<any>(`SELECT * FROM ${ch.entity} WHERE id = ?`, [ch.entityId]);
  const fields = ch.op === 2 /* delete */ ? { deleted: 1 } : JSON.parse(ch.payloadJson);
  const setCols = SET_COLS[ch.entity] ?? [];

  if (!cur) {
    const fc: Record<string, string> = {};
    const cols = ['id'];
    const vals: unknown[] = [ch.entityId];
    for (const [k, v] of Object.entries(fields) as [string, any][]) {
      if (k === 'id') continue;
      if (setCols.includes(k)) {
        const os = {};
        for (const e of v?.add ?? []) orAdd(os, e, ch.hlc);
        cols.push(k); vals.push(JSON.stringify(os));
      } else {
        cols.push(k); vals.push(v); fc[k] = ch.hlc;
      }
    }
    cols.push('field_hlc'); vals.push(JSON.stringify(fc));
    const ph = cols.map(() => '?').join(', ');
    await db.exec(`INSERT INTO ${ch.entity} (${cols.join(', ')}) VALUES (${ph})`, vals);
    return;
  }

  const fc = parseFieldClock(cur.field_hlc);
  const sets: string[] = [];
  const args: unknown[] = [];
  for (const [k, v] of Object.entries(fields) as [string, any][]) {
    if (k === 'id' || k === 'field_hlc') continue;
    if (setCols.includes(k)) {
      const os = parseORSet(cur[k]);
      for (const e of v?.add ?? []) orAdd(os, e, ch.hlc);
      for (const e of v?.remove ?? []) orRemove(os, e);
      sets.push(`${k} = ?`); args.push(JSON.stringify(os));
    } else if (accept(fc, k, ch.hlc)) {
      sets.push(`${k} = ?`); args.push(v);
    }
  }
  if (sets.length === 0) return; // everything stale
  sets.push(`field_hlc = ?`); args.push(JSON.stringify(fc));
  args.push(ch.entityId);
  await db.exec(`UPDATE ${ch.entity} SET ${sets.join(', ')} WHERE id = ?`, args);
}

// --- Orchestration ---
let timer: ReturnType<typeof setInterval> | null = null;
let running = false;
let rerun = false;

// Single-flight: the UI fires syncOnce() after every local write, so many calls
// can overlap. Running them concurrently would open competing SQLite write
// transactions on the server (-> "database is locked"). Serialize instead, and
// if new changes arrived while a run was in flight, loop once more so the latest
// outbox entries are pushed without waiting for the periodic timer.
export async function syncOnce() {
  if (!navigator.onLine) return;
  if (running) { rerun = true; return; }
  running = true;
  try {
    do {
      rerun = false;
      await push();
      await pull();
    } while (rerun);
  } catch (e) {
    console.warn('sync deferred:', e);
  } finally {
    running = false;
  }
}

export function startSync() {
  void syncOnce();
  timer = setInterval(syncOnce, 15_000);          // periodic Fallback
  window.addEventListener('online', () => void syncOnce()); // immediately on reconnect
  // Optional: syncClient.subscribe({cursor}) for a real-time "poke".
}

export function stopSync() {
  if (timer) clearInterval(timer);
}
