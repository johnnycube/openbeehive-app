import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

// SQLite-WASM on OPFS needs a cross-origin-isolated context (SharedArrayBuffer).
// COEP=credentialless keeps cross-origin assets like Google Fonts working.
const crossOriginIsolation = {
  'Cross-Origin-Opener-Policy': 'same-origin',
  'Cross-Origin-Embedder-Policy': 'credentialless'
};

export default defineConfig({
  plugins: [sveltekit()],
  // Client-exposed env vars are namespaced BEEHIVE_ (VITE_ kept for compatibility).
  envPrefix: ['BEEHIVE_', 'VITE_'],
  server: { port: 5173, headers: crossOriginIsolation },
  preview: { headers: crossOriginIsolation },
  // Don't pre-bundle sqlite-wasm: the optimizer rewrites its path so the
  // sibling sqlite3.wasm can no longer be fetched (404). Excluding it keeps
  // the wasm resolvable relative to the package.
  optimizeDeps: { exclude: ['@sqlite.org/sqlite-wasm'] },
  worker: { format: 'es' }
});
