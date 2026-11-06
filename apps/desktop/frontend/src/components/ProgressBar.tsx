type Props = {
  percent: number | null;
  label: string;
};

export function ProgressBar({ percent, label }: Props) {
  const known = percent != null;
  return (
    <div
      className="h-2 w-full overflow-hidden rounded-full bg-[var(--color-surface-muted)]"
      role="progressbar"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={known ? Math.round(percent) : undefined}
      aria-valuetext={label}
      aria-label={label}
    >
      {known ? (
        <div
          className="h-full rounded-full bg-[var(--color-accent)] transition-[width] duration-200"
          style={{ width: `${percent}%` }}
        />
      ) : (
        <div className="h-full w-1/3 animate-pulse rounded-full bg-[var(--color-border)]" />
      )}
    </div>
  );
}
