# Security and Privacy Requirements

## Threat model

The application downloads untrusted remote content and accepts URLs and commands from a browser extension. Treat remote servers, response headers, filenames, browser messages, and downloaded files as untrusted.

## Required controls

### URL and network safety
- Accept only supported schemes (`http` and `https`) in MVP.
- Reject malformed URLs and unsupported schemes.
- Define a redirect policy and revalidate each redirect target.
- Allow localhost and private-network destinations for legitimate local/dev downloads, but warn in the UI when appropriate.
- For browser-originated URLs, reassess SSRF risk before enabling broad private-network access.
- Do not disable TLS verification.
- Apply connection, read, and total timeouts.
- Bound response headers and message sizes.
- Avoid logging credentials or signed query parameters.

### Filesystem safety
- Sanitize filenames from URLs and `Content-Disposition`.
- Prevent path traversal and writes outside the chosen destination.
- Do not follow attacker-controlled symlinks when finalizing files where platform APIs allow safer handling.
- Use temporary/partial files and safe finalization.
- Handle disk-full and permission failures without corrupting unrelated files.

### Browser native messaging
- Validate message schema, version, size, URL, and allowed action (`ping` / `enqueue` only; max 256 KiB).
- Do not expose arbitrary shell, process, or filesystem commands.
- Authenticate/validate the calling browser where the platform mechanism permits (Chrome `allowed_origins`, Firefox `allowed_extensions`).
- Require explicit user configuration (`browser.integration_enabled`, default off) and provide a disable switch that unregisters the host.
- Keep permissions minimal; optional click-capture requests host permissions only when enabled.
- Optional download interception requests the browser `downloads` permission only when the user enables **Send browser downloads to ODM** in the extension popup (default off).
- Optional cookie handoff requests the browser `cookies` permission only when **Forward session cookies for intercepted downloads** is enabled (default off). Cookie/Referer strings are validated (size + no CR/LF), applied as ephemeral in-memory headers for that task, never written to SQLite or logged, and dropped on cross-host redirects.
- Interception cancels the browser download only after a successful enqueue; it does not collect browsing history or page content.
- Host enqueues only; it does not run a second download scheduler.

### Persistence and secrets
- Use parameterized SQL.
- Avoid storing credentials in plaintext.
- If authenticated downloads are added, integrate with OS credential storage or a reviewed secret-management approach.
- Provide a clear way to remove download history and task metadata.

### Updates and dependencies
- Sign release artifacts and updates (Authenticode hooks in NSIS + optional `WINDOWS_CERT_*` secrets on the Windows release workflow; required for final 1.0.0).
- Review dependency licenses and known vulnerabilities.
- Keep dependencies updated.
- Protect release workflows and signing secrets.
- Publish checksums for release artifacts (`SHA256SUMS.txt` on GitHub Releases).

## YouTube / media helpers (yt-dlp)

Opt-in only (`media.enabled`, default off). When enabled, ODM may download pinned third-party binaries (yt-dlp, ffmpeg) into the app data `tools/` directory from fixed upstream release URLs.

Controls:
- Do not invoke a shell; pass argv arrays to `exec.Command` / `CommandContext`.
- Allowlist YouTube host patterns for probe/download; reject other sites in MVP.
- Validate destination directory and sanitize output basename templates.
- Cancel via context cancellation (kill process); do not leave orphaned writers writing outside the destination.
- Do not log cookies, Authorization, or signed query strings from media URLs.
- Pin helper versions; prefer checksum verification when publishing updates.
- Document that users remain responsible for rights and service terms; feature stays disabled until helpers are ready or PATH provides them.
- Threat note: running third-party binaries expands the trusted computing base — only fetch from pinned upstream URLs after explicit user opt-in.

## Network Access / Tunneling

If an optional access/tunneling component is added:

- Keep it behind explicit opt-in controls.
- Require user-configurable endpoints; do not silently route traffic through undisclosed infrastructure.
- Document threat model, traffic-leak risks, credential handling, and safe shutdown.
- Ensure ordinary downloads still work when the feature is disabled.
- Complete a security review before shipping the feature.
- Document legal and personal-safety considerations honestly for users.

## Privacy defaults

- No account required.
- No telemetry by default.
- Do not upload download URLs, filenames, or history.
- If optional diagnostics are introduced, explain exactly what is collected and obtain consent.
- Never collect browser history or page contents by default.

## Security testing

Test malformed URLs, redirect chains, hostile filenames, oversized headers, malformed range responses, invalid native messages, path traversal strings, and interrupted writes.
