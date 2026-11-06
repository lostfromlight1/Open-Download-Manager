export type DownloadStatus =
  | "queued"
  | "inspecting"
  | "downloading"
  | "paused"
  | "finalizing"
  | "failed"
  | "completed"
  | "cancelled";

export interface DownloadDTO {
  id: string;
  url: string;
  fileName: string;
  destinationPath: string;
  status: DownloadStatus | string;
  downloadedBytes: number;
  totalBytes: number | null;
  bytesPerSecond?: number | null;
  errorCode?: string;
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ProgressEventDTO {
  taskId: string;
  downloadedBytes: number;
  totalBytes: number | null;
  bytesPerSecond: number | null;
  status: string;
}

export interface ChangedEventDTO {
  taskId: string;
  download: DownloadDTO | null;
}

export interface DecisionRequiredDTO {
  decisionId: string;
  taskId: string;
  destinationPath: string;
  savedETag: string;
  savedModified: string;
  remoteETag: string;
  remoteModified: string;
  kind: string;
}

export interface SettingsDTO {
  downloadDirectory: string;
  maxActiveDownloads: number;
  maxConnectionsPerDownload: number;
  proxyMode: string;
  proxyURL: string;
}

export type FilterTab =
  | "all"
  | "active"
  | "queued"
  | "completed"
  | "failed"
  | "cancelled";

export function isActiveStatus(status: string): boolean {
  return (
    status === "inspecting" ||
    status === "downloading" ||
    status === "finalizing"
  );
}
