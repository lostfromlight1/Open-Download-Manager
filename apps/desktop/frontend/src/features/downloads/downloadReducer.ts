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
      return {
        ...state,
        byId: {
          ...state.byId,
          [action.event.taskId]: {
            ...cur,
            downloadedBytes: action.event.downloadedBytes,
            totalBytes: action.event.totalBytes,
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
