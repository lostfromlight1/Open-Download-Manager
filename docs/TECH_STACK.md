# Technology Stack

## Decision summary

| Area | Choice | Why |
|---|---|---|
| Desktop shell | Wails v3.1.0 | Go backend, native WebView, React frontend |
| Core language | Go (1.26+) | Learning fit, concurrency, portable engine |
| HTTP client | `net/http` (+ focused helpers) | Explicit transport, proxy, timeouts |
| Serialization | encoding/json (+ typed DTOs) | IPC and config payloads |
| Database | SQLite | Embedded persistence for tasks and settings |
| Frontend | React + TypeScript | Productive UI stack |
| Build tool | Vite | Fast frontend development |
| Package manager | pnpm | Workspace support and reproducible installs |
| UI styling | Tailwind CSS | Utility-first styling for the shell |
| Frontend state | React state first; Zustand later if needed | Avoid unnecessary libraries |
| Browser integration | Chrome/Firefox WebExtensions | Later milestone |
| Native bridge | Native Messaging | Later milestone |
| Go module | `github.com/lostfromlight1/Open-Download-Manager` | Matches the GitHub repository |
| CLI | `odm` | Product CLI name |
| CI | GitHub Actions | Format, vet, test, frontend checks |
| Logging | slog (stdlib) unless needs grow | Structured diagnostics |

## Alternatives considered

- **Rust + Tauri:** excellent safety and small shell; replaced because Go matches the maintainer’s learning and velocity goals. Phase 0 Rust/Tauri artifacts were migrated out in Phase 0′.
- **Electron:** larger runtime footprint.
- **Mixing Rust and Go:** rejected. One backend language only.

## Important implementation choices

### HTTP engine

Use Go’s `net/http` with an explicit client, timeouts, redirect policy, and proxy configuration. Stream response bodies to disk.

### Proxy

Support from early phases:

- OS/system proxy detection where available.
- Manual HTTP/HTTPS proxy configuration.
- Ability to disable proxy usage.
- Design transport so later protocols (for example SOCKS) can be added without rewriting the engine.

### Database

SQLite stores task metadata, queue state, settings, and schema version. Do not store secrets in plaintext.

Driver: `modernc.org/sqlite` (pure Go, no CGO) so Windows/macOS/Linux cross-builds stay simple without a C toolchain. Schema changes go through versioned files under `internal/storage/migrations/`.

### Desktop IPC

Wails services and events are the desktop bridge. The UI requests actions and listens for progress; it does not own download workers.

### Network Access / Tunneling

Optional access assistance is a separate track. It must not be required for ordinary downloads. See `docs/ARCHITECTURE.md` and `docs/IMPLEMENTATION_PLAN.md`.

## Phase 0′ bootstrap choices

| Package | Version | Role |
|---|---|---|
| `github.com/wailsapp/wails/v3` | `v3.1.0` | Desktop shell |
| `@wailsio/runtime` | `3.1.0` | Frontend runtime / Vite plugin |
| React / Vite / Tailwind / Vitest | current stable lines used by the shell | UI toolchain |

Exact versions are in `go.sum` and `pnpm-lock.yaml`.
