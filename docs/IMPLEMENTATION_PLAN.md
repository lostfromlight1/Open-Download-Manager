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

**Deliverables**

- Task model and explicit state machine
- SQLite schema and migrations
- Queue and concurrency limits
- Cancel/remove confirmation choices (keep/delete files)
- Resume mismatch asks the user
- Full proxy settings persistence

## Phase 3 — Segmented downloads

**Deliverables**

- Range capability verification
- Segment planner and bounded workers
- Random-access writes without overlap
- Safe single-stream fallback

## Phase 4 — Desktop MVP

**Deliverables**

- Dashboard and add-download flow
- Real engine progress events
- Settings for destination and proxy
- Cancel/remove/resume decision UI

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
