import { ProgressBar } from "../../components/ProgressBar";
import { StatusBadge } from "../../components/StatusBadge";
import {
  displayProgressPercent,
  etaSeconds,
  formatBytes,
  formatEta,
  formatSpeed,
  hostnameFromUrl,
  isProgressExhaustedWhileActive,
} from "../../lib/format";
import type { DownloadDTO } from "./downloadTypes";
import { isActiveStatus } from "./downloadTypes";
import { SpeedGraph, type SpeedSample } from "./SpeedGraph";

type Props = {
  download: DownloadDTO;
  busyId: string | null;
  selected: boolean;
  onToggleSelect: (id: string) => void;
  speedSamples: SpeedSample[];
  onPause: (id: string) => void;
  onResume: (id: string) => void;
  onCancel: (id: string) => void;
  onRemove: (id: string) => void;
  onDetails: (id: string) => void;
};

export function DownloadRow({
  download: d,
  busyId,
  selected,
  onToggleSelect,
  speedSamples,
  onPause,
  onResume,
  onCancel,
  onRemove,
  onDetails,
}: Props) {
  const busy = busyId === d.id || busyId === "bulk";
  const pct = displayProgressPercent(
    d.downloadedBytes,
    d.totalBytes,
    d.status,
  );
  const exhausted = isProgressExhaustedWhileActive(
    d.downloadedBytes,
    d.totalBytes,
    d.status,
  );
  const eta = exhausted
    ? null
    : etaSeconds(
        d.downloadedBytes,
        d.totalBytes,
        d.bytesPerSecond ?? null,
      );
  const sizeLabel = exhausted
    ? `${formatBytes(d.downloadedBytes)} · finishing…`
    : d.totalBytes != null
      ? `${formatBytes(d.downloadedBytes)} / ${formatBytes(d.totalBytes)}`
      : formatBytes(d.downloadedBytes);

  const canPause = isActiveStatus(d.status) || d.status === "queued";
  const canResume =
    d.status === "paused" || d.status === "failed" || d.status === "cancelled";
  const canCancel =
    isActiveStatus(d.status) || d.status === "queued" || d.status === "paused";
  const showGraph = isActiveStatus(d.status);

  return (
    <article
      className={`min-w-0 overflow-hidden rounded-[10px] border bg-[var(--color-surface)] p-4 ${
        selected
          ? "border-[var(--color-accent)] ring-1 ring-[var(--color-accent)]/30"
          : "border-[var(--color-border)]"
      }`}
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 flex-1 basis-[12rem] gap-3">
          <label className="mt-0.5 flex shrink-0 cursor-pointer items-center">
            <input
              type="checkbox"
              className="h-4 w-4 accent-[var(--color-accent)]"
              checked={selected}
              onChange={() => onToggleSelect(d.id)}
              aria-label={`Select ${d.fileName}`}
            />
          </label>
          <div className="min-w-0 flex-1">
            <div className="flex min-w-0 flex-wrap items-center gap-2">
              <button
                type="button"
                className="max-w-full truncate text-left text-sm font-semibold text-[var(--color-text)] hover:underline"
                onClick={() => onDetails(d.id)}
                title={d.fileName}
              >
                {d.fileName}
              </button>
              <StatusBadge status={d.status} />
            </div>
            <p className="mt-0.5 truncate text-xs text-[var(--color-text-muted)]">
              {hostnameFromUrl(d.url)}
            </p>
          </div>
        </div>
        <div className="flex max-w-full flex-wrap gap-2">
          <button
            type="button"
            className="rounded-md border border-[var(--color-border)] px-2.5 py-1 text-xs disabled:opacity-50"
            disabled={busy}
            onClick={() => onDetails(d.id)}
          >
            Details
          </button>
          {canPause ? (
            <button
              type="button"
              className="rounded-md border border-[var(--color-border)] px-2.5 py-1 text-xs disabled:opacity-50"
              disabled={busy}
              onClick={() => onPause(d.id)}
            >
              Pause
            </button>
          ) : null}
          {canResume ? (
            <button
              type="button"
              className="rounded-md border border-[var(--color-border)] px-2.5 py-1 text-xs disabled:opacity-50"
              disabled={busy}
              onClick={() => onResume(d.id)}
            >
              {d.status === "failed" || d.status === "cancelled"
                ? "Retry"
                : "Resume"}
            </button>
          ) : null}
          {canCancel ? (
            <button
              type="button"
              className="rounded-md border border-[var(--color-border)] px-2.5 py-1 text-xs disabled:opacity-50"
              disabled={busy}
              onClick={() => onCancel(d.id)}
            >
              Cancel
            </button>
          ) : null}
          <button
            type="button"
            className="rounded-md border border-[var(--color-border)] px-2.5 py-1 text-xs disabled:opacity-50"
            disabled={busy}
            onClick={() => onRemove(d.id)}
          >
            Remove
          </button>
        </div>
      </div>

      <div className="mt-3">
        <ProgressBar
          percent={pct}
          label={
            pct != null
              ? `${Math.round(pct)} percent complete`
              : "Download progress unknown"
          }
        />
        <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 font-variant-numeric tabular-nums text-xs text-[var(--color-text-muted)]">
          <span>{sizeLabel}</span>
          {pct != null ? <span>{Math.round(pct)}%</span> : null}
          {isActiveStatus(d.status) ? (
            <>
              <span>{formatSpeed(d.bytesPerSecond)}</span>
              <span>{formatEta(eta)}</span>
            </>
          ) : null}
        </div>
        {showGraph ? (
          <SpeedGraph samples={speedSamples} active={showGraph} />
        ) : null}
        {d.status === "failed" && d.errorMessage ? (
          <p className="mt-2 text-xs text-[var(--color-danger)]" role="alert">
            {d.errorMessage}
          </p>
        ) : null}
      </div>
    </article>
  );
}
