// Central Connect-Web transport. The bearer token (app session) is attached per
// interceptor and attached to every RPC.
import { createClient, type Interceptor } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { browser } from '$app/environment';

// Generated via `make proto` into src/lib/proto (protoc-gen-es v2: services
// live in the *_pb modules):
import { ApiaryService } from './proto/openbeehive/v1/apiary_pb';
import { SyncService } from './proto/openbeehive/v1/sync_pb';
// import { HiveService } from './proto/openbeehive/v1/hive_pb';
// import { StatsService } from './proto/openbeehive/v1/stats_pb';

// Default to the current origin in the browser: the Go binary always serves the
// SPA and API on one origin (cloud behind a reverse proxy, or the single
// self-host binary), so same-origin is correct without a build-time override.
// Falls back to localhost only for non-browser/dev contexts.
const API_URL = import.meta.env.BEEHIVE_API_URL ?? (browser ? window.location.origin : 'http://localhost:8080');

const authInterceptor: Interceptor = (next) => async (req) => {
  if (browser) {
    const token = localStorage.getItem('session');
    if (token) req.header.set('Authorization', `Bearer ${token}`);
  }
  return next(req);
};

const transport = createConnectTransport({
  baseUrl: API_URL,
  interceptors: [authInterceptor]
});

export const apiaryClient = createClient(ApiaryService, transport);
export const syncClient = createClient(SyncService, transport);
// export const hiveClient = createClient(HiveService, transport);
// export const statsClient = createClient(StatsService, transport);
