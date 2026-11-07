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
- Phase 6: `internal/browserhost` covers NM framing, schema rejection, disabled/no-directory enqueue, and happy-path enqueue into a temp SQLite store.
- Phase 7: `internal/storage` round-trips `ui.close_to_tray`; `apps/desktop` covers `ShouldHideOnClose` and resume-mismatch `NeedUI` (show window when a decision is required).
- Phase 8: `apps/browser-extension` `npm test` covers http(s) accept, HTML/blob skip, and YouTube button context-invalidated error classification; desktop UI remains covered by existing frontend tests.
- Phase 9: browserhost cookie/referer validation; manager ephemeral Cookie handoff; engine ExtraHeaders allowlist; storage `ui.launch_at_login`; tray tooltip formatting.
- Phase 10: `internal/media` probe JSON fixtures, progress line parse, YouTube URL allowlist, tools path resolution (no network); storage media columns + `media.*` settings.
- UI polish: frontend `formatMbps` + `partitionSelection` unit tests; manual check of active-row speed graph and bulk select toolbar.
- Live YouTube (opt-in): `ODM_LIVE_YT=1 go test ./internal/media -run LiveDownload -count=1 -timeout 30m -v` and `ODM_LIVE_YT=1 go test ./internal/manager -run LiveMediaManager -count=1 -timeout 30m -v` (needs helpers under `%APPDATA%/odm/tools`).
- Windows media helpers must not flash a console (`hideHelperConsole` / `CREATE_NO_WINDOW`); progress uses on-disk byte estimates when yt-dlp omits totals.

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

1. `CI=true wails3 build` then `wails3 task windows:package` (requires NSIS/`makensis` on `PATH`; default **per-user** `INSTALL_SCOPE=user`, no UAC).
2. Run `apps/desktop/bin/odm-amd64-installer.exe`, install without admin, leave **Launch Open Download Manager** checked.
3. First-run wizard: set download folder → optionally skip browser/YouTube → land on dashboard.
4. Add a small HTTP download → pause → resume → complete.
5. Open Settings: theme + destination Browse; toggle notify-on-complete if desired.
6. From Details: Open file / Show in folder for a completed task.
7. Uninstall via the installer entry. Note: SQLite data under `%AppData%\odm` is **not** removed by uninstall (by design for Phase 5).

### End-user install smoke (1.0.0)

1. Install from a GitHub Release `odm-amd64-installer.exe` (or local NSIS build) without elevating.
2. Confirm install path is under `%LOCALAPPDATA%\Programs\…`.
3. Complete first-run folder step; confirm Add Download works afterward.
4. Optional: enable browser + load unpacked extension; optional YouTube probe of one short URL.
5. Compare installer SHA256 to `SHA256SUMS.txt` when installing from a Release.

### macOS / Linux (contributor optional; CI is the Phase 5 gate)

1. Download the CI `desktop-package` artifact for the OS, or build locally (`darwin:package` / `linux:create:deb`).
2. Install or open the package; confirm the window launches.
3. Add one download and confirm list/progress update.
4. Confirm Open / Reveal uses the platform file manager (`open` / `xdg-open`).

## Phase 6 browser smoke (manual)

1. Build host: `go build -o apps/desktop/bin/odm-nmhost.exe ./cmd/odm-nmhost` (or `odm-nmhost` on Unix) and ensure it sits next to the desktop binary or on `PATH`.
2. In ODM Settings: set download folder, enable browser integration, paste Chrome extension ID (from `chrome://extensions` with Developer mode), Save.
3. Confirm the green status text reports **browser host installed** (or a clear error if `odm-nmhost` is missing).
4. Load unpacked extension from `apps/browser-extension` (Firefox: use `manifest.firefox.json` as `manifest.json`).
5. Extension popup → Refresh status → expect “Connected” when enabled.
6. Right-click a link → Download with ODM → task appears in the desktop list within ~1s.
7. Disable integration in Settings → Save → confirm **browser host uninstalled** message; enqueue should fail in the extension.

## Phase 7 close-to-tray smoke (manual)

1. Confirm Settings → **Keep downloading in the system tray when I close the window** is checked (default).
2. Start a sizable download; confirm progress updates.
3. Close the main window (X). App should remain in the system tray; download progress should continue (re-open and check bytes/speed).
4. Tray → **Show Open Download Manager** → window restores with live state.
5. Optional: pause a ranged download, change the remote file / validators if you have a fixture, resume while hidden — window should appear for the mismatch dialog.
6. Tray → **Quit** → process exits; queue stops (no further writes).
7. Disable close-to-tray in Settings → Save → close the window → app should exit fully (no lingering tray icon).
8. CLI: `odm settings show` lists `ui.close_to_tray`; `odm settings set ui.close_to_tray false` persists.

## Phase 8 interception + UI smoke (manual)

### Download interception

1. Complete Phase 6 setup (host installed, integration enabled, extension loaded).
2. Extension popup → enable **Send browser downloads to ODM** → allow the Downloads permission when prompted (must stay checked; permission is requested from the popup, not the service worker).
3. Confirm popup status is **Connected — browser integration enabled** (otherwise enqueue fails and the browser keeps the file).
4. In Chrome, start a normal http(s) file download (e.g. a zip from a public URL).
5. Expect: browser download cancelled/removed; task appears in ODM and downloads there.
6. Disable the popup toggle → normal browser downloads work again.
7. With toggle on, confirm opening ordinary web pages still works (HTML MIME is not intercepted).
8. Optional: with ODM integration disabled, intercept should leave the browser download running after enqueue failure.

### Desktop layout

1. Resize the window near the minimum (~800×560) and at the default (~1040×720).
2. Confirm stats, filter tabs, download rows, Settings path+Browse, and dialogs remain usable without horizontal clipping.

## Phase 9 smoke (manual)

### Cookie handoff

1. Enable browser integration + intercept; enable **Forward session cookies for intercepted downloads** (allow Cookies).
2. While signed in to a site that gates downloads, start a download that would 401 without cookies.
3. Expect the task to complete in ODM. Restart ODM and resume the same auth URL without re-intercept — expect failure unless the URL is public.

### Launch-at-login + richer tray

1. Settings → enable **Launch Open Download Manager when I sign in** → Save (Windows smoke).
2. Confirm tray menu includes Pause all active / Resume paused; active download updates tooltip speed.
3. Quit; sign out/in or run the registered command with `--hidden` → app should start to tray without showing the window until Show.
4. Disable launch-at-login → Save → autostart entry removed.

## Phase 10 YouTube smoke (manual)

1. Settings → set default download folder → enable **YouTube downloads** → Save (helpers download into app data `tools/`; status shows ready).
2. Add Download → paste a single `youtube.com/watch` or `youtu.be` URL → choose Best / a height / Audio only → task progresses and file appears in the folder.
3. Paste a playlist or channel URL → select a few entries (not all required) → only those tasks enqueue.
4. Cancel a running media task → process stops; partials follow keep/delete like ordinary downloads.
5. Disable YouTube downloads → Save → Add a YouTube URL → clear error pointing at Settings.
6. Ordinary http(s) download still works with media enabled or disabled.
7. CLI: `odm settings show` lists `media.enabled`; `odm settings set media.enabled true` persists (helpers install from desktop Settings Save).
8. Extension: teal **ODM** button on watch pages, or right-click → **Download YouTube with ODM (choose format)…** → desktop format picker. Intercept must not enqueue YouTube scraps; click capture must not block watching.
9. If probe fails with “page needs to be reloaded”, Settings → **Check for helper updates** (re-downloads pinned yt-dlp) and retry.
10. Completed media: Details → Open file / Show in folder must open the merged `.mp4` (not `.f312` fragments).
11. Do not start two Best A+V downloads of the same watch URL at once — the second enqueue should error with “already queued or downloading”; after the first finishes, a second copy may be started.

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
