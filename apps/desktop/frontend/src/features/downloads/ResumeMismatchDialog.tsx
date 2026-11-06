import { useState } from "react";
import { Dialog } from "../../components/Dialog";
import type { DecisionRequiredDTO } from "./downloadTypes";

type Props = {
  decision: DecisionRequiredDTO;
  busy: boolean;
  onResolve: (choice: "restart" | "abort") => Promise<void>;
};

export function ResumeMismatchDialog({ decision, busy, onResolve }: Props) {
  const [error, setError] = useState<string | null>(null);

  async function choose(choice: "restart" | "abort") {
    setError(null);
    try {
      await onResolve(choice);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <Dialog title="Remote File Has Changed">
      <p>
        The remote file appears to have changed since the previous download
        attempt. Your existing partial download may no longer match.
      </p>
      <p className="mt-2 font-mono text-xs text-[var(--color-text-muted)]">
        {decision.destinationPath}
      </p>
      <div className="mt-4 flex flex-col gap-2">
        <button
          type="button"
          className="rounded-md bg-[var(--color-accent)] px-3 py-2 text-left text-sm font-medium text-white disabled:opacity-60"
          disabled={busy}
          onClick={() => void choose("restart")}
        >
          Restart Download
          <span className="mt-0.5 block text-xs font-normal text-blue-100">
            Discard old partial data and start again.
          </span>
        </button>
        <button
          type="button"
          className="rounded-md border border-[var(--color-border)] px-3 py-2 text-left text-sm disabled:opacity-60"
          disabled={busy}
          onClick={() => void choose("abort")}
        >
          Abort Resume and Keep Partial File
          <span className="mt-0.5 block text-xs text-[var(--color-text-muted)]">
            Stop the attempt and preserve existing data.
          </span>
        </button>
      </div>
      {error ? (
        <p className="mt-3 text-sm text-[var(--color-danger)]" role="alert">
          {error}
        </p>
      ) : null}
    </Dialog>
  );
}
