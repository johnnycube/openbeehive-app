// Pure client-side SPA (no SSR/prerender): the app is local-first and runs
// SQLite-WASM in the browser. This makes the whole app embeddable as static
// files in the Go binary.
export const ssr = false;
export const prerender = false;
