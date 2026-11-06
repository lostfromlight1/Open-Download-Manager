export type ThemeMode = "system" | "light" | "dark";

/** Apply theme to <html data-theme>. system follows prefers-color-scheme. */
export function applyTheme(mode: ThemeMode | string): void {
  const root = document.documentElement;
  const normalized = normalizeTheme(mode);
  root.setAttribute("data-theme-pref", normalized);

  let resolved: "light" | "dark" = "light";
  if (normalized === "dark") {
    resolved = "dark";
  } else if (normalized === "system") {
    const mq = window.matchMedia?.("(prefers-color-scheme: dark)");
    resolved = mq?.matches ? "dark" : "light";
  }
  root.setAttribute("data-theme", resolved);
}

export function normalizeTheme(mode: string): ThemeMode {
  if (mode === "light" || mode === "dark" || mode === "system") return mode;
  return "system";
}

/** Watch OS theme changes when preference is system. Returns unsubscribe. */
export function watchSystemTheme(onChange: () => void): () => void {
  const mq = window.matchMedia?.("(prefers-color-scheme: dark)");
  if (!mq) return () => undefined;
  const handler = () => onChange();
  mq.addEventListener("change", handler);
  return () => mq.removeEventListener("change", handler);
}
