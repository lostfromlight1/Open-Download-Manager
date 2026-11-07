import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { App } from "./App";
import { setDesktopApi, type DesktopApi } from "../services/desktopApi";

vi.mock("../services/desktopEvents", () => ({
  subscribeProgress: () => () => undefined,
  subscribeChanged: () => () => undefined,
  subscribeDecision: () => () => undefined,
  subscribeMediaPick: () => () => undefined,
}));

function mockApi(partial: Partial<DesktopApi> = {}): DesktopApi {
  return {
    listDownloads: async () => [],
    addDownload: async () => {
      throw new Error("not used");
    },
    pauseDownload: async () => undefined,
    resumeDownload: async () => undefined,
    cancelDownload: async () => undefined,
    removeDownload: async () => undefined,
    getSettings: async () => ({
      downloadDirectory: "C:/Downloads",
      maxActiveDownloads: 2,
      maxConnectionsPerDownload: 4,
      proxyMode: "system",
      proxyURL: "",
      theme: "system",
      notifyOnComplete: false,
      browserIntegrationEnabled: false,
      chromeExtensionId: "",
      browserHostRegistered: false,
      browserHostMessage: "",
      closeToTray: true,
      launchAtLogin: false,
      mediaEnabled: false,
      mediaYtdlpPath: "",
      mediaFfmpegPath: "",
      mediaToolsReady: false,
      mediaToolsMessage: "",
    }),
    getDownload: async () => {
      throw new Error("not used");
    },
    openDownload: async () => undefined,
    revealDownload: async () => undefined,
    updateSettings: async (s) => s,
    resolveDecision: async () => undefined,
    downloadsAvailable: async () => true,
    statusMessage: async () => "Download manager ready",
    isYouTubeURL: async () => false,
    probeMediaURL: async () => {
      throw new Error("not used");
    },
    addMediaDownload: async () => {
      throw new Error("not used");
    },
    ...partial,
  };
}

describe("App", () => {
  beforeEach(() => {
    setDesktopApi(null);
  });
  afterEach(() => {
    cleanup();
    setDesktopApi(null);
  });

  it("shows empty dashboard when ready with no downloads", async () => {
    setDesktopApi(mockApi());
    render(<App />);
    await waitFor(() => {
      expect(screen.getByText("No downloads yet")).toBeInTheDocument();
    });
    expect(
      screen.getByRole("heading", { name: "Open Download Manager" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByText("Download management is not available in this build."),
    ).not.toBeInTheDocument();
  });

  it("shows boot error when downloads unavailable", async () => {
    setDesktopApi(
      mockApi({
        downloadsAvailable: async () => false,
        statusMessage: async () => "database open failed",
      }),
    );
    render(<App />);
    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        "database open failed",
      );
    });
  });
});
