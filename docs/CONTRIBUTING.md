# Contributing

## Before opening an issue

Search existing issues and include:

- Operating system and version.
- App version or commit.
- Steps to reproduce.
- Expected and actual behavior.
- Relevant sanitized logs.
- Whether the issue reproduces with a direct HTTP client.

Never attach private download URLs, cookies, tokens, or personal files.

## Development setup

Follow [`SETUP.md`](SETUP.md) on a new machine. Before opening a pull request, run:

```bash
pnpm install
pnpm build
gofmt -w .
go vet ./...
go test ./...
pnpm format
pnpm lint
pnpm typecheck
pnpm test
```

Phase 3 provides segmented multi-connection downloads in the engine (`--connections`, `download.max_connections`) with safe single-stream fallback. Desktop UI download wiring belongs in Phase 4.

## Pull requests

- Keep changes focused.
- Explain the problem and solution.
- Add tests for behavior changes.
- Update documentation.
- Include screenshots for UI changes when useful.
- Mention platform-specific testing performed.
- Disclose new dependencies and their licenses.

## Code style

- Go: `gofmt`, `go vet`, idiomatic package layout.
- TypeScript: project formatter, linter, and strict type checks.
- Prefer descriptive names and small functions.
- Document non-obvious concurrency and filesystem safety decisions.
