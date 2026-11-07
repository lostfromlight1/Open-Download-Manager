# Architecture

## 1. High-level architecture

```text
React + TypeScript UI
        |
        | typed Wails services / events
        v
    Wails desktop application
        |
        v
Application orchestration layer
        |
        +---- Download manager (task lifecycle, queue, concurrency)
        |
        +---- Download engine (job execution)
        |         |
        |         +---- HTTP transport (net/http)
        |         +---- Segment planner / validator
        |         +---- File writer
        |         +---- Retry / cancellation
        |
        +---- Media runner (Phase 10; opt-in yt-dlp for YouTube)
        |
        +---- Persistence (SQLite)
        |
        +---- Optional Access provider (Network Access / Tunneling track)
        |
        +---- Platform adapters (notifications, paths)

CLI (`odm`) -------------------------> same engine / manager packages
Desktop tray (hide window) ----------> same process / RunQueue (Phase 7)
Future headless service (deferred) --> same engine / manager packages

Browser extension ---> Native Messaging host ---> enqueue API
  (optional downloads.onCreated intercept → enqueue + cancel browser item;
   optional one-shot Cookie/Referer in memory for that task)
```

## 2. Architectural boundaries

### Engine packages (`internal/...`)

Reusable Go libraries. They must not import Wails. They own transfer mechanics, byte-range validation, segment execution, safe file writing, proxy-aware transport, and persistence interfaces. `internal/media` owns YouTube probe/download via managed yt-dlp; the HTTP engine is unchanged for ordinary files. Media tasks write into `.odm-partial/<taskID>/` then promote the finished file into the user download folder so concurrent jobs cannot share fragment filenames.

### Desktop (`apps/desktop`)

Wails bootstrap, window lifecycle, and thin service adapters that call engine APIs.

### CLI (`cmd/odm`)

Thin command-line entry point over the same engine APIs.

### Network Access / Tunneling (`internal/access`, future)

Optional dialer/provider behind an interface. Default is nil/disabled. The download transport may accept an alternate dialer or proxy URL only when the user opts in.

## 3. Download lifecycle

Recommended explicit states:

```text
Queued -> Inspecting -> Downloading -> Finalizing -> Completed
                      |       |
                      |       +-> Pausing -> Paused -> Downloading
                      |
                      +-> Retrying -> Inspecting/Downloading
                      |
                      +-> Failed
Queued/Paused/Failed -> Cancelled (depending on user action)
```

User decisions that must not be silent:

- Cancel: keep or delete `.part`
- Remove task: metadata only, or metadata + file
- Resume with changed remote validators: restart, keep partial and abort, or review

## 4. Extensibility without premature abstraction

Decide early:

- Engine free of Wails
- Explicit task state machine
- Partial file + durable metadata contract
- Transport config including proxy mode: `system | manual | off`
- Progress as throttled snapshots/events
- Optional `AccessProvider` interface (may remain unimplemented for several phases)

Postpone safely:

- Per-segment crash-resume orchestration in the manager (table exists; engine still clears failed segmented partials)
- Daemon installer
- Browser native messaging
- Concrete tunnel protocols

## 5. Network Access / Tunneling track

Goals:

- Help users reach hosts that are blocked or unreliable on censored/unstable networks.
- Remain modular, opt-in, and auditable.

Non-goals for the core engine:

- Ordinary downloading must not require a tunnel.
- Do not embed circumvention logic inside Range/resume writers.

Design sketch:

```text
type AccessProvider interface {
    // Start prepares an optional local proxy or dialer after explicit user opt-in.
    Start(ctx context.Context, cfg AccessConfig) (AccessSession, error)
}

type AccessSession interface {
    ProxyURL() *url.URL // or DialContext hook
    Close() error
}
```

Security requirements before implementation:

- Explicit opt-in UI and settings
- User-configurable endpoints
- No silent third-party routing
- Credential protection
- Clear failure modes and safe shutdown
- Threat model document
- Honest limitation and legal/safety notes in user docs

Compare provider approaches during the design spike (local forwarder, pluggable external client, embedded stack). Prefer the smallest maintainable option that preserves engine isolation.
