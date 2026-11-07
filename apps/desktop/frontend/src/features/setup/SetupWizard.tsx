import { useState } from "react";
import { Dialogs } from "@wailsio/runtime";
import { getDesktopApi } from "../../services/desktopApi";
import type { SettingsDTO } from "../downloads/downloadTypes";

type Props = {
  initial: SettingsDTO;
  onComplete: (settings: SettingsDTO) => void;
};

type Step = "folder" | "browser" | "youtube";

/**
 * Shown on first launch when no download directory is configured.
 * Folder is required; browser integration and YouTube are optional.
 */
export function SetupWizard({ initial, onComplete }: Props) {
  const [step, setStep] = useState<Step>("folder");
  const [form, setForm] = useState<SettingsDTO>(initial);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);

  async function browseFolder() {
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

  async function persist(next: SettingsDTO): Promise<SettingsDTO> {
    const api = await getDesktopApi();
    return api.updateSettings(next);
  }

  async function continueFromFolder() {
    const dir = form.downloadDirectory.trim();
    if (!dir) {
      setError("Choose a download folder to continue.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const saved = await persist({ ...form, downloadDirectory: dir });
      setForm(saved);
      setStep("browser");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function continueFromBrowser(enable: boolean) {
    setBusy(true);
    setError(null);
    setStatus(null);
    try {
      const saved = await persist({
        ...form,
        browserIntegrationEnabled: enable,
      });
      setForm(saved);
      if (enable && saved.browserHostMessage?.trim()) {
        setStatus(saved.browserHostMessage.trim());
      }
      setStep("youtube");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function finish(enableYouTube: boolean) {
    setBusy(true);
    setError(null);
    setStatus(null);
    try {
      let saved = await persist({ ...form, mediaEnabled: enableYouTube });
      if (enableYouTube) {
        const api = await getDesktopApi();
        saved = await api.ensureMediaTools();
        setStatus(
          saved.mediaToolsMessage?.trim() ||
            "Downloading helpers in the background — keep ODM open.",
        );
      }
      setForm(saved);
      onComplete(saved);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-lg flex-col justify-center gap-5 px-6 py-10">
      <header>
        <p className="text-sm font-medium text-[var(--color-text-muted)]">
          Welcome
        </p>
        <h1 className="text-2xl font-semibold tracking-tight text-[var(--color-text)]">
          Set up Open Download Manager
        </h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          A few quick choices so downloads land where you want them.
        </p>
      </header>

      <ol className="flex gap-2 text-xs text-[var(--color-text-muted)]" aria-label="Setup steps">
        {(
          [
            ["folder", "1. Folder"],
            ["browser", "2. Browser"],
            ["youtube", "3. YouTube"],
          ] as const
        ).map(([id, label]) => (
          <li
            key={id}
            className={`rounded-md px-2 py-1 ${
              step === id
                ? "bg-[var(--color-surface-muted)] font-medium text-[var(--color-text)]"
                : ""
            }`}
          >
            {label}
          </li>
        ))}
      </ol>

      <section className="rounded-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] p-5">
        {step === "folder" ? (
          <div className="space-y-4">
            <div>
              <h2 className="text-sm font-semibold text-[var(--color-text)]">
                Download folder
              </h2>
              <p className="mt-1 text-sm text-[var(--color-text-muted)]">
                Required. New downloads go here unless you pick another path per
                task.
              </p>
            </div>
            <label className="block text-sm">
              Folder path
              <div className="mt-1 flex flex-col gap-2 min-[500px]:flex-row">
                <input
                  className="min-w-0 w-full rounded-md border border-[var(--color-border)] px-3 py-2 font-mono text-sm"
                  value={form.downloadDirectory}
                  onChange={(e) =>
                    setForm({ ...form, downloadDirectory: e.target.value })
                  }
                  placeholder="D:\Download"
                  aria-required
                />
                <button
                  type="button"
                  className="shrink-0 rounded-md border border-[var(--color-border)] px-3 py-2 text-sm"
                  onClick={() => void browseFolder()}
                  disabled={busy}
                >
                  Browse
                </button>
              </div>
            </label>
            <button
              type="button"
              className="rounded-md bg-[var(--color-accent)] px-3 py-2 text-sm font-medium text-white disabled:opacity-60"
              disabled={busy}
              onClick={() => void continueFromFolder()}
            >
              {busy ? "Saving…" : "Continue"}
            </button>
          </div>
        ) : null}

        {step === "browser" ? (
          <div className="space-y-4">
            <div>
              <h2 className="text-sm font-semibold text-[var(--color-text)]">
                Browser integration
              </h2>
              <p className="mt-1 text-sm text-[var(--color-text-muted)]">
                Optional. Registers a native messaging host so the Chrome/Edge
                or Firefox extension can send links to ODM. You can enable this
                later in Settings.
              </p>
              <p className="mt-2 text-xs text-[var(--color-text-muted)]">
                After enabling: load the unpacked extension from the release’s
                <code className="mx-1 rounded bg-[var(--color-surface-muted)] px-1">
                  browser-extension
                </code>
                folder, then paste the extension ID in Settings if Chrome/Edge
                asks for it.
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                className="rounded-md bg-[var(--color-accent)] px-3 py-2 text-sm font-medium text-white disabled:opacity-60"
                disabled={busy}
                onClick={() => void continueFromBrowser(true)}
              >
                {busy ? "Saving…" : "Enable browser integration"}
              </button>
              <button
                type="button"
                className="rounded-md border border-[var(--color-border)] px-3 py-2 text-sm disabled:opacity-60"
                disabled={busy}
                onClick={() => void continueFromBrowser(false)}
              >
                Skip for now
              </button>
            </div>
          </div>
        ) : null}

        {step === "youtube" ? (
          <div className="space-y-4">
            <div>
              <h2 className="text-sm font-semibold text-[var(--color-text)]">
                YouTube downloads
              </h2>
              <p className="mt-1 text-sm text-[var(--color-text-muted)]">
                Optional. Downloads managed yt-dlp/ffmpeg helpers into app data.
                Only use for content you have rights to download. You can change
                this later in Settings.
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                className="rounded-md bg-[var(--color-accent)] px-3 py-2 text-sm font-medium text-white disabled:opacity-60"
                disabled={busy}
                onClick={() => void finish(true)}
              >
                {busy ? "Working…" : "Enable YouTube downloads"}
              </button>
              <button
                type="button"
                className="rounded-md border border-[var(--color-border)] px-3 py-2 text-sm disabled:opacity-60"
                disabled={busy}
                onClick={() => void finish(false)}
              >
                Skip and finish
              </button>
            </div>
          </div>
        ) : null}

        {status ? (
          <p className="mt-4 text-sm text-[var(--color-text-muted)]">{status}</p>
        ) : null}
        {error ? (
          <p className="mt-4 text-sm text-[var(--color-danger)]" role="alert">
            {error}
          </p>
        ) : null}
      </section>
    </main>
  );
}

