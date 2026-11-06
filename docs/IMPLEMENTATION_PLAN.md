# Implementation Plan

Build vertically: each milestone should produce something executable and testable.

## Phase 0′ — Migrate to Go + Wails

**Status:** completed foundation after Rust + Tauri Phase 0 migration.

**Deliverables**

- Go module `github.com/lostfromlight1/Open-Download-Manager`
- Wails v3.1.0 desktop shell with React + TypeScript frontend
- `odm` CLI stub
- CI for Go and frontend
- Docs and Cursor rules updated for Go + Wails
- Network Access / Tunneling documented as a separate track

**Exit criteria**

- Fresh clone can follow README setup
- Frontend checks pass
- `go test ./...` passes after frontend assets exist for embed
- Desktop shell launches
- No Rust/Tauri backend remains
- Download engine is not implemented yet

## Phase 1 — Single-stream CLI engine

**Status:** implemented (CLI + desktop share the engine).

**Deliverables**

- `internal/engine` single-stream HTTP/HTTPS download
- `internal/netx` HTTP client with system / manual / off proxy modes
- `internal/testserver` local flaky HTTP fixture
- `odm download` command with progress on stderr
- Streaming writes to a `.part` file; rename on success
- Resume via HTTP Range when the server returns a valid 206
- Retry with backoff, stall detection, connect/header timeouts

**Acceptance tests** (covered under `internal/engine`)

- Small file download
- Mid-body disconnect recovery
- Stall detection
- 429 and 5xx retries
- Redirect handling
- Servers without Range support / `--force-restart`
- Destination path failure
- Invalid URL rejection

## Phase 2 — Persistence, queue, user decisions

**Status:** implemented (CLI + desktop share the manager/store).

**Deliverables**

- `internal/task` explicit state machine
- `internal/storage` SQLite schema + versioned SQL migrations
- `internal/manager` queue with `queue.max_active` concurrency
- Cancel/remove `--files keep|delete` choices
- Resume validator mismatch via `--on-mismatch abort|restart`
- Persisted proxy + download settings (`odm settings`)

## Phase 3 — Segmented downloads

**Status:** implemented (engine + CLI/manager + desktop settings).

**Deliverables**

- Range capability verification (`VerifyRangeSupport` — real HTTP 206 + Content-Range, not `Accept-Ranges`)
- Segment planner (`PlanSegments`) and bounded workers (`Connections`)
- Random-access `WriteAt` writes without overlap; preallocated `.part` cleared on failure
- Safe single-stream fallback when Range is unverified, the file is tiny, or a contiguous `.part` already exists
- SQLite `segments` table (migration 002) + `download.max_connections` setting
- CLI: `odm download --connections N`

**Acceptance tests** (under `internal/engine`, `internal/storage`)

- Segmented happy path with multiple Range GETs
- Fallback when server ignores Range / returns bad Content-Range
- Planner non-overlap / coverage unit tests
- Segment row round-trip in SQLite

## Phase 4 — Desktop MVP

**Status:** implemented + polished (theme, opt-in completion notifications, download details / open / reveal).

**Deliverables**

- Dashboard and add-download flow (`apps/desktop` DownloadService + React UI)
- Real engine progress events (`odm:download-progress` / `odm:download-changed`)
- Settings for destination, proxy, concurrency, connections, theme, and notify-on-complete
- Cancel/remove keep-or-delete dialogs and resume-mismatch decision UI
- Download detail dialog with Open file / Show in folder (paths from task rows only)
- Shared SQLite data dir with the CLI via `internal/appdir`

## Phase 5 — Cross-platform hardening

**Status:** implemented (unsigned packaging + CI; signing/notarization deferred to v1.0).

**Deliverables**

- Aligned product branding/version **0.4.0** across Wails config, Windows `info.json`/NSIS, Darwin plists, Linux nfpm/desktop metadata, and `internal/version`
- Tested open-file / reveal-in-folder command planning (`apps/desktop` openpath helpers)
- Windows: `CI=true wails3 build` + NSIS installer (`wails3 task windows:package` / `windows:create:nsis:installer`) → `apps/desktop/bin/odm-amd64-installer.exe`
- macOS: unsigned `.app` via `wails3 task darwin:package`
- Linux: unsigned `.deb` via `wails3 task linux:create:deb` (AppImage/MSIX/signing out of scope)
- CI job `desktop-package` builds and uploads artifacts on Windows, macOS, and Ubuntu (7-day retention)
- SETUP / TESTING / README packaging commands and smoke checklists

**Exit criteria**

- Windows NSIS installer builds locally and in CI
- macOS `.app` and Linux `.deb` produced in CI
- Branding/version metadata no longer placeholders
- Open/reveal helpers covered by unit tests
- Docs updated; code signing / notarization explicitly deferred

## Phase 6 — Browser integration

**Status:** implemented (opt-in WebExtension + native messaging; no download interception).

**Deliverables**

- Setting `browser.integration_enabled` (default off) + optional `browser.chrome_extension_id`
- `internal/browserhost` protocol (ping/enqueue), NM IO, filename sanitization, host install/uninstall
- `cmd/odm-nmhost` enqueues into the shared SQLite store (does not run `RunQueue`)
- Desktop Settings toggle registers the host; reconcile loop surfaces NM-enqueued tasks in the UI
- `apps/browser-extension` MV3 for Chrome (`manifest.json`) and Firefox (`manifest.firefox.json`): context menu, popup status, optional click capture

**Exit criteria**

- Enqueue rejected when integration is disabled
- Context-menu send creates a queued task when enabled + download directory set
- Protocol/settings tests cover reject and happy paths
- No history/page/cookie collection; download interception remains later

## Phase 7 — Background service

Reuse the same engine so downloads continue with the UI closed.

## Phase N — Network Access / Tunneling track

**After** reliable downloading and proxy support.

**Deliverables**

- AccessProvider interface implementation spike
- Opt-in settings and user-configurable endpoints
- Threat model, limitation docs, security review gate
- Isolation tests proving downloads work with access disabled

Do not block Phases 1–2 on this track.
