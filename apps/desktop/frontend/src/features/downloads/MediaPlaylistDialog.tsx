import { useMemo, useState } from "react";
import { Dialog } from "../../components/Dialog";
import type { MediaProbeDTO } from "./downloadTypes";

const MAX_SELECT = 100;

type Props = {
  probe: MediaProbeDTO;
  busy: boolean;
  onCancel: () => void;
  onConfirm: (urls: { url: string; title: string }[]) => Promise<void>;
};

export function MediaPlaylistDialog({
  probe,
  busy,
  onCancel,
  onConfirm,
}: Props) {
  const [selected, setSelected] = useState<Record<string, boolean>>({});
  const [error, setError] = useState<string | null>(null);

  const chosen = useMemo(
    () => probe.entries.filter((e) => selected[e.id || e.url]),
    [probe.entries, selected],
  );

  function toggle(key: string) {
    setSelected((prev) => ({ ...prev, [key]: !prev[key] }));
  }

  function selectAll(on: boolean) {
    const next: Record<string, boolean> = {};
    for (const e of probe.entries.slice(0, MAX_SELECT)) {
      next[e.id || e.url] = on;
    }
    setSelected(next);
  }

  async function submit() {
    setError(null);
    if (chosen.length === 0) {
      setError("Select at least one video");
      return;
    }
    if (chosen.length > MAX_SELECT) {
      setError(`Select at most ${MAX_SELECT} videos`);
      return;
    }
    try {
      await onConfirm(
        chosen.map((e) => ({
          url: e.url,
          title: e.title || e.id,
        })),
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <Dialog
      title="Playlist / channel"
      onClose={busy ? undefined : onCancel}
      footer={
        <>
          <button
            type="button"
            className="rounded-md border border-[var(--color-border)] px-3 py-1.5 text-sm"
            onClick={onCancel}
            disabled={busy}
          >
            Cancel
          </button>
          <button
            type="button"
            className="rounded-md bg-[var(--color-accent)] px-3 py-1.5 text-sm font-medium text-white disabled:opacity-60"
            onClick={() => void submit()}
            disabled={busy}
          >
            {busy ? "Adding…" : `Add ${chosen.length || ""}`}
          </button>
        </>
      }
    >
      <p className="text-sm font-medium">{probe.title || "Playlist"}</p>
      <p className="mt-1 text-xs text-[var(--color-text-muted)]">
        Select videos to enqueue (max {MAX_SELECT}). Each becomes its own
        download with Best format.
      </p>
      <div className="mt-2 flex gap-2 text-xs">
        <button
          type="button"
          className="underline"
          onClick={() => selectAll(true)}
        >
          Select first {Math.min(MAX_SELECT, probe.entries.length)}
        </button>
        <button
          type="button"
          className="underline"
          onClick={() => selectAll(false)}
        >
          Clear
        </button>
      </div>
      <ul className="mt-3 max-h-64 space-y-2 overflow-y-auto text-sm">
        {probe.entries.map((e) => {
          const key = e.id || e.url;
          return (
            <li key={key}>
              <label className="flex items-start gap-2">
                <input
                  type="checkbox"
                  className="mt-1"
                  checked={!!selected[key]}
                  onChange={() => toggle(key)}
                />
                <span className="min-w-0 break-words">{e.title || e.id}</span>
              </label>
            </li>
          );
        })}
      </ul>
      {error ? (
        <p className="mt-3 text-sm text-[var(--color-danger)]" role="alert">
          {error}
        </p>
      ) : null}
    </Dialog>
  );
}
