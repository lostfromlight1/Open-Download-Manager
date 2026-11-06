/** Format byte counts for the dashboard. Never invent values. */
export function formatBytes(n: number | null | undefined): string {
  if (n == null || Number.isNaN(n)) return "—";
  const abs = Math.abs(n);
  if (abs < 1024) return `${n} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let v = abs;
  let i = -1;
  do {
    v /= 1024;
    i++;
  } while (v >= 1024 && i < units.length - 1);
  const signed = n < 0 ? "-" : "";
  return `${signed}${v.toFixed(v >= 10 || i === 0 ? 0 : 1)} ${units[i]}`;
}

/** Format bytes/sec from the backend; show em dash when unknown. */
export function formatSpeed(bps: number | null | undefined): string {
  if (bps == null || !(bps > 0)) return "—";
  return `${formatBytes(bps)}/s`;
}

/**
 * ETA from remaining bytes and speed. Returns null when it cannot be known
 * honestly (unknown total or non-positive speed).
 */
export function etaSeconds(
  downloaded: number,
  total: number | null | undefined,
  bps: number | null | undefined,
): number | null {
  if (total == null || total <= 0 || bps == null || !(bps > 0)) return null;
  const left = total - downloaded;
  if (left <= 0) return 0;
  return Math.ceil(left / bps);
}

export function formatEta(seconds: number | null | undefined): string {
  if (seconds == null || Number.isNaN(seconds)) return "—";
  if (seconds < 60) return `${seconds}s left`;
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m < 60) return `${m}m ${s}s left`;
  const h = Math.floor(m / 60);
  return `${h}h ${m % 60}m left`;
}

export function progressPercent(
  downloaded: number,
  total: number | null | undefined,
): number | null {
  if (total == null || total <= 0) return null;
  return Math.min(100, Math.max(0, (100 * downloaded) / total));
}

export function hostnameFromUrl(url: string): string {
  try {
    return new URL(url).hostname || url;
  } catch {
    return url;
  }
}
