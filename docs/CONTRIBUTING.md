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

Phase 4 wires the Wails desktop UI to `internal/manager` (same DB as the CLI). Phase 5 adds unsigned packaging (NSIS / `.app` / `.deb`) and CI artifacts — see [`SETUP.md`](../SETUP.md) §5.5. Keep download logic in Go; React stays presentation-only.

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
