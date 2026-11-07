import type {
  ChangedEventDTO,
  DecisionRequiredDTO,
  DownloadDTO,
  ProgressEventDTO,
} from "./downloadTypes";

export type DownloadsState = {
  status: "loading" | "ready" | "error";
  error: string | null;
  byId: Record<string, DownloadDTO>;
  order: string[];
  pendingDecision: DecisionRequiredDTO | null;
};

export type DownloadsAction =
  | { type: "loading" }
  | { type: "snapshot"; downloads: DownloadDTO[] }
  | { type: "error"; message: string }
  | { type: "progress"; event: ProgressEventDTO }
  | { type: "changed"; event: ChangedEventDTO }
  | { type: "decision"; event: DecisionRequiredDTO }
  | { type: "clear-decision" };

export const initialDownloadsState: DownloadsState = {
  status: "loading",
  error: null,
  byId: {},
  order: [],
  pendingDecision: null,
};

export function downloadsReducer(
  state: DownloadsState,
  action: DownloadsAction,
): DownloadsState {
  switch (action.type) {
    case "loading":
      return { ...state, status: "loading", error: null };
    case "error":
      return { ...state, status: "error", error: action.message };
    case "snapshot": {
      const byId: Record<string, DownloadDTO> = {};
      const order: string[] = [];
      for (const d of action.downloads) {
        byId[d.id] = d;
        order.push(d.id);
      }
      return {
        ...state,
        status: "ready",
        error: null,
        byId,
        order,
      };
    }
    case "progress": {
      const cur = state.byId[action.event.taskId];
      if (!cur) return state;
      // Keep the last known total when a progress tick omits it (e.g. between
      // yt-dlp streams). Never let downloadedBytes move backwards in the UI.
      const nextTotal =
        action.event.totalBytes != null && action.event.totalBytes > 0
          ? action.event.totalBytes
          : cur.totalBytes;
      const nextDownloaded = Math.max(
        cur.downloadedBytes,
        action.event.downloadedBytes,
      );
      return {
        ...state,
        byId: {
          ...state.byId,
          [action.event.taskId]: {
            ...cur,
            downloadedBytes: nextDownloaded,
            totalBytes: nextTotal,
            bytesPerSecond: action.event.bytesPerSecond,
            status: action.event.status || cur.status,
          },
        },
      };
    }
    case "changed": {
      const { taskId, download } = action.event;
      if (!download) {
        const rest = { ...state.byId };
        delete rest[taskId];
        return {
          ...state,
          byId: rest,
          order: state.order.filter((id) => id !== taskId),
        };
      }
      const exists = Boolean(state.byId[taskId]);
      return {
        ...state,
        byId: { ...state.byId, [taskId]: download },
        order: exists ? state.order : [taskId, ...state.order],
      };
    }
    case "decision":
      return { ...state, pendingDecision: action.event };
    case "clear-decision":
      return { ...state, pendingDecision: null };
    default:
      return state;
  }
}
