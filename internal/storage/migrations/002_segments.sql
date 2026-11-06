-- Migration 002: segment rows for multi-connection downloads (Phase 3).
--
-- Segments are inclusive byte ranges. The engine still verifies HTTP 206
-- Content-Range before writing; this table records planner state for resume.

CREATE TABLE IF NOT EXISTS segments (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    start_byte INTEGER NOT NULL,
    end_byte INTEGER NOT NULL,
    downloaded_bytes INTEGER NOT NULL DEFAULT 0,
    state TEXT NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    CHECK (start_byte <= end_byte),
    CHECK (downloaded_bytes >= 0),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_segments_task_id ON segments(task_id);
