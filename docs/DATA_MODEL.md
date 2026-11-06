# Initial Data Model

Phase 2 ships the initial SQLite schema in `internal/storage/migrations/001_init.sql`.
Phase 3 adds `internal/storage/migrations/002_segments.sql` for multi-connection downloads.

## DownloadTask

- `id`: UUID primary key.
- `url`: original URL; treat as potentially sensitive.
- `display_name`: sanitized display filename.
- `destination_path`: selected output path.
- `state`: explicit lifecycle enum.
- `total_bytes`: nullable integer.
- `downloaded_bytes`: integer.
- `etag`: nullable string.
- `last_modified`: nullable string.
- `created_at`, `updated_at`: timestamps.
- `error_code`: nullable stable code.
- `error_message`: nullable sanitized user-facing message.
- `retry_count`: integer.

## DownloadSegment

- `id`: UUID primary key.
- `task_id`: foreign key.
- `start_byte`: inclusive offset.
- `end_byte`: inclusive offset.
- `downloaded_bytes`: integer.
- `state`: pending/running/completed/failed.
- `attempt_count`: integer.

Constraints:
- Segment start must be less than or equal to end.
- Segments for a task must not overlap.
- Segment byte counts must be within their assigned lengths.
- Use transactions when updating task and segment state together.

## AppSettings

- `download_directory`
- `max_active_downloads`
- `max_connections_per_download`
- `max_connections_per_host`
- `max_retries`
- `bandwidth_limit_bytes_per_second` (nullable)
- `theme`
- `ui.notify_on_complete` (opt-in completion notification; default false)
- `ui.close_to_tray` (hide to tray on window close; default true — tray Quit stops downloads)
- `browser.integration_enabled` (opt-in native messaging; default false)
- `browser.chrome_extension_id` (Chrome/Edge host `allowed_origins`)

Do not store secrets in this table.
