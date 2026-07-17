import type { Handle } from '@sveltejs/kit';

// SQLite-WASM on OPFS requires a cross-origin-isolated context
// (SharedArrayBuffer). COEP=credentialless keeps cross-origin assets such as
// Google Fonts loadable without an explicit CORP header. These must be set by
// any reverse proxy in front of the production server as well.
export const handle: Handle = async ({ event, resolve }) => {
  const response = await resolve(event);
  response.headers.set('Cross-Origin-Opener-Policy', 'same-origin');
  response.headers.set('Cross-Origin-Embedder-Policy', 'credentialless');
  return response;
};
