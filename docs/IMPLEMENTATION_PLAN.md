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

**Status:** implemented (opt-in WebExtension + native messaging; interception is Phase 8).

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
- No history/page/cookie collection; download interception delivered in Phase 8

## Phase 7 — Close-to-tray background

**Status:** implemented (same-process tray hide; separate daemon / autostart deferred).

Keep downloads alive in the **same** desktop process when the window is hidden (not a separate `odm-service` daemon).

**Deliverables**

- Setting `ui.close_to_tray` (default true) via storage, Settings UI, and `odm settings`
- Wails system tray with **Show Open Download Manager** and **Quit**
- `WindowClosing`: hide + cancel quit when enabled; real exit via tray Quit (or close when setting is off)
- Show the window when a resume-mismatch decision is required while hidden
- Docs + unit tests for setting round-trip and hide-on-close logic

**Deferred**

- Separate headless background service binary
- Launch-at-login delivered in Phase 9
- Deep Linux DE tray quirks beyond basic Wails support

**Exit criteria**

- Close-to-tray on: close hides to tray; active downloads continue via `RunQueue`; Show restores UI; Quit exits and stops the queue
- Close-to-tray off: close quits the app as before
- Setting persisted and controllable from Settings + CLI
- No second download engine

## Phase 8 — Browser download interception + desktop UI polish

**Status:** implemented (opt-in extension intercept; desktop window fluid layout).

**Deliverables**

- Extension popup toggle **Send browser downloads to ODM** (default off); optional `"downloads"` permission
- `chrome.downloads.onCreated`: filter http(s), skip HTML/XHTML MIME; enqueue via existing NM; cancel+erase on success
- Pure `shouldInterceptDownload` helper with Node tests
- Desktop UI: MinWidth/MinHeight, fluid stats/rows/dialogs/settings for ~800px+ windows
- Docs: SECURITY/FEATURES/TESTING/SETUP note that session cookies are not forwarded

**Exit criteria**

- With intercept off, browser downloads behave normally
- With intercept on + ODM integration enabled: normal file download appears in ODM and browser item is cancelled
- HTML-like MIME downloads are not stolen from the browser
- Enqueue failure leaves the browser download alone
- Desktop UI usable from min window size to default 1040×720

## Phase 9 — Auth handoff, launch-at-login, richer tray

**Status:** implemented (memory-only Cookie/Referer; Wails Autostart; tray speed + pause/resume).

**Deliverables**

- Extension opt-in **Forward session cookies for intercepted downloads** (`cookies` permission from popup)
- NM `cookie` / `referer` fields; manager in-memory headers; engine ExtraHeaders; strip on cross-host redirect
- Setting `ui.launch_at_login` + Wails `Autostart` with `--hidden` start to tray
- Tray tooltip aggregate speed; menu Pause all active / Resume paused
- Docs/tests; cookies never persisted to SQLite

**Deferred**

- v1.0 code signing / notarization
- Separate headless daemon
- Arbitrary Authorization header forwarding

**Exit criteria**

- Cookie handoff succeeds for auth-gated URLs in-process; restart loses ephemeral headers
- Launch-at-login opt-in registers Autostart; `--hidden` starts to tray
- Tray shows live speed; Pause/Resume work; Quit stops the queue

## Phase 10 — YouTube downloads (managed yt-dlp)

**Status:** implemented (YouTube allowlist; managed helpers; format + playlist pickers).

**Deliverables**

- `internal/media`: Resolve/EnsureTools (pinned yt-dlp + ffmpeg under data dir), Probe (`-J` / flat playlist), Download with progress parse + ctx cancel
- Settings: `media.enabled`, `media.ytdlp_path`, `media.ffmpeg_path`; EnsureTools on enable
- Task columns: `media_source`, `media_format_id`, `media_webpage_url`, `media_playlist_id` (migration 003)
- Manager `AddMedia` / `executeMedia`; ordinary `Add` rejects YouTube URLs
- Desktop: ProbeMediaURL / AddMediaDownload / IsYouTubeURL; Settings toggle; FormatPicker + PlaylistPicker
- Docs/SECURITY: subprocess allowlist, no shell concat, pinned upstream URLs, user consent

**Deferred**

- Non-YouTube extractors, subtitles/chapters UI, sponsorblock, cookies-from-browser
- Silent browser auto-download of YouTube without format selection

**Exit criteria**

- Media enabled + helpers installed → probe single URL, choose format (incl. audio), file lands with UI progress
- Playlist/channel → selectable entries; only selected become tasks
- Media disabled → clear Settings hint; http(s) engine unchanged

## v1.0 prep (0.5.0) — Windows easy install / setup

**Status:** shipped as **1.0.0** (Windows-first; Authenticode still deferred).

**Deliverables**

- Per-user NSIS installer (no UAC), finish-page launch, signing placeholders
- First-run setup wizard when `download.directory` is empty
- `docs/GETTING_STARTED.md` for end users; Release workflow + SHA256
- Version branding **1.0.0**; Authenticode signing remains a follow-up

## Post-1.0 development

Ongoing work lands on the `development` branch first; `main` stays the stable release line.

## Phase N — Network Access / Tunneling track

**After** reliable downloading and proxy support.

**Deliverables**

- AccessProvider interface implementation spike
- Opt-in settings and user-configurable endpoints
- Threat model, limitation docs, security review gate
- Isolation tests proving downloads work with access disabled

Do not block Phases 1–2 on this track.
