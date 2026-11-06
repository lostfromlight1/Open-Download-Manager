package storage

import (
	"context"
	"fmt"

	"github.com/lostfromlight1/Open-Download-Manager/internal/task"
)

// SegmentState is the lifecycle of one byte-range worker assignment.
type SegmentState string

const (
	SegmentPending   SegmentState = "pending"
	SegmentRunning   SegmentState = "running"
	SegmentCompleted SegmentState = "completed"
	SegmentFailed    SegmentState = "failed"
)

// Segment is one non-overlapping byte range belonging to a task.
type Segment struct {
	ID              string
	TaskID          string
	StartByte       int64
	EndByte         int64 // inclusive
	DownloadedBytes int64
	State           SegmentState
	AttemptCount    int
}

// Length returns the inclusive span size.
func (s Segment) Length() int64 {
	if s.EndByte < s.StartByte {
		return 0
	}
	return s.EndByte - s.StartByte + 1
}

// ReplaceSegments deletes existing rows for a task and inserts the new plan.
// Used when (re)starting a segmented download.
func (s *Store) ReplaceSegments(ctx context.Context, taskID string, segs []Segment) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM segments WHERE task_id = ?`, taskID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("clear segments: %w", err)
	}
	for _, seg := range segs {
		if seg.ID == "" {
			seg.ID = task.NewID()
		}
		if seg.TaskID == "" {
			seg.TaskID = taskID
		}
		if seg.State == "" {
			seg.State = SegmentPending
		}
		if seg.EndByte < seg.StartByte {
			_ = tx.Rollback()
			return fmt.Errorf("segment end < start")
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO segments(id, task_id, start_byte, end_byte, downloaded_bytes, state, attempt_count)
VALUES(?, ?, ?, ?, ?, ?, ?)`,
			seg.ID, seg.TaskID, seg.StartByte, seg.EndByte, seg.DownloadedBytes, string(seg.State), seg.AttemptCount,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert segment: %w", err)
		}
	}
	return tx.Commit()
}

// ListSegments returns segments for a task ordered by start_byte.
func (s *Store) ListSegments(ctx context.Context, taskID string) ([]Segment, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, task_id, start_byte, end_byte, downloaded_bytes, state, attempt_count
FROM segments WHERE task_id = ? ORDER BY start_byte ASC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list segments: %w", err)
	}
	defer rows.Close()

	var out []Segment
	for rows.Next() {
		var seg Segment
		var state string
		if err := rows.Scan(&seg.ID, &seg.TaskID, &seg.StartByte, &seg.EndByte, &seg.DownloadedBytes, &state, &seg.AttemptCount); err != nil {
			return nil, err
		}
		seg.State = SegmentState(state)
		out = append(out, seg)
	}
	return out, rows.Err()
}

// UpdateSegmentProgress updates downloaded bytes / state / attempts for one row.
func (s *Store) UpdateSegmentProgress(ctx context.Context, seg Segment) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE segments
SET downloaded_bytes = ?, state = ?, attempt_count = ?
WHERE id = ?`, seg.DownloadedBytes, string(seg.State), seg.AttemptCount, seg.ID)
	if err != nil {
		return fmt.Errorf("update segment: %w", err)
	}
	return nil
}

// DeleteSegmentsForTask removes all segment rows for a task.
func (s *Store) DeleteSegmentsForTask(ctx context.Context, taskID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM segments WHERE task_id = ?`, taskID)
	if err != nil {
		return fmt.Errorf("delete segments: %w", err)
	}
	return nil
}
