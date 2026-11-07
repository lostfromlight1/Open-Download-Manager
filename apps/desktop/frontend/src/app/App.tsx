import { useEffect, useState } from "react";
import { DownloadsPage } from "../features/downloads/DownloadsPage";
import type { SettingsDTO } from "../features/downloads/downloadTypes";
import { SettingsPage } from "../features/settings/SettingsPage";
import { SetupWizard } from "../features/setup/SetupWizard";
import { needsSetupWizard } from "../features/setup/needsSetupWizard";
import { applyTheme, watchSystemTheme } from "../lib/theme";
import { getDesktopApi } from "../services/desktopApi";

type Page = "downloads" | "settings";

export function App() {
  const [page, setPage] = useState<Page>("downloads");
  const [bootError, setBootError] = useState<string | null>(null);
  const [ready, setReady] = useState(false);
  const [settings, setSettings] = useState<SettingsDTO | null>(null);
  const [showSetup, setShowSetup] = useState(false);

  useEffect(() => {
    void (async () => {
      try {
        const api = await getDesktopApi();
        const available = await api.downloadsAvailable();
        if (!available) {
          const msg = await api.statusMessage();
          setBootError(msg || "Download manager is not ready");
          return;
        }
        const loaded = await api.getSettings();
        applyTheme(loaded.theme || "system");
        setSettings(loaded);
        setShowSetup(needsSetupWizard(loaded));
        setReady(true);
      } catch (e) {
        setBootError(e instanceof Error ? e.message : String(e));
      }
    })();
  }, []);

  useEffect(() => {
    if (settings?.theme !== "system") return;
    return watchSystemTheme(() => applyTheme("system"));
  }, [settings?.theme]);

  function handleSettingsSaved(next: SettingsDTO) {
    applyTheme(next.theme || "system");
    setSettings(next);
    if (!needsSetupWizard(next)) {
      setShowSetup(false);
    }
  }

  if (bootError) {
    return (
      <main className="mx-auto flex min-h-screen max-w-lg flex-col justify-center gap-3 px-6">
        <h1 className="text-2xl font-semibold text-[var(--color-text)]">
          Open Download Manager
        </h1>
        <p className="text-sm text-[var(--color-danger)]" role="alert">
          {bootError}
        </p>
        <button
          type="button"
          className="w-fit rounded-md border border-[var(--color-border)] px-3 py-2 text-sm"
          onClick={() => window.location.reload()}
        >
          Retry
        </button>
      </main>
    );
  }

  if (!ready || !settings) {
    return (
      <main className="mx-auto flex min-h-screen max-w-lg flex-col justify-center px-6">
        <p className="text-sm text-[var(--color-text-muted)]">
          Starting download manager…
        </p>
      </main>
    );
  }

  if (showSetup) {
    return (
      <SetupWizard
        initial={settings}
        onComplete={(next) => {
          handleSettingsSaved(next);
          setShowSetup(false);
        }}
      />
    );
  }

  if (page === "settings") {
    return (
      <SettingsPage
        onBack={() => setPage("downloads")}
        onSaved={handleSettingsSaved}
      />
    );
  }

  return (
    <DownloadsPage
      defaultDir={settings.downloadDirectory || ""}
      notifyOnComplete={settings.notifyOnComplete}
      onOpenSettings={() => setPage("settings")}
    />
  );
}
