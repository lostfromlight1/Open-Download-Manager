# Project Structure

```text
Open-Download-Manager/
├── apps/
│   └── desktop/                 # Wails v3 desktop shell
│       ├── main.go              # Wails bootstrap + manager queue
│       ├── shellservice.go
│       ├── downloadservice.go   # thin Wails facade over manager
│       ├── frontend/            # React + TypeScript UI
│       ├── build/               # Wails packaging assets
│       └── Taskfile.yml
├── cmd/
│   └── odm/                     # CLI: download, task, queue, settings
├── internal/
│   ├── appdir/                  # OS data-dir resolution
│   ├── engine/                  # single-stream + segmented download logic (no Wails)
│   ├── media/                   # YouTube probe/download via managed yt-dlp (Phase 10)
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
- Browser extension lives under `apps/browser-extension`; native host is `cmd/odm-nmhost` + `internal/browserhost`.

## Current layout notes

- Phase 0′: Go module + Wails desktop shell + React frontend.
- Phase 1: `odm download` uses `internal/engine` + `internal/netx`.
- Phase 2: `odm task` / `odm queue` / `odm settings` use `internal/manager` + SQLite.
- Phase 3: engine verifies Range, plans segments, runs bounded workers; CLI `--connections`; settings `download.max_connections`.
- Phase 4: desktop DownloadService + React dashboard/settings wired to manager events.
- Phase 5: packaging metadata under `apps/desktop/build/`; CI `desktop-package` job uploads NSIS / `.app` / `.deb`.
- Phase 6: `apps/browser-extension` + `cmd/odm-nmhost` enqueue via shared SQLite.
- Phase 10: `internal/media` + Settings YouTube opt-in; helpers under app data `tools/`.
