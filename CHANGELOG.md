# Changelog

All notable changes to the Openbeehive application are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[0.1.0]: https://github.com/johnnycube/openbeehive-app/releases/tag/v0.1.0
