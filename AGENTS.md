# Instructions for AI coding agents

These instructions apply to the entire repository. More specific instructions in subdirectories may add constraints but must not contradict this file.

## Before changing code

1. Read `README.md` and the relevant documents under `docs/`.
2. Inspect the existing code before proposing or making changes.
3. Identify the smallest coherent implementation slice.
4. Do not invent APIs, dependencies, files, or completed functionality.
5. If a requirement is ambiguous and materially affects architecture or data safety, ask before implementing.

## Engineering rules

- Go is the source of truth for download state, filesystem operations, scheduling, and persistence.
- React/TypeScript is the presentation layer. Do not implement a second download engine in the frontend.
- Keep the download engine independent of Wails so the CLI and a future background service can reuse it.
- Prefer the Go standard library for HTTP and concurrency unless a dependency is clearly justified.
- Use typed errors and explicit context cancellation. Avoid panics in production paths; tests may use them.
- Validate all user-provided URLs, paths, filenames, headers, and browser messages.
- Never trust `Content-Length`, `Content-Disposition`, `Accept-Ranges`, or browser-supplied metadata without validation.
- Never write downloaded data outside the user-selected destination.
- Do not disable TLS certificate validation.
- Do not log credentials, cookies, authorization headers, signed URLs, or full sensitive query strings.
- Use parameterized SQL queries and versioned migrations.
- Avoid shell execution for download operations.
- Do not add dependencies without explaining why and checking maintenance, license, and platform support.
- Prefer stable Go and pinned compatible dependency ranges; commit `go.sum`.
- Keep platform-specific code behind small, documented abstractions.
- Optional Network Access / Tunneling must live behind an explicit opt-in interface and must not be required for ordinary downloads.

## Download correctness requirements

- Treat HTTP Range support as verified only after receiving and validating a `206 Partial Content` response with a correct `Content-Range`.
- If range requests are unsupported or inconsistent, safely fall back to a single-stream download.
- Validate `Content-Range` boundaries and total length before writing a segment.
- Prevent concurrent workers from writing overlapping byte ranges.
- Persist enough metadata to recover safely after a crash.
- Resume only when validators or equivalent checks indicate that the remote representation is still compatible. If compatibility cannot be confirmed, ask the user rather than silently discarding progress.
- Handle redirects, authentication failures, expired URLs, 404/410, 429, 5xx, timeouts, stalls, disk-full, permission errors, and cancellation explicitly.
- Do not mark a task completed until all expected bytes are present and finalization succeeds.

## Frontend rules

- Use TypeScript strict mode.
- Keep UI components focused and accessible.
- Represent download lifecycle states explicitly; avoid ambiguous boolean combinations.
- Do not put networking or filesystem business logic in React components.
- Handle loading, empty, error, and offline states.
- Do not show fake progress or fabricated download speeds.

## Workflow

- Add or update tests for every behavior change.
- Run formatting, linting, type checks, unit tests, and relevant integration tests.
- Update docs when behavior, architecture, configuration, or user-facing features change.
- Keep pull requests focused.
- Use conventional commit messages where practical.
- Do not silently change public interfaces or database schemas.

## Definition of done

A change is done only when:
- The implementation matches the relevant requirement.
- Tests cover normal and failure paths.
- Errors are actionable and do not leak secrets.
- Formatting, linting, and type checks pass.
- Relevant docs are updated.
- Cross-platform implications are considered.
