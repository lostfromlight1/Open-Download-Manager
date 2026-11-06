package engine

import (
	// context carries cancel into every segment worker.
	"context"

	// errors matches cancel / sentinel errors.
	"errors"

	// fmt builds Range headers and error text.
	"fmt"

	// io detects EOF / short writes while streaming a segment.
	"io"

	// net/http issues per-segment GET requests.
	"net/http"

	// os opens the shared .part file for random-access WriteAt.
	"os"

	// sync coordinates progress and first-error across workers.
	"sync"

	// sync/atomic aggregates downloaded bytes without a coarse lock on every byte.
	"sync/atomic"

	// time is used for retry backoff between segment attempts.
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
)

// downloadSegmented runs a verified multi-connection download into `part`.
//
// Safety rules:
//   - Only called after VerifyRangeSupport succeeded with a known total.
//   - Each worker writes ONLY its [Start, End] via WriteAt (no overlap).
//   - Each worker opens its own file handle so concurrent WriteAt stays simple.
func downloadSegmented(
	ctx context.Context,
	client *http.Client,
	opts Options,
	rawURL string,
	dest string,
	part string,
	cap RangeCapability,
) (*Result, error) {
	if cap.Total < 0 {
		return nil, fmt.Errorf("%w: invalid total", ErrRangeNotSupported)
	}
	if cap.Total == 0 {
		// Empty remote file: create an empty final file.
		f, err := os.OpenFile(part, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("create empty partial: %w", err)
		}
		_ = f.Close()
		return finalizeEmpty(part, dest, cap)
	}

	segs, err := PlanSegments(cap.Total, opts.Connections, opts.MinSegmentBytes)
	if err != nil {
		return nil, err
	}
	// Planner fell back to a single span — use the simpler single-stream path.
	if len(segs) <= 1 {
		return downloadSingle(ctx, client, opts, rawURL, dest, part)
	}

	// Pre-size the partial file so WriteAt positions are valid on all platforms.
	if err := preallocatePart(part, cap.Total); err != nil {
		return nil, err
	}

	var (
		downloaded atomic.Int64
		firstErr   error
		errOnce    sync.Once
		wg         sync.WaitGroup
	)
	setErr := func(e error) {
		if e == nil {
			return
		}
		errOnce.Do(func() { firstErr = e })
	}

	// Bound parallelism with a buffered semaphore channel.
	// Sending into a full channel blocks until another worker finishes.
	sem := make(chan struct{}, opts.Connections)
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	reportProgress := func(force bool) {
		if opts.OnProgress == nil && opts.ProgressWriter == nil {
			return
		}
		got := downloaded.Load()
		ev := ProgressEvent{
			Downloaded: got,
			Total:      cap.Total,
			Attempt:    1,
		}
		if opts.OnProgress != nil {
			opts.OnProgress(ev)
		}
		if opts.ProgressWriter != nil && force {
			_, _ = fmt.Fprintf(opts.ProgressWriter, "\r%6.1f%%  %d/%d bytes (segmented x%d)",
				100*float64(got)/float64(cap.Total), got, cap.Total, len(segs))
		}
	}

	for i, seg := range segs {
		wg.Add(1)
		go func(idx int, seg ByteRange) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-workerCtx.Done():
				setErr(fmt.Errorf("%w: %v", ErrCancelled, workerCtx.Err()))
				return
			}

			err := downloadOneSegment(workerCtx, client, opts, rawURL, part, seg, &downloaded, func() {
				reportProgress(false)
			})
			if err != nil {
				setErr(err)
				cancelWorkers() // stop siblings; partial file stays for retry/restart
			}
		}(i, seg)
	}
	wg.Wait()
	reportProgress(true)
	if opts.ProgressWriter != nil {
		_, _ = fmt.Fprintln(opts.ProgressWriter)
	}

	if firstErr != nil {
		// Preallocate makes len(.part)==total even when bytes are missing.
		// Drop the partial so a later resume cannot mistake size for progress.
		_ = os.Remove(part)
		return nil, firstErr
	}
	if got := downloaded.Load(); got != cap.Total {
		_ = os.Remove(part)
		return nil, fmt.Errorf("%w: segmented got %d bytes, expected %d", ErrIncomplete, got, cap.Total)
	}

	// Sync via a dedicated handle before rename.
	f, err := os.OpenFile(part, os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("reopen partial for sync: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("sync partial file: %w", err)
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	return finalizeSegmented(part, dest, cap.Total, cap)
}

func finalizeEmpty(part, dest string, cap RangeCapability) (*Result, error) {
	res, err := finalize(part, dest, 0, 1, &http.Response{
		Header: http.Header{
			"ETag":          []string{cap.ETag},
			"Last-Modified": []string{cap.LastModified},
		},
	})
	return res, err
}

func finalizeSegmented(part, dest string, total int64, cap RangeCapability) (*Result, error) {
	return finalize(part, dest, total, 1, &http.Response{
		Header: http.Header{
			"ETag":          []string{cap.ETag},
			"Last-Modified": []string{cap.LastModified},
		},
	})
}

// preallocatePart creates/truncates the .part file to exactly `size` bytes.
func preallocatePart(part string, size int64) error {
	f, err := os.OpenFile(part, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open partial for preallocate: %w", err)
	}
	defer f.Close()
	if err := f.Truncate(size); err != nil {
		return fmt.Errorf("truncate partial to %d: %w", size, err)
	}
	return nil
}

// downloadOneSegment fetches one inclusive range and WriteAts it into part.
func downloadOneSegment(
	ctx context.Context,
	client *http.Client,
	opts Options,
	rawURL string,
	part string,
	seg ByteRange,
	downloaded *atomic.Int64,
	onBytes func(),
) error {
	var lastErr error
	for attempt := 1; attempt <= opts.MaxRetries+1; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("%w: %v", ErrCancelled, err)
		}
		written, err := attemptSegment(ctx, client, opts, rawURL, part, seg, downloaded, onBytes)
		if err == nil {
			return nil
		}
		// Undo progress from a failed attempt so retries do not double-count.
		if written > 0 {
			downloaded.Add(-written)
		}
		lastErr = err
		if !retryable(err) || attempt > opts.MaxRetries {
			return err
		}
		backoff := opts.RetryBackoff * time.Duration(attempt)
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %v", ErrCancelled, ctx.Err())
		case <-time.After(backoff):
		}
	}
	return lastErr
}

func attemptSegment(
	ctx context.Context,
	client *http.Client,
	opts Options,
	rawURL string,
	part string,
	seg ByteRange,
	downloaded *atomic.Int64,
	onBytes func(),
) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	netx.ApplyUserAgent(req, opts.UserAgent)
	// Closed Range: bytes=START-END (inclusive).
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", seg.Start, seg.End))

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 0, fmt.Errorf("%w: %v", ErrCancelled, err)
		}
		return 0, fmt.Errorf("segment request failed: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusPartialContent:
		// ok — continue validation below
	case resp.StatusCode == http.StatusOK:
		// Server stopped honouring Range mid-job; do NOT write a full body at an offset.
		return 0, fmt.Errorf("%w: server returned HTTP 200 for segment %d-%d", ErrRangeNotSupported, seg.Start, seg.End)
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return 0, &StatusError{StatusCode: resp.StatusCode, Message: resp.Status}
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusForbidden:
		return 0, &StatusError{StatusCode: resp.StatusCode, Message: resp.Status}
	default:
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return 0, &StatusError{StatusCode: resp.StatusCode, Message: resp.Status}
		}
		return 0, fmt.Errorf("%w: unexpected HTTP %d for segment", ErrHTTPStatus, resp.StatusCode)
	}

	start, end, total, err := parseContentRange(resp.Header.Get("Content-Range"))
	if err != nil {
		return 0, err
	}
	if start != seg.Start || end != seg.End {
		return 0, fmt.Errorf("%w: Content-Range %d-%d != requested %d-%d", ErrHTTPStatus, start, end, seg.Start, seg.End)
	}
	if total > 0 && total < seg.End+1 {
		return 0, fmt.Errorf("%w: Content-Range total %d shorter than segment end %d", ErrHTTPStatus, total, seg.End)
	}

	// Each worker uses its own handle; WriteAt positions must not overlap.
	f, err := os.OpenFile(part, os.O_RDWR, 0o644)
	if err != nil {
		return 0, fmt.Errorf("open partial for segment: %w", err)
	}
	defer f.Close()

	expect := seg.Length()
	written, err := copyAtWithStall(ctx, f, resp.Body, opts, seg.Start, expect, downloaded, onBytes)
	if err != nil {
		return written, err
	}
	if written != expect {
		return written, fmt.Errorf("%w: segment %d-%d wrote %d, want %d", ErrIncomplete, seg.Start, seg.End, written, expect)
	}
	return written, nil
}

// copyAtWithStall streams src into dst at absolute file offset `at`.
// It increments `downloaded` for every byte committed so progress stays truthful.
func copyAtWithStall(
	ctx context.Context,
	dst *os.File,
	src io.Reader,
	opts Options,
	at int64,
	expect int64,
	downloaded *atomic.Int64,
	onBytes func(),
) (int64, error) {
	buf := make([]byte, opts.BufferSize)
	var written int64
	offset := at

	for written < expect {
		if err := ctx.Err(); err != nil {
			return written, fmt.Errorf("%w: %v", ErrCancelled, err)
		}

		want := int64(len(buf))
		if remain := expect - written; remain < want {
			want = remain
		}
		n, err := readWithStall(ctx, src, buf[:want], opts.StallTimeout)
		if n > 0 {
			nw, werr := dst.WriteAt(buf[:n], offset)
			if werr != nil {
				return written, fmt.Errorf("writeat partial file: %w", werr)
			}
			if nw != n {
				return written, io.ErrShortWrite
			}
			offset += int64(nw)
			written += int64(nw)
			downloaded.Add(int64(nw))
			if onBytes != nil {
				onBytes()
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if errors.Is(err, ErrStalled) {
				return written, err
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return written, fmt.Errorf("%w: %v", ErrCancelled, err)
			}
			if errors.Is(err, io.ErrUnexpectedEOF) || isConnectionErr(err) {
				return written, fmt.Errorf("connection closed mid-segment after %d bytes: %w", written, err)
			}
			return written, err
		}
	}
	return written, nil
}
