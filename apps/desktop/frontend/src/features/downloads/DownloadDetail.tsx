import { Dialog } from "../../components/Dialog";
import { StatusBadge } from "../../components/StatusBadge";
import { formatBytes, hostnameFromUrl } from "../../lib/format";
import type { DownloadDTO } from "./downloadTypes";

type Props = {
  download: DownloadDTO;
  busy: boolean;
  onClose: () => void;
  onOpen: () => void;
  onReveal: () => void;
};

export function DownloadDetail({
  download: d,
  busy,
  onClose,
  onOpen,
  onReveal,
}: Props) {
  const canOpen = d.status === "completed";

  return (
    <Dialog
      title="Download details"
      onClose={onClose}
      footer={
        <>
          <button
            type="button"
            className="rounded-md border border-[var(--color-border)] px-3 py-1.5 text-sm"
            onClick={onClose}
          >
            Close
          </button>
          <button
            type="button"
            className="rounded-md border border-[var(--color-border)] px-3 py-1.5 text-sm disabled:opacity-50"
            disabled={busy}
            onClick={onReveal}
          >
            Show in folder
          </button>
          <button
            type="button"
            className="rounded-md bg-[var(--color-accent)] px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
            disabled={busy || !canOpen}
            onClick={onOpen}
            title={canOpen ? "Open file" : "Available when completed"}
          >
            Open file
          </button>
        </>
      }
    >
      <dl className="space-y-3 text-sm">
        <div>
          <dt className="text-[var(--color-text-muted)]">File</dt>
          <dd className="font-medium break-all">{d.fileName}</dd>
        </div>
        <div className="flex items-center gap-2">
          <dt className="text-[var(--color-text-muted)]">Status</dt>
          <dd>
            <StatusBadge status={d.status} />
          </dd>
        </div>
        <div>
          <dt className="text-[var(--color-text-muted)]">URL</dt>
          <dd className="break-all font-mono text-xs">{d.url}</dd>
          <dd className="mt-0.5 text-xs text-[var(--color-text-muted)]">
            {hostnameFromUrl(d.url)}
          </dd>
        </div>
        <div>
          <dt className="text-[var(--color-text-muted)]">Destination</dt>
          <dd className="break-all font-mono text-xs">{d.destinationPath}</dd>
        </div>
        <div>
          <dt className="text-[var(--color-text-muted)]">Size</dt>
          <dd className="tabular-nums">
            {d.totalBytes != null
              ? `${formatBytes(d.downloadedBytes)} / ${formatBytes(d.totalBytes)}`
              : formatBytes(d.downloadedBytes)}
          </dd>
        </div>
        {d.errorMessage ? (
          <div>
            <dt className="text-[var(--color-text-muted)]">Error</dt>
            <dd className="text-[var(--color-danger)]">{d.errorMessage}</dd>
          </div>
        ) : null}
        <div className="grid grid-cols-2 gap-3 text-xs text-[var(--color-text-muted)]">
          <div>
            <dt>Created</dt>
            <dd>{d.createdAt || "—"}</dd>
          </div>
          <div>
            <dt>Updated</dt>
            <dd>{d.updatedAt || "—"}</dd>
          </div>
        </div>
      </dl>
    </Dialog>
  );
}
