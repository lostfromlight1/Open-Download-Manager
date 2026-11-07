import { describe, expect, it } from "vitest";
import type { DownloadDTO } from "./downloadTypes";
import { partitionSelection } from "./selectionHelpers";

function row(
  id: string,
  status: DownloadDTO["status"],
): DownloadDTO {
  return {
    id,
    url: "https://example.com/f",
    fileName: `${id}.bin`,
    destinationPath: `/tmp/${id}.bin`,
    status,
    downloadedBytes: 0,
    totalBytes: 100,
    createdAt: "",
    updatedAt: "",
  };
}

describe("partitionSelection", () => {
  it("splits pause / resume / cancel eligibility", () => {
    const selected = [
      row("a", "downloading"),
      row("b", "paused"),
      row("c", "completed"),
      row("d", "queued"),
      row("e", "failed"),
    ];
    const parts = partitionSelection(selected);
    expect(parts.pauseable.map((d) => d.id)).toEqual(["a", "d"]);
    expect(parts.resumeable.map((d) => d.id)).toEqual(["b", "e"]);
    expect(parts.cancelable.map((d) => d.id)).toEqual(["a", "b", "d"]);
  });
});
