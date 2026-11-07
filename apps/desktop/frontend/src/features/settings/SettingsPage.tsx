import { useEffect, useState } from "react";
import { Dialogs } from "@wailsio/runtime";
import { getDesktopApi } from "../../services/desktopApi";
import type { SettingsDTO } from "../downloads/downloadTypes";

type Props = {
  onBack: () => void;
  onSaved: (settings: SettingsDTO) => void;
};

export function SettingsPage({ onBack, onSaved }: Props) {
  const [form, setForm] = useState<SettingsDTO | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [savedMsg, setSavedMsg] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void (async () => {
      try {
        const api = await getDesktopApi();
        setForm(await api.getSettings());
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
      }
    })();
  }, []);

  async function save() {
    if (!form) return;
    setBusy(true);
    setError(null);
    setSavedMsg(null);
    const enabling = form.browserIntegrationEnabled;
    const wasEnabled = form.browserHostRegistered || false;
    try {
      const api = await getDesktopApi();
      const saved = await api.updateSettings(form);
      setForm({ ...saved, browserHostMessage: saved.browserHostMessage || "" });
      onSaved(saved);
      if (saved.mediaToolsMessage?.trim() && form.mediaEnabled) {
        setSavedMsg(saved.mediaToolsMessage.trim());
      } else if (saved.browserHostMessage?.trim()) {
        setSavedMsg(saved.browserHostMessage.trim());
      } else if (enabling) {
        setSavedMsg(
          "Settings saved. Browser host should now be registered for this user.",
        );
      } else if (wasEnabled && !enabling) {
        setSavedMsg("Settings saved. Browser host uninstalled for this user.");
      } else {
        setSavedMsg("Settings saved.");
      }
    } catch (e) {
      const message = e instanceof Error ? e.message : String(e);
      setError(message);
      // Settings may still have been persisted; refresh so host status is accurate.
      try {
        const api = await getDesktopApi();
        const latest = await api.getSettings();
        setForm(latest);
        onSaved(latest);
      } catch {
        // keep the primary error
      }
    } finally {
      setBusy(false);
    }
  }

  async function retryMediaTools() {
    if (!form?.mediaEnabled) return;
    setBusy(true);
    setError(null);
    setSavedMsg(null);
    try {
      const api = await getDesktopApi();
      // Re-save kicks a background EnsureTools that resumes partial ffmpeg downloads.
      const saved = await api.updateSettings({ ...form, mediaEnabled: true });
      setForm(saved);
      onSaved(saved);
      setSavedMsg(
        saved.mediaToolsMessage ||
          "Helper install started — keep ODM open until ffmpeg finishes (~190MB).",
      );
      // Poll a few times while the background download runs.
      for (let i = 0; i < 15; i++) {
        await new Promise((r) => setTimeout(r, 2000));
        const latest = await api.getSettings();
        setForm(latest);
        onSaved(latest);
        if (latest.mediaToolsReady) {
          setSavedMsg(latest.mediaToolsMessage || "Helpers ready.");
          break;
        }
        if (latest.mediaToolsMessage?.includes("Helper install issue")) {
          setError(latest.mediaToolsMessage);
          break;
        }
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function browseFolder() {
    if (!form) return;
    try {
      const selected = await Dialogs.OpenFile({
        CanChooseDirectories: true,
        CanChooseFiles: false,
        Title: "Default download folder",
        Directory: form.downloadDirectory || undefined,
      });
      if (typeof selected === "string" && selected.trim()) {
        setForm({ ...form, downloadDirectory: selected });
      }
    } catch {
      // Dialog unavailable in browser-only preview; keep typing the path.
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6 px-6 py-6">
      <header className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-[var(--color-text)]">
            Settings
          </h1>
          <p className="text-sm text-[var(--color-text-muted)]">
            Destination, appearance, concurrency, and proxy
          </p>
        </div>
        <button
          type="button"
          className="rounded-md border border-[var(--color-border)] px-3 py-2 text-sm"
          onClick={onBack}
        >
          Back
        </button>
      </header>

      {!form && !error ? (
        <p className="text-sm text-[var(--color-text-muted)]">Loading…</p>
      ) : null}
      {error ? (
        <p className="text-sm text-[var(--color-danger)]" role="alert">
          {error}
        </p>
      ) : null}

      {form ? (
        <form
          className="space-y-6 rounded-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] p-5"
          onSubmit={(e) => {
            e.preventDefault();
            void save();
          }}
        >
          <fieldset className="space-y-3">
            <legend className="text-sm font-semibold">Downloads</legend>
            <label className="block text-sm">
              Default download location
              <div className="mt-1 flex flex-col gap-2 min-[700px]:flex-row">
                <input
                  className="min-w-0 w-full rounded-md border border-[var(--color-border)] px-3 py-2 font-mono text-sm"
                  value={form.downloadDirectory}
                  onChange={(e) =>
                    setForm({ ...form, downloadDirectory: e.target.value })
                  }
                />
                <button
                  type="button"
                  className="shrink-0 rounded-md border border-[var(--color-border)] px-3 py-2 text-sm"
                  onClick={() => void browseFolder()}
                >
                  Browse
                </button>
              </div>
            </label>
            <label className="block text-sm">
              Max active downloads
              <input
                type="number"
                min={1}
                className="mt-1 w-32 rounded-md border border-[var(--color-border)] px-3 py-2 text-sm"
                value={form.maxActiveDownloads}
                onChange={(e) =>
                  setForm({
                    ...form,
                    maxActiveDownloads: Number(e.target.value) || 1,
                  })
                }
              />
            </label>
            <label className="block text-sm">
              Max connections per download
              <input
                type="number"
                min={1}
                className="mt-1 w-32 rounded-md border border-[var(--color-border)] px-3 py-2 text-sm"
                value={form.maxConnectionsPerDownload}
                onChange={(e) =>
                  setForm({
                    ...form,
                    maxConnectionsPerDownload: Number(e.target.value) || 1,
                  })
                }
              />
            </label>
          </fieldset>

          <fieldset className="space-y-3">
            <legend className="text-sm font-semibold">Appearance</legend>
            <div className="space-y-2 text-sm">
              {(
                [
                  ["system", "System"],
                  ["light", "Light"],
                  ["dark", "Dark"],
                ] as const
              ).map(([value, label]) => (
                <label key={value} className="flex items-center gap-2">
                  <input
                    type="radio"
                    name="theme"
                    checked={form.theme === value}
                    onChange={() => setForm({ ...form, theme: value })}
                  />
                  {label}
                </label>
              ))}
            </div>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.notifyOnComplete}
                onChange={(e) =>
                  setForm({ ...form, notifyOnComplete: e.target.checked })
                }
              />
              Notify when a download completes (opt-in)
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.closeToTray}
                onChange={(e) =>
                  setForm({ ...form, closeToTray: e.target.checked })
                }
              />
              Keep downloading in the system tray when I close the window
            </label>
            <p className="text-sm text-[var(--color-text-muted)]">
              When enabled, closing the window hides ODM to the tray; downloads
              keep running. Use Quit in the tray menu to exit fully.
            </p>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.launchAtLogin}
                onChange={(e) =>
                  setForm({ ...form, launchAtLogin: e.target.checked })
                }
              />
              Launch Open Download Manager when I sign in to Windows
            </label>
            <p className="text-sm text-[var(--color-text-muted)]">
              Starts hidden to the tray (with --hidden). Turn off and Save to
              remove the login entry.
            </p>
          </fieldset>

          <fieldset className="space-y-3">
            <legend className="text-sm font-semibold">YouTube downloads</legend>
            <p className="text-sm text-[var(--color-text-muted)]">
              Opt-in. Uses managed yt-dlp/ffmpeg helpers under your ODM data
              folder. You are responsible for complying with YouTube&apos;s
              terms and only downloading content you have rights to. A default
              download folder is required.
            </p>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.mediaEnabled}
                onChange={(e) =>
                  setForm({ ...form, mediaEnabled: e.target.checked })
                }
              />
              Enable YouTube downloads
            </label>
            {form.mediaToolsMessage ? (
              <p className="text-sm text-[var(--color-text-muted)]" role="status">
                {form.mediaToolsMessage}
              </p>
            ) : null}
            {form.mediaEnabled && !form.mediaToolsReady ? (
              <button
                type="button"
                className="rounded-md border border-[var(--color-border)] px-3 py-1.5 text-sm"
                disabled={busy}
                onClick={() => void retryMediaTools()}
              >
                {busy ? "Working…" : "Retry helper install"}
              </button>
            ) : null}
          </fieldset>

          <fieldset className="space-y-3">
            <legend className="text-sm font-semibold">
              Browser integration
            </legend>
            <p className="text-sm text-[var(--color-text-muted)]">
              Opt-in only. The extension may send http(s) download links to ODM
              via native messaging. It does not read browsing history or page
              content. A default download folder is required when enabled. To
              replace the browser&apos;s own downloads with ODM, enable{" "}
              <span className="font-medium">Send browser downloads to ODM</span>{" "}
              in the extension popup (not here).
            </p>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.browserIntegrationEnabled}
                onChange={(e) =>
                  setForm({
                    ...form,
                    browserIntegrationEnabled: e.target.checked,
                  })
                }
              />
              Enable browser integration
            </label>
            <label className="block text-sm">
              Chrome / Edge extension ID (from chrome://extensions)
              <input
                className="mt-1 w-full rounded-md border border-[var(--color-border)] px-3 py-2 font-mono text-sm"
                value={form.chromeExtensionId}
                onChange={(e) =>
                  setForm({ ...form, chromeExtensionId: e.target.value })
                }
                placeholder="abcdefghijklmnopqrstuvwxyzabcdef"
                disabled={!form.browserIntegrationEnabled}
              />
            </label>
            {form.browserIntegrationEnabled &&
            !form.chromeExtensionId.trim() ? (
              <p className="text-sm text-[var(--color-text-muted)]">
                Firefox works without this ID. Chrome/Edge need the ID so the
                host can allow your unpacked extension.
              </p>
            ) : null}
            {form.browserHostRegistered ? (
              <p className="text-sm text-[var(--color-success)]" role="status">
                Status: browser host is installed for this user. Saving with
                integration enabled reinstalls/updates the host; turning it off
                and saving uninstalls the host.
              </p>
            ) : form.browserIntegrationEnabled ? (
              <p
                className="text-sm text-[var(--color-text-muted)]"
                role="status"
              >
                Status: host not registered yet. Build{" "}
                <code className="font-mono text-xs">odm-nmhost</code> next to
                the app (or on PATH), then Save to install it.
              </p>
            ) : (
              <p
                className="text-sm text-[var(--color-text-muted)]"
                role="status"
              >
                Status: browser host is not installed. Enable integration and
                Save to install; disable and Save to uninstall.
              </p>
            )}
          </fieldset>

          <fieldset className="space-y-3">
            <legend className="text-sm font-semibold">Network</legend>
            <div className="space-y-2 text-sm">
              {(
                [
                  ["system", "System Proxy"],
                  ["manual", "Manual Proxy"],
                  ["off", "No Proxy"],
                ] as const
              ).map(([value, label]) => (
                <label key={value} className="flex items-center gap-2">
                  <input
                    type="radio"
                    name="proxy"
                    checked={form.proxyMode === value}
                    onChange={() => setForm({ ...form, proxyMode: value })}
                  />
                  {label}
                </label>
              ))}
            </div>
            {form.proxyMode === "manual" ? (
              <label className="block text-sm">
                Proxy address
                <input
                  className="mt-1 w-full rounded-md border border-[var(--color-border)] px-3 py-2 font-mono text-sm"
                  value={form.proxyURL}
                  onChange={(e) =>
                    setForm({ ...form, proxyURL: e.target.value })
                  }
                  placeholder="http://127.0.0.1:8080"
                />
              </label>
            ) : null}
          </fieldset>

          <div className="flex items-center gap-3">
            <button
              type="submit"
              className="rounded-md bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
              disabled={busy}
            >
              {busy ? "Saving…" : "Save Settings"}
            </button>
            {savedMsg ? (
              <p
                className="max-w-xl text-sm text-[var(--color-success)]"
                role="status"
              >
                {savedMsg}
              </p>
            ) : null}
          </div>
        </form>
      ) : null}
    </div>
  );
}
