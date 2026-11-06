import { useState } from "react";
import { Dialog } from "../../components/Dialog";

type Props = {
  defaultDir: string;
  busy: boolean;
  onCancel: () => void;
  onSubmit: (url: string, destination: string) => Promise<void>;
};

export function AddDownloadDialog({
  defaultDir,
  busy,
  onCancel,
  onSubmit,
}: Props) {
  const [url, setUrl] = useState("");
  // Empty destination → Go uses the saved download directory + URL filename.
  const [destination, setDestination] = useState("");
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit() {
    setError(null);
    const u = url.trim();
    if (!u) {
      setError("URL is required");
      return;
    }
    try {
      // Quick client check; Go re-validates.
      const parsed = new URL(u);
      if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
        setError("Only http and https are supported");
        return;
      }
    } catch {
      setError("Enter a valid http(s) URL");
      return;
    }
    try {
      await onSubmit(u, destination.trim());
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <Dialog
      title="Add Download"
      onClose={busy ? undefined : onCancel}
      footer={
        <>
          <button
            type="button"
            className="rounded-md border border-[var(--color-border)] px-3 py-1.5 text-sm"
            onClick={onCancel}
            disabled={busy}
          >
            Cancel
          </button>
          <button
            type="button"
            className="rounded-md bg-[var(--color-accent)] px-3 py-1.5 text-sm font-medium text-white disabled:opacity-60"
            onClick={() => void handleSubmit()}
            disabled={busy}
          >
            {busy ? "Adding…" : "Add Download"}
          </button>
        </>
      }
    >
      <label className="block text-sm font-medium">
        Download URL
        <input
          className="mt-1 w-full rounded-md border border-[var(--color-border)] px-3 py-2 font-mono text-sm"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://example.com/file.zip"
          autoFocus
        />
      </label>
      <label className="mt-3 block text-sm font-medium">
        Save to (full file path)
        <input
          className="mt-1 w-full rounded-md border border-[var(--color-border)] px-3 py-2 font-mono text-sm"
          value={destination}
          onChange={(e) => setDestination(e.target.value)}
          placeholder={
            defaultDir
              ? `Optional — default folder: ${defaultDir}`
              : "Required if no default folder is set in Settings"
          }
        />
      </label>
      {error ? (
        <p className="mt-3 text-sm text-[var(--color-danger)]" role="alert">
          {error}
        </p>
      ) : null}
    </Dialog>
  );
}
