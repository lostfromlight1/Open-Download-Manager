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

export interface MediaPickRequiredDTO {
  url: string;
}

export interface SettingsDTO {
  downloadDirectory: string;
  maxActiveDownloads: number;
  maxConnectionsPerDownload: number;
  proxyMode: string;
  proxyURL: string;
  theme: string;
  notifyOnComplete: boolean;
  browserIntegrationEnabled: boolean;
  chromeExtensionId: string;
  browserHostRegistered: boolean;
  browserHostMessage: string;
  closeToTray: boolean;
  launchAtLogin: boolean;
  mediaEnabled: boolean;
  mediaYtdlpPath: string;
  mediaFfmpegPath: string;
  mediaToolsReady: boolean;
  mediaToolsMessage: string;
}

export interface MediaFormatDTO {
  id: string;
  ext: string;
  resolution: string;
  note: string;
  isAudio: boolean;
  videoOnly: boolean;
  height: number;
}

export interface MediaPlaylistEntryDTO {
  id: string;
  title: string;
  url: string;
}

export interface MediaProbeDTO {
  kind: string;
  id: string;
  title: string;
  webpageUrl: string;
  formats: MediaFormatDTO[];
  entries: MediaPlaylistEntryDTO[];
}

export type FilterTab =
  "all" | "active" | "queued" | "completed" | "failed" | "cancelled";

export function isActiveStatus(status: string): boolean {
  return (
    status === "inspecting" ||
    status === "downloading" ||
    status === "finalizing"
  );
}
