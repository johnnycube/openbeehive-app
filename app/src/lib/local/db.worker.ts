// Dedicated SQLite-WASM worker. OPFS sync-access handles are only available in
// a Worker context, so the database lives here and the main thread proxies
// queries via postMessage (see db.ts).
//
// The database file is per-tenant: the main thread sends an `open` message with
// the file name (derived from the active tenant) before any query, so each
// tenant's data lives in its own local store and switching tenants never mixes
// them on the same device.
import sqlite3InitModule from '@sqlite.org/sqlite-wasm';

type Req = { id: number; method: 'open' | 'exec' | 'all'; sql?: string; params?: unknown[]; name?: string };

let dbName = 'openbeehive.sqlite3';
let dbPromise: Promise<any> | null = null;

async function getDb() {
  if (!dbPromise) {
    dbPromise = (async () => {
      const sqlite3 = await sqlite3InitModule();
      // OPFS SAHPool VFS: persistent, no SharedArrayBuffer / cross-origin
      // isolation required — works behind any reverse proxy.
      const pool = await sqlite3.installOpfsSAHPoolVfs({ name: 'openbeehive' });
      return new pool.OpfsSAHPoolDb('/' + dbName);
    })();
  }
  return dbPromise;
}

self.onmessage = async (e: MessageEvent<Req>) => {
  const { id, method, sql, params, name } = e.data;
  try {
    if (method === 'open') {
      if (name) dbName = name; // must arrive before the first query
      (self as unknown as Worker).postMessage({ id, ok: true });
      return;
    }
    const db = await getDb();
    if (method === 'all') {
      const rows = db.exec({ sql, bind: params, rowMode: 'object', returnValue: 'resultRows' });
      (self as unknown as Worker).postMessage({ id, ok: true, rows });
    } else {
      db.exec({ sql, bind: params });
      (self as unknown as Worker).postMessage({ id, ok: true });
    }
  } catch (err) {
    (self as unknown as Worker).postMessage({ id, ok: false, error: String((err as Error)?.message ?? err) });
  }
};
