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

## Quality gates

Before merging:

- `gofmt` clean / `go vet ./...`
- `go test ./...`
- Frontend format, lint, typecheck, and unit tests
- Frontend production build
- Review dependency changes and license compatibility

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
