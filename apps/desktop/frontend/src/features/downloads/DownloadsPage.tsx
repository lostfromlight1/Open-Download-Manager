import { useCallback, useEffect, useMemo, useReducer, useRef, useState } from "react";
import { notifyDownloadComplete } from "../../lib/notify";
import { getDesktopApi } from "../../services/desktopApi";
import {
  subscribeChanged,
  subscribeDecision,
  subscribeProgress,
} from "../../services/desktopEvents";
import { AddDownloadDialog } from "./AddDownloadDialog";
import { DownloadDetail } from "./DownloadDetail";
import { DownloadRow } from "./DownloadRow";
import {
  downloadsReducer,
  initialDownloadsState,
} from "./downloadReducer";
import { FileActionDialog } from "./FileActionDialog";
import { ResumeMismatchDialog } from "./ResumeMismatchDialog";
import type { DownloadDTO, FilterTab } from "./downloadTypes";
import { isActiveStatus } from "./downloadTypes";

type Props = {
  defaultDir: string;
  notifyOnComplete: boolean;
  onOpenSettings: () => void;
};

export function DownloadsPage({
  defaultDir,
  notifyOnComplete,
  onOpenSettings,
}: Props) {
  const [state, dispatch] = useReducer(downloadsReducer, initialDownloadsState);
  const [filter, setFilter] = useState<FilterTab>("all");
  const [showAdd, setShowAdd] = useState(false);
  const [detailId, setDetailId] = useState<string | null>(null);
  const [fileDialog, setFileDialog] = useState<{
    mode: "cancel" | "remove";
    download: DownloadDTO;
  } | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [decisionBusy, setDecisionBusy] = useState(false);
  const [addBusy, setAddBusy] = useState(false);
  const notifyRef = useRef(notifyOnComplete);
  const prevStatus = useRef<Record<string, string>>({});

  useEffect(() => {
    notifyRef.current = notifyOnComplete;
  }, [notifyOnComplete]);

  const refresh = useCallback(async () => {
    try {
      const api = await getDesktopApi();
      const list = await api.listDownloads();
      dispatch({ type: "snapshot", downloads: list });
    } catch (e) {
      dispatch({
        type: "error",
        message: e instanceof Error ? e.message : String(e),
      });
    }
  }, []);

  useEffect(() => {
    dispatch({ type: "loading" });
    void refresh();

    const offProgress = subscribeProgress((event) =>
      dispatch({ type: "progress", event }),
    );
    const offChanged = subscribeChanged((event) => {
      const prev = prevStatus.current[event.taskId];
      const next = event.download?.status;
      if (
        notifyRef.current &&
        next === "completed" &&
        prev &&
        prev !== "completed"
      ) {
        void notifyDownloadComplete(event.download?.fileName ?? "Download");
      }
      if (event.download) {
        prevStatus.current[event.taskId] = event.download.status;
      } else {
        delete prevStatus.current[event.taskId];
      }
      dispatch({ type: "changed", event });
    });
    const offDecision = subscribeDecision((event) =>
      dispatch({ type: "decision", event }),
    );
    return () => {
      offProgress();
      offChanged();
      offDecision();
    };
  }, [refresh]);

  useEffect(() => {
    for (const d of Object.values(state.byId)) {
      prevStatus.current[d.id] = d.status;
    }
  }, [state.byId]);

  const downloads = useMemo(() => {
    const list: DownloadDTO[] = [];
    for (const id of state.order) {
      const row = state.byId[id];
      if (row) list.push(row);
    }
    return list;
  }, [state.byId, state.order]);

  const counts = useMemo(() => {
    let active = 0;
    let queued = 0;
    let completed = 0;
    let failed = 0;
    let cancelled = 0;
    for (const d of downloads) {
      if (isActiveStatus(d.status)) active++;
      else if (d.status === "queued") queued++;
      else if (d.status === "completed") completed++;
      else if (d.status === "failed") failed++;
      else if (d.status === "cancelled") cancelled++;
    }
    return { active, queued, completed, failed, cancelled, all: downloads.length };
  }, [downloads]);

  const filtered = useMemo(() => {
    switch (filter) {
      case "active":
        return downloads.filter((d) => isActiveStatus(d.status));
      case "queued":
        return downloads.filter((d) => d.status === "queued");
      case "completed":
        return downloads.filter((d) => d.status === "completed");
      case "failed":
        return downloads.filter((d) => d.status === "failed");
      case "cancelled":
        return downloads.filter((d) => d.status === "cancelled");
      default:
        return downloads;
    }
  }, [downloads, filter]);

  async function runCommand(id: string, fn: () => Promise<void>) {
    setBusyId(id);
    try {
      await fn();
      await refresh();
    } catch (e) {
      dispatch({
        type: "error",
        message: e instanceof Error ? e.message : String(e),
      });
    } finally {
      setBusyId(null);
    }
  }

  const tabs: { id: FilterTab; label: string; count: number }[] = [
    { id: "all", label: "All", count: counts.all },
    { id: "active", label: "Active", count: counts.active },
    { id: "queued", label: "Queued", count: counts.queued },
    { id: "completed", label: "Completed", count: counts.completed },
    { id: "failed", label: "Failed", count: counts.failed },
    { id: "cancelled", label: "Cancelled", count: counts.cancelled },
  ];

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-5 px-6 py-6">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="text-sm font-medium text-[var(--color-text-muted)]">ODM</p>
          <h1 className="text-2xl font-semibold tracking-tight text-[var(--color-text)]">
            Open Download Manager
          </h1>
        </div>
        <div className="flex gap-2">
          <button
            type="button"
            className="rounded-md border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm"
            onClick={onOpenSettings}
          >
            Settings
          </button>
          <button
            type="button"
            className="rounded-md bg-[var(--color-accent)] px-3 py-2 text-sm font-medium text-white"
            onClick={() => setShowAdd(true)}
          >
            + Add
          </button>
        </div>
      </header>

      <section className="grid grid-cols-3 gap-3">
        {[
          { label: "Active", value: counts.active },
          { label: "Queued", value: counts.queued },
          { label: "Completed", value: counts.completed },
        ].map((c) => (
          <div
            key={c.label}
            className="rounded-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-3"
          >
            <p className="text-xs text-[var(--color-text-muted)]">{c.label}</p>
            <p className="mt-1 text-2xl font-semibold tabular-nums">{c.value}</p>
          </div>
        ))}
      </section>

      <div
        className="flex flex-wrap gap-1 border-b border-[var(--color-border)] pb-2"
        role="tablist"
        aria-label="Download filters"
      >
        {tabs.map((t) => (
          <button
            key={t.id}
            type="button"
            role="tab"
            aria-selected={filter === t.id}
            className={`rounded-md px-3 py-1.5 text-sm ${
              filter === t.id
                ? "bg-[var(--color-surface-muted)] font-medium text-[var(--color-text)]"
                : "text-[var(--color-text-muted)]"
            }`}
            onClick={() => setFilter(t.id)}
          >
            {t.label} ({t.count})
          </button>
        ))}
      </div>

      {state.status === "loading" ? (
        <p className="text-sm text-[var(--color-text-muted)]">Loading downloads…</p>
      ) : null}
      {state.status === "error" ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-[var(--color-danger)]">
          <p role="alert">{state.error}</p>
          <button
            type="button"
            className="mt-2 underline"
            onClick={() => void refresh()}
          >
            Retry
          </button>
        </div>
      ) : null}
      {state.status === "ready" && filtered.length === 0 ? (
        <div className="rounded-[10px] border border-dashed border-[var(--color-border)] bg-[var(--color-surface)] px-6 py-12 text-center">
          <p className="text-sm font-medium text-[var(--color-text)]">
            No downloads yet
          </p>
          <p className="mt-1 text-sm text-[var(--color-text-muted)]">
            Add an HTTP or HTTPS URL to get started.
          </p>
          <button
            type="button"
            className="mt-4 rounded-md bg-[var(--color-accent)] px-3 py-2 text-sm font-medium text-white"
            onClick={() => setShowAdd(true)}
          >
            Add Download
          </button>
        </div>
      ) : null}

      <div className="flex flex-col gap-3">
        {filtered.map((d) => (
          <DownloadRow
            key={d.id}
            download={d}
            busyId={busyId}
            onDetails={(id) => setDetailId(id)}
            onPause={(id) =>
              void runCommand(id, async () => {
                const api = await getDesktopApi();
                await api.pauseDownload(id);
              })
            }
            onResume={(id) =>
              void runCommand(id, async () => {
                const api = await getDesktopApi();
                await api.resumeDownload(id);
              })
            }
            onCancel={(id) => {
              const row = state.byId[id];
              if (row) setFileDialog({ mode: "cancel", download: row });
            }}
            onRemove={(id) => {
              const row = state.byId[id];
              if (row) setFileDialog({ mode: "remove", download: row });
            }}
          />
        ))}
      </div>

      {showAdd ? (
        <AddDownloadDialog
          defaultDir={defaultDir}
          busy={addBusy}
          onCancel={() => setShowAdd(false)}
          onSubmit={async (url, destination) => {
            setAddBusy(true);
            try {
              const api = await getDesktopApi();
              await api.addDownload({ url, destination });
              setShowAdd(false);
              await refresh();
            } finally {
              setAddBusy(false);
            }
          }}
        />
      ) : null}

      {fileDialog ? (
        <FileActionDialog
          mode={fileDialog.mode}
          fileName={fileDialog.download.fileName}
          busy={busyId === fileDialog.download.id}
          onBack={() => setFileDialog(null)}
          onConfirm={async (action) => {
            const id = fileDialog.download.id;
            setBusyId(id);
            try {
              const api = await getDesktopApi();
              if (fileDialog.mode === "cancel") {
                await api.cancelDownload(id, action);
              } else {
                await api.removeDownload(id, action);
              }
              setFileDialog(null);
              await refresh();
            } finally {
              setBusyId(null);
            }
          }}
        />
      ) : null}

      {state.pendingDecision ? (
        <ResumeMismatchDialog
          decision={state.pendingDecision}
          busy={decisionBusy}
          onResolve={async (choice) => {
            setDecisionBusy(true);
            try {
              const api = await getDesktopApi();
              await api.resolveDecision(
                state.pendingDecision!.decisionId,
                choice,
              );
              dispatch({ type: "clear-decision" });
              await refresh();
            } finally {
              setDecisionBusy(false);
            }
          }}
        />
      ) : null}

      {detailId && state.byId[detailId] ? (
        <DownloadDetail
          download={state.byId[detailId]}
          busy={busyId === detailId}
          onClose={() => setDetailId(null)}
          onOpen={() =>
            void runCommand(detailId, async () => {
              const api = await getDesktopApi();
              await api.openDownload(detailId);
            })
          }
          onReveal={() =>
            void runCommand(detailId, async () => {
              const api = await getDesktopApi();
              await api.revealDownload(detailId);
            })
          }
        />
      ) : null}
    </div>
  );
}
