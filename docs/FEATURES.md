# Product Features and Scope

Feature labels:
- **MVP**: required for the first usable desktop release.
- **Later**: planned after MVP.
- **Out of scope**: intentionally excluded unless product direction changes.

## Download lifecycle

| Feature | Priority | Acceptance notes |
|---|---|---|
| Add download from URL | MVP | Validate URL and show clear errors |
| Choose destination | MVP | User-selected folder; safe filename handling |
| Single-stream HTTP/HTTPS | MVP | Stream to disk without loading full file into RAM |
| Progress percentage and bytes | MVP | Derived from actual bytes written |
| Current speed and ETA | MVP | Smoothed speed; ETA omitted when unknown |
| Pause and resume | MVP | State transitions are race-safe |
| Cancel and remove task | MVP | Ask whether to keep or delete partial/downloaded files; explain the choice clearly |
| Retry transient failures | MVP | Bounded retries with backoff |
| Download queue | MVP | Configurable concurrency and ordering |
| Restart recovery | MVP | Reconcile DB metadata with partial files |
| Multi-connection segments | MVP+ | Only after validated Range support |
| Per-task connection limit | Later | Global and per-host caps |
| Bandwidth limit | Later | Global and per-task limits |
| Schedule downloads | Later | Respect local time and sleep/shutdown behavior |
| Checksums | Later | Verify user-provided hash |
| Import/export task list | Later | Versioned portable format |
| Categories/tags | Later | User-defined organization |

## Desktop experience

### MVP
- Dashboard with active, queued, completed, failed, and paused views.
- Add-download dialog.
- Download detail view.
- Pause/resume/cancel/retry actions.
- Destination folder setting.
- Global concurrency setting.
- Theme preference (light/dark/system).
- Clear empty, loading, and error states.
- Native notifications for completion (opt-in).
- Close-to-tray: hide the window to the system tray (default on); downloads keep running until tray Quit.
- Launch at login (opt-in): OS autostart via Wails; starts hidden to the tray (`--hidden`).
- Tray: live aggregate speed tooltip; Pause all active / Resume paused.
- Desktop window layout usable from ~800px wide to wide screens (no phone layout).

### Later
- Search and filters.
- Download categories and tags.
- Per-download settings.
- Speed graph and history.
- Per-task tray actions.
- Global hotkey.
- Automatic clipboard URL detection (opt-in).
- Drag-and-drop URL support.

## Browser integration

### MVP extension milestone
- Chrome and Firefox WebExtension builds.
- Context-menu action to send a link to the desktop app.
- Optional capture of a direct link from a user click.
- Native Messaging connection to the desktop host.
- Connection status and troubleshooting screen.
- Explicit permission disclosure.

### Download interception (opt-in)
- Extension popup: **Send browser downloads to ODM** (off by default; requests `downloads` permission).
- On browser download start: enqueue http(s) URL (+ suggested filename) into ODM, then cancel/erase the browser item when enqueue succeeds.
- Skip `blob:`/`data:` and likely HTML page bodies (`text/html`, `application/xhtml`).
- Optional **Forward session cookies for intercepted downloads** (requests `cookies` permission): one-shot Cookie + Referer for that task only, held in memory — never stored in SQLite. Restart may require re-intercept for signed-in URLs.
- Requires desktop `browser.integration_enabled` + default download folder (same as enqueue).

### Later
- Broader authenticated intercept (e.g. Authorization header) with a reviewed secret model.
- Send multiple selected links.
- Detect supported media links only when permitted and technically lawful.

Do not collect page contents, browsing history, or cookies by default. Do not inject scripts broadly without a documented need.

## YouTube / media downloads (Phase 10)

Opt-in in Settings (`media.enabled`, default off). ODM downloads and pins **yt-dlp** and **ffmpeg** into the app data `tools/` directory (paths overridable via `media.ytdlp_path` / `media.ffmpeg_path`). Ordinary http(s) downloads still use the Go engine.

| Feature | Priority | Acceptance notes |
|---|---|---|
| Enable YouTube downloads | MVP (Phase 10) | Off by default; short rights/ToS notice; requires default download folder |
| Managed helpers | MVP (Phase 10) | Auto-install pinned yt-dlp/ffmpeg on enable; PATH fallback if present |
| Single video format picker | MVP (Phase 10) | Best / height formats / audio-only (`ba`) |
| Playlist / channel select | MVP (Phase 10) | Flat list → multi-select → enqueue each as its own task (cap 100) |
| Progress + cancel | MVP (Phase 10) | Progress from yt-dlp output; cancel kills process |
| YouTube host allowlist | MVP (Phase 10) | Reject non-YouTube sites even if yt-dlp supports them |
| Extension “choose format” | MVP (Phase 10) | Context menu / page action → desktop format picker; intercept never steals YouTube hosts |

User is responsible for complying with YouTube terms and only downloading content they have rights to.

## Protocol and networking scope

### MVP
- HTTP and HTTPS.
- Redirect handling with safe limits.
- HEAD request only as an optimization; it must not be required for correctness.
- GET streaming.
- Range requests with response validation.
- ETag and Last-Modified validators for resume decisions.
- Retry/backoff for transient network failures.
- Timeouts and cancellation.

### MVP proxy and unreliable networks
- Detect and use OS proxy settings by default where possible.
- Allow manual HTTP/HTTPS proxy configuration.
- Allow disabling proxy usage.
- Retry/backoff, timeouts, and stalled-transfer detection for flaky links.
- On resume, if remote validators changed, ask the user instead of silently discarding progress.

### Later
- FTP/SFTP only after a separate threat model and dependency review.
- Metalink/checksum metadata.
- Additional proxy protocols such as SOCKS.
- Optional Network Access / Tunneling track (separate component, explicit opt-in; not required for ordinary downloads).

## Reliability requirements

- No overlapping segment writes.
- No unbounded task creation.
- No unbounded in-memory response buffering.
- Safe behavior on disk-full and permission errors.
- Safe handling of malformed headers and inconsistent server responses.
- Detect stale partial files and incompatible remote representations.
- Never display 100% until final validation/finalization succeeds.

## Explicitly excluded from MVP

- BitTorrent.
- DRM bypass.
- Paywall/authentication bypass.
- Arbitrary yt-dlp sites beyond YouTube (Phase 10 is YouTube-only).
- Cloud account/synchronization.
- Remote-control server.
