# Project Overview

## 1. Vision

Build a trustworthy, open-source, cross-platform download manager that makes large downloads easier to manage, pause, resume, organize, and monitor.

The product should feel like a modern desktop application while keeping its core download engine reusable by a future CLI, browser integration, or background service.

## 2. Target platforms

- Windows 10/11 (confirm minimum versions during packaging work)
- macOS (support current maintained versions; document minimum OS)
- Linux (prioritize AppImage/deb/rpm only after validating the target distributions)

Platform support is a release commitment: do not advertise a platform as supported until CI and manual smoke tests cover it.

## 3. Target users

- People downloading large files or many files.
- Developers and power users who want queues, scheduling, and resumability.
- Users who want an open-source alternative to proprietary download managers.

## 4. Product goals

1. Reliable HTTP/HTTPS downloads.
2. Pause, resume, retry, and queue management.
3. Multi-connection downloads when the server safely supports byte ranges.
4. Recovery after application restart or crash.
5. Clear progress, speed, ETA, and error reporting.
6. Browser handoff through supported WebExtensions.
7. Consistent behavior across Windows, macOS, and Linux.

## 5. Non-goals for the first release

- BitTorrent client.
- Video scraping or downloading from streaming services.
- Bypassing paywalls, DRM, authentication, or access controls.
- Cloud synchronization or user accounts.
- Mobile applications.
- Remote download server or multi-device control.
- Built-in browser.
- Guaranteed acceleration for every server or network.

## 6. Product principles

- Correctness over raw speed.
- User control over automation.
- Explicit consent for browser integration and destination folders.
- No silent collection of browsing or download history.
- Minimal privileges and least authority.
- Graceful fallback when a server does not support segmented downloads.

## 7. Release strategy

- v0.1: reliable single-stream HTTP/HTTPS `odm` CLI engine and flaky-network test harness.
- v0.2: persistence, queue, proxy settings, and user decision flows for cancel/remove/resume.
- v0.3: segmented downloads with safe fallback (engine + CLI; desktop UI later).
- v0.4: Wails desktop application with queue and settings.
- v0.5: browser extension and native messaging host.
- Phase 7: close-to-tray keeps downloads alive when the window is hidden (same process).
- Phase 8: opt-in browser download interception + desktop UI fluid layout (~800px+).
- Phase 9: cookie/Referer handoff (memory-only), launch-at-login, richer tray.
- Phase 10: optional YouTube downloads (managed yt-dlp/ffmpeg; format + playlist select).
- 0.5.0 (v1.0 prep): Windows per-user NSIS installer, first-run setup wizard, GitHub Release + SHA256 (signing hooks only).
- **1.0.0:** first stable release (unsigned Windows installer + SHA256; Authenticode signing deferred).
- Later: Authenticode signing, optional separate background service; Network Access / Tunneling track.

Version numbers are planning labels, not promises.
