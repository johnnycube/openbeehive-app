# Changelog

All notable changes to the Openbeehive application are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.2] - 2026-08-01

### Fixed

- **Saving could fail permanently with `SQLITE_CANTOPEN` ("unable to open
  database file")** — most visible on the demo as an add button that did
  nothing. Local OPFS storage pools created by earlier versions kept their
  small slot count forever; the app now reserves the minimum capacity on
  every start and grows the pool automatically if a write still runs out of
  slots.
- **Sync no longer stalls behind a failing push:** fresh data is pulled even
  while the outbox cannot be sent, and pushes rejected as read-only (the demo
  account, unwritable scopes) are dropped instead of retrying every 15
  seconds — the affected changes simply stay on the device.
- **Sync protocol, photo lists:** removes now carry the origin's observed
  add-tags, so a photo re-added concurrently on another device survives a
  remove (true add-wins). Deltas from older versions keep the previous
  behavior.
- **Failed local writes are visible:** the apiary form reports save errors
  inline and keeps the typed input; everywhere else an error toast appears
  instead of a silently dead button.

### Added

- **Interactive location picker for apiaries:** a map with a draggable pin
  that stays in sync with the address field (OpenStreetMap/Nominatim), with
  visible lookup states — searching, location set, no match, service
  unreachable.
- **Passkeys are bound to registered accounts:** enrollment lives in
  settings, sign-in offers the passkey only for accounts that registered one.
- **Test suites for the offline-first core.** Web (vitest on a real SQLite):
  per-field LWW and OR-Set merge semantics, HLC ordering, repo/outbox writes,
  sync error paths, geocoding states, and a two-device offline convergence
  simulation. Server (Go): merge mirrors of the client tests, read-only-guard
  coverage, and the OR-Set observed-tags round-trip through Push.

## [0.2.1] - 2026-07-31

### Added

- **Invite management** in settings: the invite link is now shown (with a copy
  button) right after creating it, all open invites are listed with their
  signup links, and invites can be revoked. New owner-only endpoints
  `GET /tenants/invites` and `POST /tenants/invite/revoke`.
- **Invite emails** — the invitee receives the signup link by email when SMTP
  is configured (shared mailer with the verification flow); without SMTP the
  link is logged as before and always shown in the UI.
- **Tenant deletion** (danger zone) — a tenant admin can permanently delete a
  tenant via `POST /tenants/delete`: all records, memberships, invites and the
  change log are removed in one transaction, inspection photo blobs are purged
  from the object store, and the session moves to a remaining tenant (or a
  fresh personal one). Confirmed in the UI, translated in all five languages.

### Security

- **FS blob store:** client-supplied blob keys (synced `photo_keys`) can no
  longer escape the storage base directory (path-traversal guard in the
  filesystem backend).

## [0.1.0] - 2026-07-17

First public release. 🐝

### Added

- **Offline-first PWA** (SvelteKit, Svelte 5) with a local SQLite-WASM store on
  OPFS — fully usable in the bee yard with or without a signal.
- **Domain model:** Apiaries, Hives, Queens, Inspections, Tasks, Events,
  Harvests and Treatments.
- **Per-hive records** (stock cards) with colony strength, brood, stores,
  varroa, behaviour and per-visit activities.
- **Queen management** with the international marking-colour scheme and full
  reign history.
- **Location history** — hives tracked across apiaries over time.
- **Honey harvests** with batch, water content and best-before, feeding season
  statistics.
- **QR codes** per hive — print labels and scan straight into a hive's record.
- **Sync engine** with HLC timestamps, per-field last-writer-wins and OR-Sets;
  apiary-level sharing via scopes.
- **Pluggable backends:** PostgreSQL · MySQL · SQLite, and MinIO/S3 ·
  filesystem, selected via `DEPLOYMENT_PROFILE` (`cloud` | `selfhost`).
- **Connect-RPC API** (gRPC + HTTP/JSON) with the `.proto` contract as the
  source of truth.
- **OIDC authentication** with multiple providers.
- **Email/password onboarding** with optional email verification, multi-tenant
  invites, and an **invite-only mode** (`BEEHIVE_REGISTRATION=false`) that
  disables open registration and shows a notice on the sign-in screen.
- **Single-binary self-hosting** (SQLite + filesystem, no Docker required) and a
  Docker image for the cloud profile.
- **Multi-language UI:** English, German, French, Spanish, Italian.

[0.2.2]: https://github.com/johnnycube/openbeehive-app/releases/tag/v0.2.2
[0.2.1]: https://github.com/johnnycube/openbeehive-app/releases/tag/v0.2.1
[0.1.0]: https://github.com/johnnycube/openbeehive-app/releases/tag/v0.1.0
