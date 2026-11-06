import { ProgressBar } from "../../components/ProgressBar";
import { StatusBadge } from "../../components/StatusBadge";
import {
  etaSeconds,
  formatBytes,
  formatEta,
  formatSpeed,
  hostnameFromUrl,
  progressPercent,
} from "../../lib/format";
import type { DownloadDTO } from "./downloadTypes";
import { isActiveStatus } from "./downloadTypes";

type Props = {
  download: DownloadDTO;
  busyId: string | null;
  onPause: (id: string) => void;
  onResume: (id: string) => void;
  onCancel: (id: string) => void;
  onRemove: (id: string) => void;
  onDetails: (id: string) => void;
};

export function DownloadRow({
  download: d,
  busyId,
  onPause,
  onResume,
  onCancel,
  onRemove,
  onDetails,
}: Props) {
  const busy = busyId === d.id;
  const pct = progressPercent(d.downloadedBytes, d.totalBytes);
  const eta = etaSeconds(
    d.downloadedBytes,
    d.totalBytes,
    d.bytesPerSecond ?? null,
  );
  const sizeLabel =
    d.totalBytes != null
      ? `${formatBytes(d.downloadedBytes)} / ${formatBytes(d.totalBytes)}`
      : formatBytes(d.downloadedBytes);

  const canPause = isActiveStatus(d.status) || d.status === "queued";
  const canResume =
    d.status === "paused" || d.status === "failed" || d.status === "cancelled";
  const canCancel =
    isActiveStatus(d.status) || d.status === "queued" || d.status === "paused";

  return (
    <article className="rounded-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              className="truncate text-left text-sm font-semibold text-[var(--color-text)] hover:underline"
              onClick={() => onDetails(d.id)}
            >
              {d.fileName}
            </button>
            <StatusBadge status={d.status} />
          </div>
          <p className="mt-0.5 truncate text-xs text-[var(--color-text-muted)]">
            {hostnameFromUrl(d.url)}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
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
        {d.status === "failed" && d.errorMessage ? (
          <p className="mt-2 text-xs text-[var(--color-danger)]" role="alert">
            {d.errorMessage}
          </p>
        ) : null}
      </div>
    </article>
  );
}
