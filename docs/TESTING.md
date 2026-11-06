# Testing Strategy

## Test layers

### Unit tests
- URL validation and normalization.
- Segment boundary calculation.
- Content-Range parsing.
- Lifecycle state transitions.
- Retry/backoff policy.
- Filename sanitization.
- Progress aggregation.
- Persistence mapping.
- Proxy mode selection.

### Integration tests
Use a local deterministic HTTP server. Do not rely on public internet servers for correctness tests.

Required server behaviors:

- Standard 200 response.
- Valid 206 range response.
- Ignore Range and return 200.
- Malformed Content-Range.
- Incorrect segment length.
- Redirect.
- 404, 410, 429, and 5xx.
- Slow response and timeout.
- Stall after headers / slow first byte.
- Connection closes mid-body.
- Changed ETag between attempts.
- Missing Content-Length / chunked response.

### End-to-end tests
- Add URL and select destination.
- Pause/resume.
- Cancel and retry with keep/delete choices.
- Restart and recover tasks.
- Queue multiple tasks and enforce concurrency.
- Proxy system/manual/off paths.
- Browser extension handoff (when implemented).

## Phase 0′ / Phase 1 checks

- Go tests for shell status service and version package.
- Frontend tests render the product name and unavailable-download message, and assert no fabricated progress.
- `go test ./...` requires `apps/desktop/frontend/dist` to exist because the desktop package embeds it.
- `internal/engine` integration tests use `internal/testserver` for OK, redirect, 404, 429, 5xx, stall, mid-body drop, and no-Range behaviors.
- `internal/netx` unit tests cover proxy mode parsing and client construction.
- `internal/task` unit tests cover allowed/illegal state transitions.
- `internal/storage` tests round-trip tasks/settings through SQLite migrations.
- `internal/manager` tests cover queue completion, cancel+delete files, resume mismatch abort, and remove+keep files.
- Phase 3: `internal/engine` covers segmented downloads, Range verification, planner boundaries, and single-stream fallback; `internal/storage` covers segment row CRUD after migration 002.
- Phase 4: `apps/desktop` tests cover DownloadService list/add/settings/cancel and interactive mismatch decisions; frontend tests cover empty dashboard, boot errors, progress reducer, and format helpers.
- Phase 5: `apps/desktop` openpath planner tests cover open/reveal command args (Windows/macOS/Linux) and missing-file parent fallback without launching a GUI.

## Quality gates

Before merging:

- `gofmt` clean / `go vet ./...`
- `go test ./...`
- Frontend format, lint, typecheck, and unit tests
- Frontend production build
- Review dependency changes and license compatibility
- CI `desktop-package` job builds platform packages (NSIS / `.app` / `.deb`) when packaging files change

## Phase 5 smoke checklists

### Windows (required for local release verification)

1. `CI=true wails3 build` then `wails3 task windows:package` (requires NSIS/`makensis` on `PATH`).
2. Run `apps/desktop/bin/odm-amd64-installer.exe`, install, launch the app.
3. Add a small HTTP download → pause → resume → complete.
4. Open Settings: theme + destination Browse; toggle notify-on-complete if desired.
5. From Details: Open file / Show in folder for a completed task.
6. Uninstall via the installer entry. Note: SQLite data under `%AppData%\odm` is **not** removed by uninstall (by design for Phase 5).

### macOS / Linux (contributor optional; CI is the Phase 5 gate)

1. Download the CI `desktop-package` artifact for the OS, or build locally (`darwin:package` / `linux:create:deb`).
2. Install or open the package; confirm the window launches.
3. Add one download and confirm list/progress update.
4. Confirm Open / Reveal uses the platform file manager (`open` / `xdg-open`).

## Reliability tests

- Repeated pause/resume.
- Cancel while writers are active.
- Crash/restart during transfer.
- Disk full and permission denied.
- Remote file changes while paused.
- Many simultaneous small downloads.
- One very large download.
- Non-ASCII names and long paths.
- App shutdown while jobs are active.
