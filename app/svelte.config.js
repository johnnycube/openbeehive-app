import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

// Static SPA: the app is a pure client-side single-page app (local-first,
// SQLite-WASM). It builds to ./build and is embedded into the Go binary for
// single-binary production. The fallback serves index.html for every route so
// client-side routing (/, /apiaries, /hives/<id>, …) works.
export default {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({ fallback: 'index.html', pages: 'build', assets: 'build', strict: false })
  }
};
