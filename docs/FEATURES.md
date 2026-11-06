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

### Later
- Search and filters.
- Download categories and tags.
- Per-download settings.
- Speed graph and history.
- Tray/menu-bar integration.
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

### Later
- Intercept browser downloads with explicit user opt-in.
- Send multiple selected links.
- Detect supported media links only when permitted and technically lawful.

Do not collect page contents, browsing history, or cookies by default. Do not inject scripts broadly without a documented need.

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
- Video streaming extraction.
- Cloud account/synchronization.
- Remote-control server.
