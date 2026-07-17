// A tiny reactivity signal that bridges the (plain-TS) sync engine to Svelte
// pages. The engine bumps it after a pull applies remote changes; pages read it
// inside an $effect so their queries re-run when fresh data lands — e.g. on the
// first app load (local DB empty until the initial pull) or live multi-device
// sync. Local writes already reload their own page, so only remote applies bump.
import { writable } from 'svelte/store';

export const dataVersion = writable(0);

export function bumpData() {
  dataVersion.update((n) => n + 1);
}
