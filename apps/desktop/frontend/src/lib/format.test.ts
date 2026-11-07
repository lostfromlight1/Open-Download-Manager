import { describe, expect, it } from "vitest";
import {
  etaSeconds,
  formatBytes,
  formatMbps,
  formatSpeed,
  progressPercent,
} from "./format";

describe("format helpers", () => {
  it("formats bytes", () => {
    expect(formatBytes(512)).toBe("512 B");
    expect(formatBytes(null)).toBe("—");
  });

  it("does not invent speed", () => {
    expect(formatSpeed(null)).toBe("—");
    expect(formatSpeed(0)).toBe("—");
  });

  it("formats megabits from bytes/sec", () => {
    expect(formatMbps(null)).toBe("—");
    expect(formatMbps(0)).toBe("—");
    // 1_250_000 B/s = 10 Mbps
    expect(formatMbps(1_250_000)).toBe("10.0 Mbps");
  });

  it("computes percent and eta only when known", () => {
    expect(progressPercent(50, 100)).toBe(50);
    expect(progressPercent(50, null)).toBeNull();
    expect(etaSeconds(50, 100, 10)).toBe(5);
    expect(etaSeconds(50, null, 10)).toBeNull();
  });
});

