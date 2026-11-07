# Getting started (end users)

Install and use ODM on Windows without cloning the repo or installing Go/Node.

**Current prep release:** `0.5.0` (v1.0 release candidate). Windows code signing lands with final **1.0.0**.

## 1. Download the installer

1. Open the [GitHub Releases](https://github.com/lostfromlight1/Open-Download-Manager/releases) page.
2. Download `odm-amd64-installer.exe` (and optionally verify the SHA256 in `SHA256SUMS.txt`).

### SmartScreen (unsigned builds)

Until 1.0.0 is Authenticode-signed, Windows may show **Windows protected your PC**:

1. Click **More info**
2. Click **Run anyway**

Only do this for installers you downloaded from the official GitHub Releases page.

## 2. Install

1. Run the installer.
2. It installs **per user** under `%LOCALAPPDATA%\Programs\Open Download Manager` (no admin/UAC required).
3. On the finish page, leave **Launch Open Download Manager** checked (or start it from the Desktop / Start Menu shortcut).

## 3. First-run setup

On first launch ODM walks you through:

1. **Download folder** (required) — where new files are saved.
2. **Browser integration** (optional) — registers the native messaging host for the extension.
3. **YouTube downloads** (optional) — installs managed yt-dlp/ffmpeg helpers.

You can change any of these later in **Settings**.

## 4. Optional: browser extension

1. In ODM Settings, enable **Browser integration** (or do it in the wizard) and save.
2. Chrome/Edge: open `chrome://extensions` → Developer mode → **Load unpacked** → select the `browser-extension` folder from the release zip (or the repo’s `apps/browser-extension`).
3. Copy the extension ID into Settings → **Chrome extension ID** if prompted, then Save.
4. Open the extension popup → confirm status is connected → optionally enable **Send browser downloads to ODM**.
5. YouTube: open a watch page and use the teal **↓ ODM** button (or the context menu). If you reload/update the extension, **refresh the YouTube tab** first — otherwise the button shows “Extension was reloaded”.

Firefox: use `manifest.firefox.json` as `manifest.json` when loading the temporary add-on (see contributor docs for details).

## 5. Optional: YouTube

1. Enable **YouTube downloads** in Settings or the wizard.
2. Keep ODM open while helpers download the first time.
3. Add a YouTube URL → choose format / playlist entries. Only one active/queued download per video is allowed (avoids fragment collisions).

Only download content you have rights to use.

## 6. Everyday use

- **+ Add** for http(s) URLs (and YouTube when enabled).
- Pause / resume / cancel / remove; multi-select for bulk actions.
- Active downloads show a short speed graph (network Mbps + disk write).
- Close to tray keeps downloads running (default on).

## Need to build from source?

See [`SETUP.md`](../SETUP.md) (developers / contributors).
