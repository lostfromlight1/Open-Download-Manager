# Open Download Manager (ODM)

An open-source, cross-platform download manager for Windows, macOS, and Linux.

ODM is a desktop download manager with a reusable **Go** download engine, a **React/TypeScript** UI hosted by **Wails v3**, and planned browser integration.

## Start here

1. Read [`docs/PROJECT_OVERVIEW.md`](docs/PROJECT_OVERVIEW.md).
2. Follow [`docs/TECH_STACK.md`](docs/TECH_STACK.md).
3. Use [`docs/FEATURES.md`](docs/FEATURES.md) as the scope authority.
4. Implement in the order defined by [`docs/IMPLEMENTATION_PLAN.md`](docs/IMPLEMENTATION_PLAN.md).
5. Follow [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md), [`docs/SECURITY.md`](docs/SECURITY.md), and [`docs/TESTING.md`](docs/TESTING.md).
6. For coding-agent behavior, read [`AGENTS.md`](AGENTS.md) and `.cursor/rules/`.

## Product principles

- Reliable downloads before feature breadth.
- Cross-platform behavior must be tested, not assumed.
- No hidden telemetry; be transparent about any diagnostics.
- Keep the download engine independent of the desktop UI.
- Prefer small, reviewable changes with tests.
- Do not claim a speed improvement unless measured under repeatable conditions.

## Current project status

**0.5.0** — v1.0 prep (Windows installer, first-run setup, release workflow). Feature set includes Phase 10 YouTube (managed yt-dlp/ffmpeg), browser intercept, tray, and UI polish. Final **1.0.0** adds Authenticode signing and broader platform QA. A separate background daemon remains later.

License: MIT.

## Install (end users)

Windows: download the NSIS installer from [Releases](https://github.com/lostfromlight1/Open-Download-Manager/releases) and follow **[`docs/GETTING_STARTED.md`](docs/GETTING_STARTED.md)** (no Go/Node required).

## Setup and how to run (developers)

For a **new development machine**, follow **[`SETUP.md`](SETUP.md)** (prerequisites, clone, verify, desktop/CLI commands, troubleshooting).

Quick start once tools are installed:

```bash
git clone https://github.com/lostfromlight1/Open-Download-Manager.git
cd Open-Download-Manager
pnpm install
pnpm build
go test ./...
cd apps/desktop && wails3 dev
```

## Module path

```text
github.com/lostfromlight1/Open-Download-Manager
```
