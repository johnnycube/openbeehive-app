// Tiny global toast stack, used for errors that would otherwise be invisible
// (e.g. a local write failing after its form already validated). Push with
// toast('message'); the layout renders <Toasts/> once for the whole app.
import { writable } from 'svelte/store';

export type Toast = { id: number; text: string };

export const toasts = writable<Toast[]>([]);
let nextId = 1;

export function toast(text: string, ttlMs = 8000) {
  const id = nextId++;
  toasts.update((ts) => [...ts, { id, text }]);
  setTimeout(() => dismiss(id), ttlMs);
}

export function dismiss(id: number) {
  toasts.update((ts) => ts.filter((t) => t.id !== id));
}
