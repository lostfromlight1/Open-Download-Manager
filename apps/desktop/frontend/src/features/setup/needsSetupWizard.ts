import type { SettingsDTO } from "../downloads/downloadTypes";

/** True when the first-run wizard should block the dashboard. */
export function needsSetupWizard(settings: SettingsDTO): boolean {
  return !settings.downloadDirectory?.trim();
}
