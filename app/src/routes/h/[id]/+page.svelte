<script lang="ts">
  // Scan target. A hive QR points here (`/h/<id>`). We resolve the hive from
  // the local DB and redirect into the app. If it isn't local yet (e.g. just
  // shared), we trigger a sync and retry; if still missing, we explain why.
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { hives } from '$lib/local/repo';
  import { syncOnce } from '$lib/local/sync';

  let state = $state<'resolving' | 'missing' | 'offline'>('resolving');
  const id = $page.params.id ?? '';

  onMount(async () => {
    try {
      let hive = await hives.get(id);
      if (!hive && navigator.onLine) {
        await syncOnce();          // maybe it's shared but not pulled yet
        hive = await hives.get(id);
      }
      if (hive) {
        await goto(`/hives/${id}`, { replaceState: true });
        return;
      }
      state = navigator.onLine ? 'missing' : 'offline';
    } catch {
      state = 'missing';
    }
  });
</script>

<div class="wrap">
  {#if state === 'resolving'}
    <div class="spin">⬡</div>
    <p>Beute wird geöffnet …</p>
  {:else if state === 'offline'}
    <h2>Offline</h2>
    <p>Diese Beute ist auf dem Gerät noch nicht vorhanden. Sobald wieder eine
       Verbindung besteht, wird sie synchronisiert.</p>
    <a class="btn" href="/">Zur Übersicht</a>
  {:else}
    <h2>Nicht gefunden</h2>
    <p>Diese Beute gibt es nicht oder sie wurde nicht mit dir geteilt.</p>
    <a class="btn" href="/">Zur Übersicht</a>
  {/if}
</div>

<style>
  .wrap { min-height: 100vh; display: grid; place-content: center; gap: 12px;
    text-align: center; padding: 24px; color: var(--ink, #2c2316); }
  .spin { font-size: 40px; color: var(--honey, #c77f22); animation: s 1.4s linear infinite; }
  @keyframes s { to { transform: rotate(360deg); } }
  h2 { font-family: 'Fraunces', serif; }
  p { color: var(--ink-soft, #6b5e48); max-width: 36ch; }
  .btn { margin-top: 8px; background: var(--honey, #c77f22); color: #fff;
    padding: 11px 20px; border-radius: 11px; text-decoration: none; font-weight: 700; }
</style>
