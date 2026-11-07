import type { DownloadDTO } from "./downloadTypes";
import { isActiveStatus } from "./downloadTypes";

export function canPauseTask(d: DownloadDTO): boolean {
  return isActiveStatus(d.status) || d.status === "queued";
}

export function canResumeTask(d: DownloadDTO): boolean {
  return (
    d.status === "paused" || d.status === "failed" || d.status === "cancelled"
  );
}

export function canCancelTask(d: DownloadDTO): boolean {
  return (
    isActiveStatus(d.status) || d.status === "queued" || d.status === "paused"
  );
}

export function partitionSelection(
  selected: DownloadDTO[],
): {
  pauseable: DownloadDTO[];
  resumeable: DownloadDTO[];
  cancelable: DownloadDTO[];
} {
  return {
    pauseable: selected.filter(canPauseTask),
    resumeable: selected.filter(canResumeTask),
    cancelable: selected.filter(canCancelTask),
  };
}
