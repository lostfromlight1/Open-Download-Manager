# Open Download Manager

An open-source, cross-platform Internet Download Manager alternative for Windows, macOS, and Linux.

This repository is planned as a desktop download manager with a reusable Rust download engine, a React/TypeScript UI, and browser integration.

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

Planning/documentation baseline. The implementation plan starts with a Rust CLI prototype and local HTTP test server before building the desktop UI.
