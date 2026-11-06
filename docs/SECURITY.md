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
- Validate message schema, version, size, URL, and allowed action.
- Do not expose arbitrary shell, process, or filesystem commands.
- Authenticate/validate the calling browser where the platform mechanism permits.
- Require explicit user configuration and provide a disable switch.
- Keep permissions minimal.

### Persistence and secrets
- Use parameterized SQL.
- Avoid storing credentials in plaintext.
- If authenticated downloads are added, integrate with OS credential storage or a reviewed secret-management approach.
- Provide a clear way to remove download history and task metadata.

### Updates and dependencies
- Sign release artifacts and updates.
- Review dependency licenses and known vulnerabilities.
- Keep dependencies updated.
- Protect release workflows and signing secrets.
- Publish checksums for release artifacts.

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
