import { useState } from "react";
import { Dialog } from "../../components/Dialog";
import type { MediaFormatDTO, MediaProbeDTO } from "./downloadTypes";

type Props = {
  probe: MediaProbeDTO;
  busy: boolean;
  onCancel: () => void;
  onConfirm: (formatId: string) => Promise<void>;
};

export function MediaFormatDialog({
  probe,
  busy,
  onCancel,
  onConfirm,
}: Props) {
  const [formatId, setFormatId] = useState(
    probe.formats[0]?.id ?? "best",
  );
  const [error, setError] = useState<string | null>(null);

  async function submit() {
    setError(null);
    try {
      await onConfirm(formatId);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <Dialog
      title="YouTube format"
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
            {busy ? "Adding…" : "Download"}
          </button>
        </>
      }
    >
      <p className="text-sm font-medium text-[var(--color-text)]">
        {probe.title || probe.webpageUrl}
      </p>
      <p className="mt-2 text-xs text-[var(--color-text-muted)]">
        Prefer Best or a height labeled “merges audio”. Picking a video-only
        stream alone used to produce silent files — ODM now merges best audio
        automatically (needs ffmpeg).
      </p>
      <label className="mt-3 block text-sm">
        Format
        <select
          className="mt-1 w-full rounded-md border border-[var(--color-border)] px-3 py-2 text-sm"
          value={formatId}
          onChange={(e) => setFormatId(e.target.value)}
        >
          {probe.formats.map((f: MediaFormatDTO) => (
            <option key={f.id} value={f.id}>
              {formatLabel(f)}
            </option>
          ))}
        </select>
      </label>
      {error ? (
        <p className="mt-3 text-sm text-[var(--color-danger)]" role="alert">
          {error}
        </p>
      ) : null}
    </Dialog>
  );
}

function formatLabel(f: MediaFormatDTO): string {
  const bits = [f.id];
  if (f.note) bits.push(f.note);
  else if (f.resolution) bits.push(f.resolution);
  if (f.ext) bits.push(f.ext);
  if (f.isAudio) bits.push("audio only");
  else if (f.videoOnly) bits.push("merges audio");
  return bits.join(" · ");
}
