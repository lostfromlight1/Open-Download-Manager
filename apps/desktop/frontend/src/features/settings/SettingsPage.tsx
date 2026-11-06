import { useEffect, useState } from "react";
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
    try {
      const api = await getDesktopApi();
      const saved = await api.updateSettings(form);
      setForm(saved);
      onSaved(saved);
      setSavedMsg("Settings saved. New downloads use these values.");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
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
            Destination, concurrency, and proxy
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
              <input
                className="mt-1 w-full rounded-md border border-[var(--color-border)] px-3 py-2 font-mono text-sm"
                value={form.downloadDirectory}
                onChange={(e) =>
                  setForm({ ...form, downloadDirectory: e.target.value })
                }
              />
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
              <p className="text-sm text-[var(--color-success)]">{savedMsg}</p>
            ) : null}
          </div>
        </form>
      ) : null}
    </div>
  );
}
