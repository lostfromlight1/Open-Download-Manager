# Setup guide (new device)

This guide gets ODM running on a clean machine. Stack: **Go + Wails v3** desktop shell + `odm` CLI with single-stream and segmented HTTP downloads.

Official Wails install notes: [Wails v3 installation](https://v3.wails.io/getting-started/installation/).

---

## 1. What you need

| Tool | Version | Why |
|---|---|---|
| Git | any recent | Clone the repo |
| Go | **1.26+** (1.27.x is fine) | Backend, CLI, Wails |
| Node.js | **22+** | Frontend toolchain |
| pnpm | **10.20.0** | Workspace installs |
| Wails CLI | **v3.1.0** | Desktop `dev` / `build` / `package` |
| Platform WebView deps | see below | Native desktop window |
| NSIS (`makensis`) | 3.x | Windows installer packaging only |

Module path:

```text
github.com/lostfromlight1/Open-Download-Manager
```

---

## 2. Install prerequisites

### 2.1 Git

Install Git for your OS, then open a new terminal.

### 2.2 Go

1. Install Go from https://go.dev/dl/ (1.26 or newer).
2. Confirm `~/go/bin` (Windows: `%USERPROFILE%\go\bin`) is on your `PATH`.
3. Check:

```bash
go version
```

### 2.3 Node.js and pnpm

1. Install Node.js 22+ from https://nodejs.org/
2. Enable pnpm via Corepack:

```bash
corepack enable
corepack prepare pnpm@10.20.0 --activate
pnpm --version
```

If Corepack is unavailable, `npm install -g pnpm@10.20.0` also works.

### 2.4 Wails CLI (pin v3.1.0)

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
wails3 version
```

Expected: `v3.1.0`.

If `wails3` is not found, add Go’s bin directory to `PATH` and open a new terminal.

### 2.5 Platform dependencies

#### Windows

- **WebView2 Runtime** — usually already installed on Windows 10 (1803+) / Windows 11.
- Optional: run `wails3 doctor` after installing the CLI.
- For `wails3 build` in non-interactive shells (CI / some terminals), set `CI=true` so pnpm does not abort.
- **NSIS** (for installers only): install [NSIS 3](https://nsis.sourceforge.io/) and ensure `makensis` is on `PATH` (default install path: `C:\Program Files (x86)\NSIS`). winget: `winget install NSIS.NSIS`.

#### macOS

```bash
xcode-select --install
```

Minimum deployment target for the desktop build is **macOS 12**.

#### Linux

Install GTK4 / WebKitGTK and build tools as shown by:

```bash
wails3 doctor
```

Ubuntu 24.04+ / Debian 13+ (GTK4 + WebKitGTK 6.0):

```bash
sudo apt update
sudo apt install build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev
```

Distros differ; follow the doctor output.

---

## 3. Clone and install

```bash
git clone https://github.com/lostfromlight1/Open-Download-Manager.git
cd Open-Download-Manager

pnpm install
pnpm build
```

`pnpm build` generates Wails TypeScript bindings and builds the frontend into `apps/desktop/frontend/dist`. That directory is required before `go test ./...` because the desktop package embeds it.

---

## 4. Verify the install

From the repo root:

```bash
go test ./...
pnpm test
pnpm lint
pnpm typecheck
go run ./cmd/odm version
```

Expected CLI output:

```text
ODM — Open Download Manager 0.4.0
```

---

## 5. How to run

### 5.1 Desktop app (recommended for UI work)

From `apps/desktop`:

```bash
cd apps/desktop
wails3 dev
```

Or from the repo root:

```bash
pnpm --dir apps/desktop exec wails3 dev -config ./build/config.yml
```

This starts the Vite frontend and the native Wails window with hot reload.

The desktop dashboard should list downloads from the shared ODM database (add one via the UI or `odm task add`).

### 5.2 Frontend only (browser preview, no native window)

```bash
pnpm dev:web
```

Useful for UI/layout checks. This is not the full desktop app.

### 5.3 CLI

```bash
go run ./cmd/odm version
go run ./cmd/odm download https://example.com/file.bin -o file.bin --proxy off
go run ./cmd/odm download https://example.com/file.bin -o file.bin --connections 4
go run ./cmd/odm task add https://example.com/file.bin -o file.bin
go run ./cmd/odm task list
go run ./cmd/odm settings show
```

- `odm download`: one-shot transfer (no database). Progress on stderr; `saved ...` on stdout.
- `--connections N`: try up to N parallel Range workers after a verified 206 probe; falls back to single-stream when Range is unsupported or the file is small.
- `odm task add`: enqueue into SQLite and run until that task finishes (default `--start`).
- Database default location: OS config dir `/odm` (override with `--data-dir` or `ODM_DATA_DIR`).
- Cancel/remove file choice: `--files keep|delete`.
- Resume mismatch policy for `queue run`: `--on-mismatch abort|restart`.
- `odm settings set download.max_connections 4`: per-download worker cap for the queue.
- `odm settings set ui.close_to_tray true|false`: when true (default), closing the desktop window hides to the system tray and keeps downloads running; tray **Quit** exits fully and stops the queue.
- `odm settings set ui.launch_at_login true|false`: when true, the desktop app registers OS login autostart (Wails; Windows verified) with `--hidden` (starts to tray).

### 5.4 Production desktop binary

```bash
cd apps/desktop
```

PowerShell (Windows):

```powershell
$env:CI = "true"
wails3 build
```

bash:

```bash
CI=true wails3 build
```

Output binary: `apps/desktop/bin/odm.exe` (Windows) or `apps/desktop/bin/odm` (macOS/Linux).

You can also compile without the Wails task runner:

```bash
# after pnpm build
go build -o bin/odm ./cmd/odm
go build -o bin/odm-desktop ./apps/desktop
```

### 5.5 Unsigned installers / packages (Phase 5)

All packages are **unsigned**. Code signing and macOS notarization are deferred to v1.0.

From `apps/desktop` with `CI=true`:

| OS | Command | Artifact |
|---|---|---|
| Windows | `wails3 task windows:package` | `bin/odm-amd64-installer.exe` (NSIS) |
| macOS | `wails3 task darwin:package` | `bin/odm.app` |
| Linux | `wails3 task linux:create:deb` | `bin/odm_*.deb` |

Windows packaging requires `makensis` on `PATH`. The NSIS task downloads a WebView2 bootstrapper into `build/windows/nsis/` (gitignored).

CI uploads these artifacts from the `desktop-package` job (7-day retention). Local Windows smoke steps are in [`docs/TESTING.md`](docs/TESTING.md).

Shared SQLite/settings live under the OS config dir `/odm` (Windows: `%AppData%\odm`). Uninstalling the Windows NSIS package does **not** delete that data directory.

### 5.6 Browser extension (Phase 6)

1. Build the native host next to the desktop app (or put it on `PATH`):

```powershell
go build -o apps/desktop/bin/odm-nmhost.exe ./cmd/odm-nmhost
```

```bash
go build -o apps/desktop/bin/odm-nmhost ./cmd/odm-nmhost
```

2. In the desktop app: Settings → set default download folder → enable **Browser integration** → for Chrome/Edge paste the extension ID from `chrome://extensions` → Save (registers the host).
3. Load the unpacked extension from `apps/browser-extension`.
   - Chrome/Edge: Developer mode → Load unpacked → select that folder (`manifest.json`).
   - Firefox: use `manifest.firefox.json` (copy over `manifest.json` or load via `about:debugging` temporary add-on).
4. Open the extension popup and click **Refresh status**. Right-click a link → **Download with ODM**.
5. Optional: enable **Send browser downloads to ODM** in the popup (requests the Downloads permission). Normal browser file downloads are then enqueued into ODM and cancelled in the browser when enqueue succeeds. HTML page bodies and `blob:`/`data:` URLs are skipped.
6. Optional: enable **Forward session cookies for intercepted downloads** (Cookies permission). Cookie/Referer are one-shot and not stored by ODM.

CLI: `odm settings set browser.integration_enabled true` (still register the host via desktop Save, or call install helpers after building `odm-nmhost`).

Optional click capture is off by default and asks for http(s) page access when enabled. Extension unit tests: `cd apps/browser-extension && npm test` (or `node --test intercept.test.js`).

---

## 6. Day-to-day commands

| Goal | Command |
|---|---|
| Install JS deps | `pnpm install` |
| Build frontend (+ bindings) | `pnpm build` |
| Frontend unit tests | `pnpm test` |
| Frontend lint / types / format | `pnpm lint` / `pnpm typecheck` / `pnpm format` |
| Go tests | `go test ./...` or `pnpm test:go` |
| Desktop dev | `cd apps/desktop && wails3 dev` |
| Desktop release build | `cd apps/desktop` then `CI=true wails3 build` |
| Windows NSIS installer | `cd apps/desktop` then `CI=true wails3 task windows:package` |
| macOS `.app` | `cd apps/desktop` then `CI=true wails3 task darwin:package` |
| Linux `.deb` | `cd apps/desktop` then `CI=true wails3 task linux:create:deb` |
| CLI version | `go run ./cmd/odm version` |
| Native messaging host | `go build -o apps/desktop/bin/odm-nmhost.exe ./cmd/odm-nmhost` |

---

## 7. Troubleshooting

### `wails3: command not found`

- Confirm `go install ...@v3.1.0` succeeded.
- Ensure `%USERPROFILE%\go\bin` or `$HOME/go/bin` is on `PATH`.
- Restart the terminal (and IDE) after changing `PATH`.

### `Event bindings module not found` during Vite build

Bindings are generated by Wails. Run:

```bash
pnpm build
```

or from `apps/desktop`:

```bash
wails3 generate bindings -ts -i -clean=true .
```

### `go test` fails on embed / missing `frontend/dist`

Run `pnpm build` first so `apps/desktop/frontend/dist` exists.

### `pnpm install` aborted with `ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY`

Set `CI=true` (common when `wails3 build` runs pnpm through Task on Windows).

### Desktop window does not open

- Run `wails3 doctor` and install anything it reports missing.
- On Windows, confirm WebView2 is installed.
- On Linux, confirm WebKitGTK / GTK packages from the doctor output.

### `makensis: command not found` when packaging Windows

Install NSIS 3 and add it to `PATH` (see §2.5). Confirm with `makensis /VERSION`.

### Wrong Go or Wails version

```bash
go version
wails3 version
```

Use Go ≥ 1.26 and Wails **v3.1.0** (not a nightly).

---

## 8. What works today vs later

**Works now**

- Clone → install → test
- Desktop app via `wails3 dev` / `wails3 build` (dashboard, settings, live progress, keep/delete and mismatch dialogs)
- Close-to-tray (default on): hide window to system tray; downloads keep running until tray Quit
- Launch-at-login (opt-in) and richer tray (speed + pause/resume)
- Unsigned packaging: Windows NSIS, macOS `.app`, Linux `.deb` (CI `desktop-package`)
- Opt-in browser extension + `odm-nmhost` native messaging (context-menu enqueue)
- Opt-in browser download interception + optional cookie/Referer handoff (memory-only)
- `odm version` and `odm download` (single-stream + segmented HTTP/HTTPS, resume, retry, proxy modes)
- `odm task` / `odm queue` / `odm settings` (SQLite queue shared with desktop, concurrency, connections, keep/delete, mismatch policy, proxy persistence, close-to-tray)

**Not implemented yet (later phases)**

- Interactive TTY prompts (CLI currently uses explicit flags for decisions)
- Per-segment crash resume wired through the manager (segment rows exist; engine clears failed segmented `.part`s for safety)
- Signed / notarized releases (v1.0)
- Separate headless background service (beyond close-to-tray + launch-at-login in the desktop process)
- Authorization-header / secret-store authenticated downloads

See [`docs/IMPLEMENTATION_PLAN.md`](docs/IMPLEMENTATION_PLAN.md) for the roadmap.
