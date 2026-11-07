-- Migration 003: YouTube / media task metadata (Phase 10).
ALTER TABLE tasks ADD COLUMN media_source TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN media_format_id TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN media_webpage_url TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN media_playlist_id TEXT NOT NULL DEFAULT '';
