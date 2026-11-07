import { formatBytes, formatMbps } from "../../lib/format";

export type SpeedSample = {
  /** unix ms */
  at: number;
  /** network throughput bytes/sec (from engine) */
  netBps: number;
  /** disk write estimate bytes/sec (from downloadedBytes delta) */
  diskBps: number;
};

type Props = {
  samples: SpeedSample[];
  /** Show graph even with few points while active */
  active: boolean;
};

const W = 280;
const H = 56;
const PAD = 2;

/** Steam-style dual sparkline: network (Mbps) + disk write. */
export function SpeedGraph({ samples, active }: Props) {
  if (!active && samples.length < 2) {
    return null;
  }

  const pts = samples.length > 0 ? samples : [{ at: 0, netBps: 0, diskBps: 0 }];
  const maxBps = Math.max(
    1,
    ...pts.map((s) => Math.max(s.netBps, s.diskBps)),
  );
  const last = pts[pts.length - 1]!;
  const netPath = toPath(pts.map((s) => s.netBps), maxBps);
  const diskPath = toPath(pts.map((s) => s.diskBps), maxBps);

  return (
    <div className="mt-3 rounded-md border border-[var(--color-border)] bg-[var(--color-surface-muted)]/40 px-3 py-2">
      <div className="flex flex-wrap items-center justify-between gap-2 text-[11px] tabular-nums text-[var(--color-text-muted)]">
        <span className="inline-flex items-center gap-1.5">
          <span
            className="inline-block h-1.5 w-3 rounded-sm bg-[var(--color-accent)]"
            aria-hidden
          />
          Net {formatMbps(last.netBps)}
        </span>
        <span className="inline-flex items-center gap-1.5">
          <span
            className="inline-block h-1.5 w-3 rounded-sm bg-emerald-600"
            aria-hidden
          />
          Disk {formatBytes(last.diskBps)}/s
        </span>
      </div>
      <svg
        viewBox={`0 0 ${W} ${H}`}
        className="mt-1.5 h-14 w-full"
        role="img"
        aria-label={`Network ${formatMbps(last.netBps)}, disk write ${formatBytes(last.diskBps)} per second`}
      >
        <path
          d={diskPath}
          fill="none"
          stroke="rgb(5 150 105)"
          strokeWidth="1.5"
          opacity="0.85"
        />
        <path
          d={netPath}
          fill="none"
          stroke="var(--color-accent)"
          strokeWidth="1.75"
        />
      </svg>
    </div>
  );
}

function toPath(values: number[], maxBps: number): string {
  if (values.length === 0) return "";
  const n = Math.max(values.length - 1, 1);
  return values
    .map((v, i) => {
      const x = PAD + (i / n) * (W - PAD * 2);
      const y = H - PAD - (Math.min(v, maxBps) / maxBps) * (H - PAD * 2);
      return `${i === 0 ? "M" : "L"}${x.toFixed(1)} ${y.toFixed(1)}`;
    })
    .join(" ");
}
