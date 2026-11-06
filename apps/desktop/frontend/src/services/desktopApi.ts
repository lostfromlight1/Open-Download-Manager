import type { DownloadDTO, SettingsDTO } from "../features/downloads/downloadTypes";

/**
 * Thin wrappers around generated Wails bindings.
 * In Vitest (no Wails), callers can inject mocks via setDesktopApi.
 */

export interface DesktopApi {
  listDownloads(): Promise<DownloadDTO[]>;
  addDownload(req: {
    url: string;
    destination: string;
  }): Promise<DownloadDTO>;
  pauseDownload(id: string): Promise<void>;
  resumeDownload(id: string): Promise<void>;
  cancelDownload(id: string, action: string): Promise<void>;
  removeDownload(id: string, action: string): Promise<void>;
  getSettings(): Promise<SettingsDTO>;
  updateSettings(settings: SettingsDTO): Promise<SettingsDTO>;
  resolveDecision(decisionId: string, choice: string): Promise<void>;
  getDownload(id: string): Promise<DownloadDTO>;
  openDownload(id: string): Promise<void>;
  revealDownload(id: string): Promise<void>;
  downloadsAvailable(): Promise<boolean>;
  statusMessage(): Promise<string>;
}

let api: DesktopApi | null = null;

export function setDesktopApi(next: DesktopApi | null): void {
  api = next;
}

async function loadBindings(): Promise<DesktopApi> {
  const shell = await import(
    "../../bindings/github.com/lostfromlight1/Open-Download-Manager/apps/desktop/shellservice"
  );
  const downloads = await import(
    "../../bindings/github.com/lostfromlight1/Open-Download-Manager/apps/desktop/downloadservice"
  );
  return {
    listDownloads: async () => (await downloads.ListDownloads()) ?? [],
    addDownload: (req) => downloads.AddDownload(req),
    pauseDownload: (id) => downloads.PauseDownload(id).then(() => undefined),
    resumeDownload: (id) => downloads.ResumeDownload(id).then(() => undefined),
    cancelDownload: (id, action) =>
      downloads.CancelDownload(id, action).then(() => undefined),
    removeDownload: (id, action) =>
      downloads.RemoveDownload(id, action).then(() => undefined),
    getSettings: () => downloads.GetSettings(),
    updateSettings: (settings) => downloads.UpdateSettings(settings),
    resolveDecision: (decisionId, choice) =>
      downloads.ResolveDecision(decisionId, choice).then(() => undefined),
    getDownload: (id) => downloads.GetDownload(id),
    openDownload: (id) => downloads.OpenDownload(id).then(() => undefined),
    revealDownload: (id) => downloads.RevealDownload(id).then(() => undefined),
    downloadsAvailable: () => shell.DownloadsAvailable(),
    statusMessage: () => shell.StatusMessage(),
  };
}

export async function getDesktopApi(): Promise<DesktopApi> {
  if (api) return api;
  api = await loadBindings();
  return api;
}
