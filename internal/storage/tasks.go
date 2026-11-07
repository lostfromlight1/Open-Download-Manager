package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/task"
)

// InsertTask writes a new task row.
//
// All queries use `?` placeholders. The driver binds Go values safely, which
// prevents SQL injection even if a URL contains quotes.
func (s *Store) InsertTask(ctx context.Context, t task.Task) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO tasks(
    id, url, display_name, destination_path, state,
    total_bytes, downloaded_bytes, etag, last_modified,
    error_code, error_message, retry_count, created_at, updated_at,
    media_source, media_format_id, media_webpage_url, media_playlist_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.URL, t.DisplayName, t.DestinationPath, string(t.State),
		nullInt64(t.TotalBytes), t.DownloadedBytes, t.ETag, t.LastModified,
		t.ErrorCode, t.ErrorMessage, t.RetryCount,
		t.CreatedAt.UTC().Format(time.RFC3339Nano),
		t.UpdatedAt.UTC().Format(time.RFC3339Nano),
		t.MediaSource, t.MediaFormatID, t.MediaWebpageURL, t.MediaPlaylistID,
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

// UpdateTask overwrites mutable fields for an existing id.
func (s *Store) UpdateTask(ctx context.Context, t task.Task) error {
	res, err := s.db.ExecContext(ctx, `
UPDATE tasks SET
    url=?, display_name=?, destination_path=?, state=?,
    total_bytes=?, downloaded_bytes=?, etag=?, last_modified=?,
    error_code=?, error_message=?, retry_count=?, updated_at=?,
    media_source=?, media_format_id=?, media_webpage_url=?, media_playlist_id=?
WHERE id=?`,
		t.URL, t.DisplayName, t.DestinationPath, string(t.State),
		nullInt64(t.TotalBytes), t.DownloadedBytes, t.ETag, t.LastModified,
		t.ErrorCode, t.ErrorMessage, t.RetryCount,
		t.UpdatedAt.UTC().Format(time.RFC3339Nano),
		t.MediaSource, t.MediaFormatID, t.MediaWebpageURL, t.MediaPlaylistID,
		t.ID,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return task.ErrNotFound
	}
	return nil
}

// GetTask loads one task by id.
func (s *Store) GetTask(ctx context.Context, id string) (task.Task, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, url, display_name, destination_path, state,
       total_bytes, downloaded_bytes, etag, last_modified,
       error_code, error_message, retry_count, created_at, updated_at,
       media_source, media_format_id, media_webpage_url, media_playlist_id
FROM tasks WHERE id=?`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return task.Task{}, task.ErrNotFound
	}
	return t, err
}

// ListTasks returns tasks ordered by updated_at descending (newest first).
func (s *Store) ListTasks(ctx context.Context) ([]task.Task, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, url, display_name, destination_path, state,
       total_bytes, downloaded_bytes, etag, last_modified,
       error_code, error_message, retry_count, created_at, updated_at,
       media_source, media_format_id, media_webpage_url, media_playlist_id
FROM tasks
ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var out []task.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListByStates returns tasks whose state is in the given set, oldest first
// (fair queue order).
func (s *Store) ListByStates(ctx context.Context, states ...task.State) ([]task.Task, error) {
	if len(states) == 0 {
		return nil, nil
	}

	// Build `IN (?,?,?)` dynamically. Still parameterized — never splice values.
	placeholders := make([]string, len(states))
	args := make([]any, len(states))
	for i, st := range states {
		placeholders[i] = "?"
		args[i] = string(st)
	}
	q := fmt.Sprintf(`
SELECT id, url, display_name, destination_path, state,
       total_bytes, downloaded_bytes, etag, last_modified,
       error_code, error_message, retry_count, created_at, updated_at,
       media_source, media_format_id, media_webpage_url, media_playlist_id
FROM tasks
WHERE state IN (%s)
ORDER BY created_at ASC`, joinComma(placeholders))

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list by states: %w", err)
	}
	defer rows.Close()

	var out []task.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// DeleteTask removes the task row. Files on disk are the caller's responsibility.
func (s *Store) DeleteTask(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return task.ErrNotFound
	}
	return nil
}

// rowScanner is implemented by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(row rowScanner) (task.Task, error) {
	var (
		t          task.Task
		state      string
		total      sql.NullInt64
		createdRaw string
		updatedRaw string
	)
	err := row.Scan(
		&t.ID, &t.URL, &t.DisplayName, &t.DestinationPath, &state,
		&total, &t.DownloadedBytes, &t.ETag, &t.LastModified,
		&t.ErrorCode, &t.ErrorMessage, &t.RetryCount, &createdRaw, &updatedRaw,
		&t.MediaSource, &t.MediaFormatID, &t.MediaWebpageURL, &t.MediaPlaylistID,
	)
	if err != nil {
		return task.Task{}, err
	}
	t.State = task.State(state)
	if total.Valid {
		v := total.Int64
		t.TotalBytes = &v
	}
	t.CreatedAt, err = time.Parse(time.RFC3339Nano, createdRaw)
	if err != nil {
		return task.Task{}, fmt.Errorf("parse created_at: %w", err)
	}
	t.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedRaw)
	if err != nil {
		return task.Task{}, fmt.Errorf("parse updated_at: %w", err)
	}
	return t, nil
}

func nullInt64(v *int64) any {
	if v == nil {
		return nil // becomes SQL NULL
	}
	return *v
}

func joinComma(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += "," + parts[i]
	}
	return out
}
