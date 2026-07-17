<script lang="ts">
  import { onMount } from 'svelte';
  import { _, locale } from 'svelte-i18n';
  import { changeLanguage, SUPPORTED, LANG_LABELS, type Lang } from '$lib/i18n';
  import {
    exportJSON, exportXLSX, exportCSVZip, exportBeeXML, printReport, importFile, type ImportMode
  } from '$lib/data/portability';

  const API = (import.meta.env.BEEHIVE_API_URL ?? '').replace(/\/$/, '');

  // Legal pages live on the separate marketing site (e.g. openbeehive.org).
  // Link to them only when its base URL is configured (central deployments).
  const marketingUrl = (import.meta.env.BEEHIVE_MARKETING_URL ?? '').replace(/\/$/, '');
  const showLegal = !!marketingUrl;

  const LABELS = LANG_LABELS;

  let email = $state('');
  $effect(() => { email = localStorage.getItem('obh.email') ?? 'local@openbeehive'; });

  function pick(l: Lang) { changeLanguage(l); }

  // --- Tenants (multi-tenant: switch / create / invite) ---
  type Tenant = { id: string; name: string; role: string };
  let tenants = $state<Tenant[]>([]);
  let activeOrg = $state('');
  let newTenant = $state('');
  let inviteEmail = $state('');
  let tenantMsg = $state('');
  let signedIn = $state(false);
  const activeTenant = $derived(tenants.find((t) => t.id === activeOrg));
  const isOwner = $derived(activeTenant?.role === 'owner');

  const authHeaders = (): Record<string, string> => {
    const h: Record<string, string> = { 'Content-Type': 'application/json' };
    const t = localStorage.getItem('session');
    if (t) h.Authorization = `Bearer ${t}`;
    return h;
  };

  async function loadTenants() {
    try {
      const r = await fetch(`${API}/auth/me`, { credentials: 'include', headers: authHeaders() });
      signedIn = r.ok;
      if (!r.ok) return;
      const j = await r.json();
      tenants = j.tenants ?? [];
      activeOrg = j.active_org ?? '';
    } catch { /* single-user mode: no tenants */ }
  }
  onMount(loadTenants);

  async function logout() {
    try {
      await fetch(`${API}/auth/logout`, { method: 'POST', credentials: 'include', headers: authHeaders() });
    } catch { /* clear client state regardless */ }
    for (const k of ['session', 'obh.userId', 'obh.orgId', 'obh.email']) localStorage.removeItem(k);
    location.href = '/login';
  }

  // Persist the new session + active tenant, then reload so the per-tenant local
  // store opens and the sync client re-authenticates for the new tenant.
  function applyTenant(j: any) {
    if (j?.token) localStorage.setItem('session', j.token);
    if (j?.active_org) localStorage.setItem('obh.orgId', j.active_org);
    location.reload();
  }

  async function switchTenant(id: string) {
    if (id === activeOrg) return;
    const r = await fetch(`${API}/auth/switch`, {
      method: 'POST', credentials: 'include',
      headers: authHeaders(), body: JSON.stringify({ org_id: id })
    });
    applyTenant(await r.json().catch(() => ({})));
  }

  async function createTenant() {
    const name = newTenant.trim();
    if (!name) return;
    const r = await fetch(`${API}/tenants/create`, {
      method: 'POST', credentials: 'include',
      headers: authHeaders(), body: JSON.stringify({ name })
    });
    applyTenant(await r.json().catch(() => ({})));
  }

  async function sendInvite() {
    const email2 = inviteEmail.trim();
    if (!email2) return;
    const r = await fetch(`${API}/tenants/invite`, {
      method: 'POST', credentials: 'include',
      headers: authHeaders(), body: JSON.stringify({ email: email2, org_id: activeOrg })
    });
    if (r.ok) { tenantMsg = $_('tenant.invite_sent'); inviteEmail = ''; }
  }

  // --- Data export / import ---
  let busy = $state(false);
  let importMode = $state<ImportMode>('auto');
  let importMsg = $state('');
  let importErr = $state(false);

  async function runExport(fn: () => Promise<void>) {
    if (busy) return;
    busy = true; importMsg = ''; importErr = false;
    try { await fn(); } catch (e) { importErr = true; importMsg = String((e as Error)?.message ?? e); }
    finally { busy = false; }
  }

  async function onImport(ev: Event) {
    const input = ev.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    busy = true; importMsg = $_('data.importing'); importErr = false;
    try {
      const res = await importFile(file, importMode);
      const total = Object.values(res).reduce((a, b) => a + b, 0);
      importMsg = $_('data.imported', { values: { n: total } });
    } catch (e) {
      importErr = true;
      importMsg = $_('data.import_failed', { values: { msg: String((e as Error)?.message ?? e) } });
    } finally {
      busy = false; input.value = '';
    }
  }
</script>

<div class="page">
  <h1>{$_('settings.title')}</h1>

  <section class="card">
    <h2>{$_('settings.language')}</h2>
    <p class="help">{$_('settings.language_help')}</p>
    <div class="langs">
      {#each SUPPORTED as l}
        <button class="lang" class:on={$locale?.startsWith(l)} onclick={() => pick(l)}>
          <span class="code">{l.toUpperCase()}</span>
          <span class="name">{LABELS[l]}</span>
          {#if $locale?.startsWith(l)}<span class="tick">✓</span>{/if}
        </button>
      {/each}
    </div>
  </section>

  {#if tenants.length}
    <section class="card">
      <h2>{$_('tenant.title')}</h2>
      <p class="help">{$_('tenant.help')}</p>
      <div class="tenants">
        {#each tenants as t}
          <button class="tenant" class:on={t.id === activeOrg} onclick={() => switchTenant(t.id)}>
            <span class="tname">{t.name}</span>
            <span class="trole">{t.role === 'owner' ? $_('tenant.role_owner') : $_('tenant.role_member')}</span>
            {#if t.id === activeOrg}<span class="ttick">✓ {$_('tenant.active')}</span>{/if}
          </button>
        {/each}
      </div>
      <div class="trow">
        <input bind:value={newTenant} placeholder={$_('tenant.create_ph')} />
        <button class="data-btn" onclick={createTenant}>+ {$_('tenant.create')}</button>
      </div>
      {#if isOwner}
        <h3 class="sub">{$_('tenant.invite')}</h3>
        <div class="trow">
          <input type="email" bind:value={inviteEmail} placeholder={$_('tenant.invite_ph')} />
          <button class="data-btn" onclick={sendInvite}>{$_('tenant.invite_send')}</button>
        </div>
        {#if tenantMsg}<p class="import-msg">{tenantMsg}</p>{/if}
      {/if}
    </section>
  {/if}

  <section class="card">
    <h2>{$_('data.title')}</h2>
    <p class="help">{$_('data.export_help')}</p>
    <div class="data-btns">
      <button class="data-btn" disabled={busy} onclick={() => runExport(exportJSON)}>⤓ {$_('data.json')}</button>
      <button class="data-btn" disabled={busy} onclick={() => runExport(exportXLSX)}>⤓ {$_('data.xlsx')}</button>
      <button class="data-btn" disabled={busy} onclick={() => runExport(exportCSVZip)}>⤓ {$_('data.csv')}</button>
      <button class="data-btn" disabled={busy} onclick={() => runExport(exportBeeXML)}>⤓ {$_('data.beexml')}</button>
      <button class="data-btn" disabled={busy} onclick={() => runExport(printReport)}>🖨 {$_('data.pdf')}</button>
    </div>

    <h3 class="sub">{$_('data.import')}</h3>
    <p class="help">{$_('data.import_help')}</p>
    <div class="import-row">
      <select bind:value={importMode} disabled={busy}>
        <option value="auto">{$_('data.mode_auto')}</option>
        <option value="json">{$_('data.mode_json')}</option>
        <option value="beexml">{$_('data.mode_beexml')}</option>
        <option value="csv">{$_('data.mode_csv')}</option>
      </select>
      <label class="data-btn file">
        {$_('data.import_btn')}
        <input type="file" accept=".json,.xml,.csv,.txt" hidden disabled={busy} onchange={onImport} />
      </label>
    </div>
    {#if importMsg}<p class="import-msg" class:err={importErr}>{importMsg}</p>{/if}
  </section>

  <section class="card">
    <h2>{$_('settings.account')}</h2>
    <p class="help">{$_('settings.user')}</p>
    <div class="user">
      <span class="ava">⬡</span>
      <strong>{email}</strong>
    </div>
    {#if signedIn}
      <button class="data-btn logout" onclick={logout}>{$_('settings.logout')}</button>
    {/if}
  </section>

  {#if showLegal}
    <section class="card">
      <h2>{$_('footer.legal')}</h2>
      <nav class="legal-links">
        <a href={`${marketingUrl}/about`}>{$_('footer.about')} ›</a>
        <a href={`${marketingUrl}/imprint`}>{$_('footer.imprint')} ›</a>
        <a href={`${marketingUrl}/privacy`}>{$_('footer.privacy')} ›</a>
      </nav>
    </section>
  {/if}
</div>

<style>
  .page { max-width: 640px; margin: 0 auto; padding: 26px 18px 96px; }
  h1 { font-family: 'Fraunces', serif; font-size: 1.9rem; margin-bottom: 18px; }
  h2 { font-family: 'Fraunces', serif; font-size: 1.2rem; margin-bottom: 6px; }
  .help { color: var(--ink-soft); font-size: .88rem; margin-bottom: 14px; }
  .card { background: var(--cream2); border: 1px solid var(--line); border-radius: 16px;
    padding: 20px; margin-bottom: 16px; }

  .langs { display: grid; gap: 10px; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); }
  .lang { display: flex; align-items: center; gap: 11px; padding: 13px 15px; cursor: pointer;
    border: 1px solid var(--line); border-radius: 13px; background: #fff; font: inherit; color: var(--ink);
    text-align: left; }
  .lang.on { border-color: var(--honey); background: rgba(199,127,34,.08); }
  .lang .code { font-weight: 800; font-size: .82rem; background: var(--cream); border: 1px solid var(--line);
    border-radius: 7px; padding: 3px 7px; color: var(--ink-soft); }
  .lang.on .code { background: var(--honey); color: #fff; border-color: var(--honey); }
  .lang .name { font-weight: 600; flex: 1; }
  .lang .tick { color: var(--honey-d); font-weight: 800; }

  .sub { font-family: 'Fraunces', serif; font-size: 1.02rem; margin: 20px 0 6px; }
  .data-btns { display: grid; gap: 10px; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); }
  .data-btn { display: inline-flex; align-items: center; justify-content: center; gap: 6px;
    padding: 12px 14px; cursor: pointer; border: 1px solid var(--line); border-radius: 12px;
    background: #fff; font: inherit; font-weight: 600; color: var(--ink); text-align: center; }
  .data-btn:hover:not(:disabled) { border-color: var(--honey); background: rgba(199,127,34,.08); }
  .data-btn:disabled { opacity: .55; cursor: default; }
  .import-row { display: flex; gap: 10px; flex-wrap: wrap; align-items: center; }
  .import-row select { padding: 11px 12px; border: 1px solid var(--line); border-radius: 12px;
    background: #fff; font: inherit; color: var(--ink); flex: 1; min-width: 150px; }
  .data-btn.file { flex-shrink: 0; }
  .import-msg { margin-top: 12px; font-size: .9rem; font-weight: 600; color: var(--moss); }
  .import-msg.err { color: #b5402f; }

  .tenants { display: grid; gap: 8px; margin-bottom: 12px; }
  .tenant { display: flex; align-items: center; gap: 10px; padding: 11px 14px; cursor: pointer;
    border: 1px solid var(--line); border-radius: 12px; background: #fff; font: inherit; color: var(--ink); text-align: left; }
  .tenant.on { border-color: var(--honey); background: rgba(199,127,34,.08); }
  .tenant .tname { font-weight: 700; flex: 1; }
  .tenant .trole { font-size: .72rem; font-weight: 700; color: var(--ink-soft); background: var(--cream);
    border: 1px solid var(--line); border-radius: 7px; padding: 2px 7px; }
  .tenant .ttick { color: var(--honey-d); font-weight: 700; font-size: .76rem; }
  .trow { display: flex; gap: 10px; flex-wrap: wrap; margin-top: 6px; }
  .trow input { flex: 1; min-width: 150px; padding: 11px 12px; border: 1px solid var(--line);
    border-radius: 12px; background: #fff; font: inherit; color: var(--ink); }
  .logout { margin-top: 14px; }
  .logout:hover { border-color: #b5402f; color: #b5402f; }

  .user { display: flex; align-items: center; gap: 12px; }
  .user .ava { font-size: 1.6rem; color: var(--honey); }
  .legal-links { display: flex; flex-direction: column; gap: 2px; }
  .legal-links a { padding: 11px 2px; text-decoration: none; color: var(--ink);
    font-weight: 600; border-top: 1px solid var(--line); }
  .legal-links a:first-child { border-top: none; }
  .legal-links a:hover { color: var(--honey-d); }
</style>
