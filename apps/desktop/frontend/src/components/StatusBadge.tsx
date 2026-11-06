import { isActiveStatus } from "../features/downloads/downloadTypes";

const labels: Record<string, string> = {
  queued: "Queued",
  inspecting: "Inspecting",
  downloading: "Downloading",
  paused: "Paused",
  finalizing: "Finalizing",
  failed: "Failed",
  completed: "Completed",
  cancelled: "Cancelled",
};

export function StatusBadge({ status }: { status: string }) {
  let tone = "bg-[var(--color-surface-muted)] text-[var(--color-text-muted)]";
  if (isActiveStatus(status)) {
    tone = "bg-blue-50 text-[var(--color-accent)]";
  } else if (status === "completed") {
    tone = "bg-green-50 text-[var(--color-success)]";
  } else if (status === "failed") {
    tone = "bg-red-50 text-[var(--color-danger)]";
  } else if (status === "paused") {
    tone = "bg-amber-50 text-[var(--color-warning)]";
  }

  return (
    <span
      className={`inline-flex rounded-md px-2 py-0.5 text-xs font-medium ${tone}`}
    >
      {labels[status] ?? status}
    </span>
  );
}
