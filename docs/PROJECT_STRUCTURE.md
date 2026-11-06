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
│   └── odm/                     # CLI: download, task, queue, settings
├── internal/
│   ├── appdir/                  # OS data-dir resolution
│   ├── engine/                  # single-stream download logic (no Wails)
│   ├── manager/                 # queue, lifecycle, user decisions
│   ├── netx/                    # HTTP client, proxy modes, timeouts
│   ├── storage/                 # SQLite + migrations
│   ├── task/                    # task model + state machine
│   ├── testserver/              # deterministic flaky HTTP fixtures
│   └── version/                 # shared version metadata
│   # future: access (tunneling track)
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
- `cmd/odm` stays a thin CLI over `internal/engine` and `internal/manager`.
- Create additional `internal/*` packages when a phase needs them; do not invent empty packages early.
- Browser extension arrives in a later phase under `apps/browser-extension`.

## Current layout notes

- Phase 0′: Go module + Wails desktop shell + React frontend.
- Phase 1: `odm download` uses `internal/engine` + `internal/netx`.
- Phase 2: `odm task` / `odm queue` / `odm settings` use `internal/manager` + SQLite.
- Desktop UI still reports downloads unavailable until Phase 4.
