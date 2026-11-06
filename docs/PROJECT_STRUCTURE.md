# Project Structure

```text
Open-Download-Manager/
├── apps/
│   └── desktop/                 # Wails v3 desktop shell
│       ├── main.go
│       ├── shellservice.go
│       ├── frontend/            # React + TypeScript UI
│       ├── build/               # Wails packaging assets
│       └── Taskfile.yml
├── cmd/
│   └── odm/                     # CLI: version + download
├── internal/
│   ├── engine/                  # single-stream download logic (no Wails)
│   ├── netx/                    # HTTP client, proxy modes, timeouts
│   ├── testserver/              # deterministic flaky HTTP fixtures
│   └── version/                 # shared version metadata
│   # future: manager, storage, access
├── docs/
├── .cursor/rules/
├── .github/workflows/
├── go.mod
├── go.sum
├── package.json
├── pnpm-workspace.yaml
├── AGENTS.md
├── CONTRIBUTING.md
├── SECURITY.md
├── LICENSE
└── README.md
```

## Structure rules

- Engine code under `internal/` must not import Wails.
- `apps/desktop` wires UI to Go services; it must not become the download engine.
- `cmd/odm` stays a thin CLI over `internal/engine`.
- Create additional `internal/*` packages when a phase needs them; do not invent empty packages early.
- Browser extension arrives in a later phase under `apps/browser-extension`.

## Current layout notes

- Phase 0′: Go module + Wails desktop shell + React frontend.
- Phase 1: `odm download` uses `internal/engine` + `internal/netx`; desktop UI still reports downloads unavailable.
