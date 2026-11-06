import { useState } from "react";
import { Dialog } from "../../components/Dialog";

type Props = {
  mode: "cancel" | "remove";
  fileName: string;
  busy: boolean;
  onBack: () => void;
  onConfirm: (action: "keep" | "delete") => Promise<void>;
};

export function FileActionDialog({
  mode,
  fileName,
  busy,
  onBack,
  onConfirm,
}: Props) {
  const [action, setAction] = useState<"keep" | "delete">("keep");
  const [error, setError] = useState<string | null>(null);

  const title = mode === "cancel" ? "Cancel Download?" : "Remove Download?";
  const confirmLabel = mode === "cancel" ? "Confirm Cancel" : "Remove";

  return (
    <Dialog
      title={title}
      onClose={busy ? undefined : onBack}
      footer={
        <>
          <button
            type="button"
            className="rounded-md border border-[var(--color-border)] px-3 py-1.5 text-sm"
            onClick={onBack}
            disabled={busy}
          >
            Back
          </button>
          <button
            type="button"
            className="rounded-md bg-[var(--color-danger)] px-3 py-1.5 text-sm font-medium text-white disabled:opacity-60"
            disabled={busy}
            onClick={() => {
              setError(null);
              void onConfirm(action).catch((e) =>
                setError(e instanceof Error ? e.message : String(e)),
              );
            }}
          >
            {busy ? "Working…" : confirmLabel}
          </button>
        </>
      }
    >
      <p className="text-[var(--color-text-muted)]">{fileName}</p>
      <p className="mt-2">
        {mode === "cancel"
          ? "What should ODM do with the partial file?"
          : "How should ODM handle the associated file?"}
      </p>
      <fieldset className="mt-3 space-y-2">
        <legend className="sr-only">File action</legend>
        <label className="flex items-start gap-2 text-sm">
          <input
            type="radio"
            name="file-action"
            checked={action === "keep"}
            onChange={() => setAction("keep")}
          />
          <span>
            {mode === "cancel"
              ? "Keep partial file"
              : "Remove task only (keep file)"}
          </span>
        </label>
        <label className="flex items-start gap-2 text-sm">
          <input
            type="radio"
            name="file-action"
            checked={action === "delete"}
            onChange={() => setAction("delete")}
          />
          <span>
            {mode === "cancel"
              ? "Delete partial file"
              : "Remove task and delete associated file"}
          </span>
        </label>
      </fieldset>
      {error ? (
        <p className="mt-3 text-sm text-[var(--color-danger)]" role="alert">
          {error}
        </p>
      ) : null}
    </Dialog>
  );
}
