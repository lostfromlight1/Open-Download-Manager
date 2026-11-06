package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// copyWithStall copies from src into dst while:
//   - detecting stalls (no bytes for StallTimeout)
//   - honouring context cancellation
//   - reporting progress occasionally
//
// already = bytes already on disk before this attempt (resume offset)
// total   = expected final size, or 0 if unknown
func copyWithStall(
	ctx context.Context,
	dst *os.File,
	src io.Reader,
	opts Options,
	already int64,
	total int64,
	attempt int,
) (int64, error) {
	// make allocates a slice with length=capacity=BufferSize.
	buf := make([]byte, opts.BufferSize)

	var written int64 // bytes written during THIS attempt only
	lastProgress := time.Now()
	start := time.Now()

	// report is a nested function (closure). It can read variables from the
	// surrounding function such as written, already, and opts.
	report := func(force bool) {
		if opts.OnProgress == nil && opts.ProgressWriter == nil {
			return
		}
		now := time.Now()
		if !force && now.Sub(lastProgress) < opts.ProgressInterval {
			return // throttle: skip if we reported too recently
		}
		lastProgress = now

		downloaded := already + written
		elapsed := now.Sub(start).Seconds()
		bps := 0.0
		if elapsed > 0 {
			bps = float64(written) / elapsed
		}

		ev := ProgressEvent{
			Downloaded:  downloaded,
			Total:       total,
			BytesPerSec: bps,
			Attempt:     attempt,
		}
		if opts.OnProgress != nil {
			opts.OnProgress(ev)
		}
		if opts.ProgressWriter != nil {
			// \r returns the cursor to the start of the line so the CLI can
			// overwrite the previous progress line instead of flooding output.
			if total > 0 {
				_, _ = fmt.Fprintf(opts.ProgressWriter, "\r%6.1f%%  %d/%d bytes  %.1f KB/s (attempt %d)",
					100*float64(downloaded)/float64(total), downloaded, total, bps/1024, attempt)
			} else {
				_, _ = fmt.Fprintf(opts.ProgressWriter, "\r%d bytes  %.1f KB/s (attempt %d)",
					downloaded, bps/1024, attempt)
			}
		}
	}

	for {
		if err := ctx.Err(); err != nil {
			return written, fmt.Errorf("%w: %v", ErrCancelled, err)
		}

		n, err := readWithStall(ctx, src, buf, opts.StallTimeout)
		if n > 0 {
			// Write only the bytes that were actually read: buf[:n].
			nw, werr := dst.Write(buf[:n])
			written += int64(nw)
			if werr != nil {
				return written, fmt.Errorf("write partial file: %w", werr)
			}
			if nw != n {
				return written, io.ErrShortWrite
			}
			report(false)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				// EOF means the server finished sending the body normally.
				report(true)
				if opts.ProgressWriter != nil {
					_, _ = fmt.Fprintln(opts.ProgressWriter) // finish the \r line
				}
				return written, nil
			}
			if errors.Is(err, ErrStalled) {
				return written, err
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return written, fmt.Errorf("%w: %v", ErrCancelled, err)
			}
			if errors.Is(err, io.ErrUnexpectedEOF) || isConnectionErr(err) {
				// Mid-body disconnect: caller can retry with Range from already+written.
				return written, fmt.Errorf("connection closed mid-body after %d bytes: %w", already+written, err)
			}
			return written, err
		}
	}
}

// readWithStall performs one Read with a maximum wait time.
//
// Go's io.Reader.Read blocks until data arrives. To detect stalls we:
//  1. start Read in a goroutine (lightweight concurrent function)
//  2. wait on either: context cancel, stall timer, or read result
//
// Using a buffered channel of size 1 (make(chan T, 1)) prevents the goroutine
// from blocking forever if we return early due to stall/cancel.
func readWithStall(ctx context.Context, src io.Reader, buf []byte, stall time.Duration) (int, error) {
	// Local struct type: only visible inside this function.
	type readResult struct {
		n   int
		err error
	}

	ch := make(chan readResult, 1)

	// go starts a new goroutine. It runs concurrently with this function.
	go func() {
		n, err := src.Read(buf)
		ch <- readResult{n: n, err: err}
	}()

	timer := time.NewTimer(stall)
	defer timer.Stop() // always stop timers to avoid leaks

	// select blocks until one case is ready.
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-timer.C:
		return 0, fmt.Errorf("%w: no data for %s", ErrStalled, stall)
	case rr := <-ch:
		return rr.n, rr.err
	}
}

// isConnectionErr does a best-effort check for transport failures.
// Exact error types differ by OS, so we inspect the message carefully.
func isConnectionErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, p := range []string{
		"connection reset",
		"broken pipe",
		"wsarecv",
		"forcibly closed",
		"eof",
	} {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}
