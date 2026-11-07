import { useCallback, useRef, useState } from "react";
import type { SpeedSample } from "./SpeedGraph";
import { isActiveStatus } from "./downloadTypes";

const MAX_SAMPLES = 48;

type DiskTracker = {
  bytes: number;
  at: number;
};

/**
 * Keeps a short rolling speed history per task for the Steam-style graph.
 * Disk write is estimated from downloadedBytes deltas (bytes actually reported written).
 */
export function useSpeedHistory() {
  const [byId, setById] = useState<Record<string, SpeedSample[]>>({});
  const diskRef = useRef<Record<string, DiskTracker>>({});

  const recordProgress = useCallback(
    (
      taskId: string,
      downloadedBytes: number,
      netBps: number | null | undefined,
      status: string,
    ) => {
      if (!isActiveStatus(status) && status !== "queued") {
        return;
      }
      const now = Date.now();
      const prev = diskRef.current[taskId];
      let diskBps = 0;
      if (prev && now > prev.at) {
        const dt = (now - prev.at) / 1000;
        if (dt > 0 && downloadedBytes >= prev.bytes) {
          diskBps = (downloadedBytes - prev.bytes) / dt;
        }
      }
      diskRef.current[taskId] = { bytes: downloadedBytes, at: now };
      const net = netBps != null && netBps > 0 ? netBps : 0;
      // If engine speed is missing, fall back to disk estimate for the net line.
      const sample: SpeedSample = {
        at: now,
        netBps: net > 0 ? net : diskBps,
        diskBps: diskBps > 0 ? diskBps : net,
      };
      setById((prevMap) => {
        const list = [...(prevMap[taskId] ?? []), sample];
        return {
          ...prevMap,
          [taskId]: list.length > MAX_SAMPLES ? list.slice(-MAX_SAMPLES) : list,
        };
      });
    },
    [],
  );

  const clearTask = useCallback((taskId: string) => {
    delete diskRef.current[taskId];
    setById((prev) => {
      if (!(taskId in prev)) return prev;
      const next = { ...prev };
      delete next[taskId];
      return next;
    });
  }, []);

  return { samplesById: byId, recordProgress, clearTask };
}
