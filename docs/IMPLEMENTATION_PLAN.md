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

**Status:** implemented for the `odm` CLI (desktop UI download wiring is still Phase 4).

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

**Status:** implemented for the `odm` CLI manager (desktop UI wiring remains Phase 4).

**Deliverables**

- `internal/task` explicit state machine
- `internal/storage` SQLite schema + versioned SQL migrations
- `internal/manager` queue with `queue.max_active` concurrency
- Cancel/remove `--files keep|delete` choices
- Resume validator mismatch via `--on-mismatch abort|restart`
- Persisted proxy + download settings (`odm settings`)

## Phase 3 — Segmented downloads

**Status:** implemented in the engine + CLI/manager settings (desktop UI wiring remains Phase 4).

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

**Status:** implemented (Option 1 scope). Theme selector, native notifications, and detail inspector remain deferred.

**Deliverables**

- Dashboard and add-download flow (`apps/desktop` DownloadService + React UI)
- Real engine progress events (`odm:download-progress` / `odm:download-changed`)
- Settings for destination, proxy, concurrency, and connections
- Cancel/remove keep-or-delete dialogs and resume-mismatch decision UI
- Shared SQLite data dir with the CLI via `internal/appdir`

## Phase 5 — Cross-platform hardening

Windows first; macOS and Linux smoke builds and packaging.

## Phase 6 — Browser integration

WebExtension + native messaging, opt-in only.

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
