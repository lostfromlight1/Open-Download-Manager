import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { setDesktopApi, type DesktopApi } from "../../services/desktopApi";
import type { SettingsDTO } from "../downloads/downloadTypes";
import { needsSetupWizard } from "./needsSetupWizard";
import { SetupWizard } from "./SetupWizard";

const emptySettings: SettingsDTO = {
  downloadDirectory: "",
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
};

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
    getSettings: async () => emptySettings,
    getDownload: async () => {
      throw new Error("not used");
    },
    openDownload: async () => undefined,
    revealDownload: async () => undefined,
    updateSettings: async (s) => s,
    resolveDecision: async () => undefined,
    downloadsAvailable: async () => true,
    statusMessage: async () => "ok",
    isYouTubeURL: async () => false,
    probeMediaURL: async () => {
      throw new Error("not used");
    },
    addMediaDownload: async () => {
      throw new Error("not used");
    },
    ensureMediaTools: async () => ({
      ...emptySettings,
      downloadDirectory: "D:\\Download",
      mediaEnabled: true,
      mediaToolsReady: true,
      mediaToolsMessage: "Helpers ready.",
    }),
    ...partial,
  };
}

describe("needsSetupWizard", () => {
  it("is true when download directory is empty", () => {
    expect(needsSetupWizard(emptySettings)).toBe(true);
    expect(
      needsSetupWizard({ ...emptySettings, downloadDirectory: "  " }),
    ).toBe(true);
    expect(
      needsSetupWizard({ ...emptySettings, downloadDirectory: "D:/Download" }),
    ).toBe(false);
  });
});

describe("SetupWizard", () => {
  beforeEach(() => {
    setDesktopApi(null);
  });
  afterEach(() => {
    cleanup();
    setDesktopApi(null);
  });

  it("requires a folder before continuing", async () => {
    setDesktopApi(mockApi());
    render(
      <SetupWizard initial={emptySettings} onComplete={() => undefined} />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Choose a download folder",
    );
  });

  it("completes after folder + skips", async () => {
    let finished: SettingsDTO | null = null;
    setDesktopApi(mockApi());
    render(
      <SetupWizard
        initial={emptySettings}
        onComplete={(s) => {
          finished = s;
        }}
      />,
    );
    fireEvent.change(screen.getByPlaceholderText("D:\\Download"), {
      target: { value: "D:\\Download" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    await waitFor(() => {
      expect(
        screen.getByRole("heading", { name: "Browser integration" }),
      ).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole("button", { name: "Skip for now" }));
    await waitFor(() => {
      expect(
        screen.getByRole("heading", { name: "YouTube downloads" }),
      ).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole("button", { name: "Skip and finish" }));
    await waitFor(() => {
      expect(finished?.downloadDirectory).toBe("D:\\Download");
    });
  });
});
