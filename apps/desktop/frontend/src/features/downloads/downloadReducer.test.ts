import { describe, expect, it } from "vitest";
import { downloadsReducer, initialDownloadsState } from "./downloadReducer";
import type { DownloadDTO } from "./downloadTypes";

const sample: DownloadDTO = {
  id: "1",
  url: "https://example.com/a.bin",
  fileName: "a.bin",
  destinationPath: "C:/tmp/a.bin",
  status: "downloading",
  downloadedBytes: 10,
  totalBytes: 100,
  createdAt: "",
  updatedAt: "",
};

describe("downloadsReducer", () => {
  it("applies snapshot and progress", () => {
    let state = downloadsReducer(initialDownloadsState, {
      type: "snapshot",
      downloads: [sample],
    });
    expect(state.status).toBe("ready");
    expect(state.order).toEqual(["1"]);

    state = downloadsReducer(state, {
      type: "progress",
      event: {
        taskId: "1",
        downloadedBytes: 40,
        totalBytes: 100,
        bytesPerSecond: 5,
        status: "downloading",
      },
    });
    expect(state.byId["1"]?.downloadedBytes).toBe(40);
    expect(state.byId["1"]?.bytesPerSecond).toBe(5);
  });

  it("removes on changed null", () => {
    let state = downloadsReducer(initialDownloadsState, {
      type: "snapshot",
      downloads: [sample],
    });
    state = downloadsReducer(state, {
      type: "changed",
      event: { taskId: "1", download: null },
    });
    expect(state.order).toEqual([]);
    expect(state.byId["1"]).toBeUndefined();
  });
});
