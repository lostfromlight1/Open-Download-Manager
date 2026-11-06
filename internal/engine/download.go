package engine

import (
	// context carries cancellation (Ctrl+C) and deadlines across API calls.
	"context"

	// errors helps match sentinel errors (errors.Is) and typed errors (errors.As).
	"errors"

	// fmt builds error messages and formats Range headers.
	"fmt"

	// io provides EOF and related stream errors.
	"io"

	// net/http is the standard HTTP client/server package.
	"net/http"

	// os is used for files: create, append, sync, rename, remove, stat.
	"os"

	// path/filepath joins and inspects OS-specific paths (backslash on Windows).
	"path/filepath"

	// strconv parses integers from header strings (Content-Length, Content-Range).
	"strconv"

	// strings has TrimSpace, HasPrefix, Contains, Split helpers.
	"strings"

	// time is used for retry backoff delays.
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
)

// Result is returned when a download completes successfully.
//
// In Go, exported struct fields (Uppercase) are visible to other packages
// such as cmd/odm. Lowercase fields would be invisible outside `engine`.
type Result struct {
	Destination  string // final path after renaming away from ".part"
	BytesWritten int64  // total bytes in the final file
	Attempts     int    // how many try loops were needed (1 = first try worked)
	ETag         string // optional validator from the response (may be empty)
	LastModified string // optional validator from the response (may be empty)
}

// Download performs an HTTP/HTTPS download to Destination.
//
// Design rules:
//   - Stream bytes to disk (never hold the whole file in memory).
//   - Write to a ".part" file first; rename only after success.
//   - Retry transient failures with backoff.
//   - Resume with HTTP Range when a partial file already exists (single-stream).
//   - Phase 3: when Connections > 1, verify real 206 Range support, plan
//     non-overlapping segments, and fan out bounded workers. If verification
//     fails (or a contiguous .part already exists), fall back to single-stream.
//
// The first return value is a pointer (*Result). Pointers are nil-able, so we
// can return (nil, err) on failure. On success we return (&Result{...}, nil).
func Download(ctx context.Context, opts Options) (*Result, error) {
	// withDefaults fills zero values. We assign back into opts (a local copy
	// of the struct passed by value — mutating opts here does not change the caller).
	opts = opts.withDefaults()

	u, err := validateURL(opts.URL)
	if err != nil {
		// return nil, err is the idiomatic failure pair for (pointer, error).
		return nil, err
	}
	dest, err := resolveDestination(opts.Destination, u)
	if err != nil {
		return nil, err
	}

	// MkdirAll creates every missing parent directory (like `mkdir -p`).
	// 0o755 is octal permission: owner rwx, group rx, others rx.
	// filepath.Dir("C:\\a\\b.bin") → "C:\\a"
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return nil, fmt.Errorf("%w: create destination directory: %v", ErrInvalidDestination, err)
	}

	part := partPath(dest) // e.g. "file.bin.part"
	if opts.ForceRestart {
		// Ignore "file does not exist" by discarding Remove's error with `_`.
		// `_` is Go's blank identifier: "assign and throw away".
		_ = os.Remove(part)
		_ = os.Remove(dest)
	}

	client := opts.HTTPClient
	if client == nil {
		// Build a client from proxy + timeout options when the caller did not
		// inject one (tests sometimes inject a custom client).
		client, err = netx.NewClient(netx.ClientOptions{
			Proxy:                 opts.Proxy,
			ConnectTimeout:        opts.ConnectTimeout,
			ResponseHeaderTimeout: opts.ResponseHeaderTimeout,
			UserAgent:             opts.UserAgent,
		})
		if err != nil {
			return nil, err
		}
	}

	rawURL := u.String()

	// Segmented path: only when requested AND safe.
	// An existing contiguous .part is resumed via single-stream Range append —
	// multi-connection resume needs persisted per-segment offsets (manager DB).
	if opts.Connections > 1 {
		existing, err := existingSize(part)
		if err != nil {
			return nil, err
		}
		if existing == 0 {
			cap, verr := VerifyRangeSupport(ctx, client, rawURL, opts.UserAgent, opts.ExtraHeaders)
			if verr != nil && !errors.Is(verr, ErrRangeNotSupported) {
				// Transport / HTTP errors: honour normal retry via single-stream
				// only when the probe itself was inconclusive in a retryable way.
				if !retryable(verr) {
					return nil, verr
				}
			}
			if verr == nil && cap.Supported && cap.Total >= opts.MinSegmentBytes*2 {
				result, err := downloadSegmented(ctx, client, opts, rawURL, dest, part, cap)
				if err == nil {
					return result, nil
				}
				// If segmented mode discovers the server lied mid-transfer, fall back
				// once after clearing the sparse partial — safer than mixing modes.
				if errors.Is(err, ErrRangeNotSupported) {
					_ = os.Remove(part)
					return downloadSingle(ctx, client, opts, rawURL, dest, part)
				}
				return nil, err
			}
			// Range not verified / file too small → single-stream below.
		}
	}

	return downloadSingle(ctx, client, opts, rawURL, dest, part)
}

// downloadSingle is the Phase 1 attempt loop (one connection, optional resume).
func downloadSingle(
	ctx context.Context,
	client *http.Client,
	opts Options,
	rawURL string,
	dest string,
	part string,
) (*Result, error) {
	// Attempt loop: try once, then retry up to MaxRetries more times.
	// Example: MaxRetries=5 → attempts 1..6.
	//
	// `var lastErr error` declares a variable with the zero value of error: nil.
	var lastErr error
	for attempt := 1; attempt <= opts.MaxRetries+1; attempt++ {
		// ctx.Err() is non-nil when the context was cancelled or timed out.
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrCancelled, err)
		}

		result, err := attemptDownload(ctx, client, opts, rawURL, dest, part, attempt)
		if err == nil {
			return result, nil // success — leave the loop
		}
		lastErr = err

		// Stop if the error is permanent, or we have no retries left.
		if !retryable(err) || attempt > opts.MaxRetries {
			return nil, err
		}

		// Linear backoff: 1s, 2s, 3s, ... (RetryBackoff * attempt).
		// time.Duration is an int64 nanosecond count; multiplying by attempt scales it.
		backoff := opts.RetryBackoff * time.Duration(attempt)

		// select waits until one of its cases can run.
		// This either waits for backoff OR returns early on cancel.
		//
		// `<-ctx.Done()` receives from a channel that closes when ctx ends.
		// `time.After` returns a channel that receives once after the duration.
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("%w: %v", ErrCancelled, ctx.Err())
		case <-time.After(backoff):
			// timer fired; continue the for-loop into the next attempt
		}
	}
	return nil, lastErr
}

// attemptDownload performs one GET (possibly with Range) and writes to the .part file.
//
// Multi-line function signatures are normal in Go when there are many parameters.
func attemptDownload(
	ctx context.Context,
	client *http.Client,
	opts Options,
	rawURL string,
	dest string,
	part string,
	attempt int,
) (*Result, error) {
	existing, err := existingSize(part)
	if err != nil {
		return nil, err
	}

	// NewRequestWithContext ties the request lifetime to ctx.
	// If ctx is cancelled, in-flight I/O should abort.
	// nil body means "no request body" (normal for GET).
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	netx.ApplyUserAgent(req, opts.UserAgent)
	ApplyExtraHeaders(req, opts.ExtraHeaders)

	// HTTP Range asks the server for bytes starting at `existing`.
	// Example: bytes=1024- means "send from offset 1024 to the end".
	if existing > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existing))
	}

	resp, err := client.Do(req) // send the request and wait for headers
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %v", ErrCancelled, err)
		}
		// %w wraps the original error so callers can unwrap it later.
		return nil, fmt.Errorf("request failed: %w", err)
	}
	// defer schedules Close to run when this function returns (success or error).
	// Always close response bodies or you leak connections.
	defer resp.Body.Close()

	// Validate status codes carefully. Do not trust Accept-Ranges alone.
	// A switch without a tag (switch { case ... }) is like if/else-if chains.
	switch {
	case resp.StatusCode == http.StatusOK: // 200 = full body
		if existing > 0 {
			// Server ignored our Range request. Appending a full body would corrupt data.
			return nil, fmt.Errorf("%w: server returned HTTP 200 for a resume request (use --force-restart)", ErrResumeUnsupported)
		}

	case resp.StatusCode == http.StatusPartialContent: // 206 = ranged body
		if existing == 0 {
			return nil, fmt.Errorf("%w: unexpected 206 without a Range request", ErrHTTPStatus)
		}
		start, _, total, err := parseContentRange(resp.Header.Get("Content-Range"))
		if err != nil {
			return nil, err
		}
		if start != existing {
			return nil, fmt.Errorf("%w: Content-Range start %d != local size %d", ErrHTTPStatus, start, existing)
		}
		// total may be -1 when unknown ("*"); we still stream carefully.
		// Assigning to `_` documents that we intentionally ignore it here.
		_ = total

	case resp.StatusCode == http.StatusRequestedRangeNotSatisfiable: // 416
		// Sometimes this means we already have the whole file.
		info, statErr := os.Stat(part)
		if statErr == nil && resp.Header.Get("Content-Range") != "" {
			_, _, total, perr := parseContentRange(resp.Header.Get("Content-Range"))
			if perr == nil && total > 0 && info.Size() == total {
				return finalize(part, dest, total, attempt, resp)
			}
		}
		// &StatusError{...} creates a pointer to a StatusError value.
		return nil, &StatusError{StatusCode: resp.StatusCode, Message: "range not satisfiable"}

	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		// 429 and 5xx are usually temporary → retryable.
		return nil, &StatusError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(resp.Status)}

	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusForbidden:
		// Permanent client/resource errors → do not retry endlessly.
		return nil, &StatusError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(resp.Status)}

	default:
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return nil, &StatusError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(resp.Status)}
		}
	}

	// Figure out expected final size when the server tells us.
	expectedTotal := int64(0)
	if resp.StatusCode == http.StatusOK {
		if cl := resp.Header.Get("Content-Length"); cl != "" {
			// ParseInt(string, base, bitSize). base 10 = decimal; bitSize 64 = int64.
			if n, perr := strconv.ParseInt(cl, 10, 64); perr == nil && n >= 0 {
				expectedTotal = n
			}
		}
	} else if resp.StatusCode == http.StatusPartialContent {
		_, _, total, perr := parseContentRange(resp.Header.Get("Content-Range"))
		if perr == nil && total > 0 {
			expectedTotal = total
		}
	}

	// Open flags are bitmasks combined with |.
	// O_CREATE: create if missing; O_WRONLY: write-only; O_APPEND / O_TRUNC below.
	flag := os.O_CREATE | os.O_WRONLY
	if existing > 0 && resp.StatusCode == http.StatusPartialContent {
		flag |= os.O_APPEND // continue writing after existing bytes
	} else {
		flag |= os.O_TRUNC // overwrite from the start
		existing = 0
	}

	// 0o644 = owner rw, group r, others r (typical file permission).
	f, err := os.OpenFile(part, flag, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open partial file: %w", err)
	}

	// Stream the body to disk with stall detection and progress reporting.
	written, err := copyWithStall(ctx, f, resp.Body, opts, existing, expectedTotal, attempt)
	if err != nil {
		_ = f.Close() // best-effort close before returning the copy error
		return nil, err
	}
	totalWritten := existing + written

	if expectedTotal > 0 && totalWritten != expectedTotal {
		_ = f.Close()
		return nil, fmt.Errorf("%w: got %d bytes, expected %d", ErrIncomplete, totalWritten, expectedTotal)
	}

	// Sync flushes file data to durable storage before we rename.
	// Without Sync, a crash right after rename could leave an empty/truncated file
	// on some systems.
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("sync partial file: %w", err)
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	return finalize(part, dest, totalWritten, attempt, resp)
}

// finalize renames the .part file to the final destination.
func finalize(part, dest string, bytes int64, attempt int, resp *http.Response) (*Result, error) {
	// Remove an existing final file first (Windows often needs this before rename).
	if err := os.Remove(dest); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("remove existing destination: %w", err)
	}
	if err := os.Rename(part, dest); err != nil {
		return nil, fmt.Errorf("finalize download: %w", err)
	}
	// &Result{...} allocates and returns a pointer. Field names are explicit.
	return &Result{
		Destination:  dest,
		BytesWritten: bytes,
		Attempts:     attempt,
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
	}, nil
}

// existingSize returns the size of the partial file, or 0 if it does not exist.
func existingSize(part string) (int64, error) {
	info, err := os.Stat(part) // Stat reads file metadata without opening contents.
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil // no partial yet → start from byte 0
	}
	if err != nil {
		return 0, fmt.Errorf("stat partial file: %w", err)
	}
	if info.IsDir() {
		return 0, fmt.Errorf("%w: partial path is a directory", ErrInvalidDestination)
	}
	return info.Size(), nil
}

// parseContentRange parses headers like: "bytes 0-99/100" or "bytes 50-99/100".
//
// Returns inclusive start/end and total size. total is -1 when the server sends "*".
// end is -1 when the range unit is "*" (rare; "bytes */100").
func parseContentRange(h string) (start, end, total int64, err error) {
	h = strings.TrimSpace(h)
	if h == "" {
		return 0, 0, 0, fmt.Errorf("%w: missing Content-Range", ErrHTTPStatus)
	}
	if !strings.HasPrefix(h, "bytes ") {
		return 0, 0, 0, fmt.Errorf("%w: invalid Content-Range %q", ErrHTTPStatus, h)
	}

	rest := strings.TrimPrefix(h, "bytes ")
	parts := strings.Split(rest, "/") // ["0-99", "100"]
	if len(parts) != 2 {
		return 0, 0, 0, fmt.Errorf("%w: invalid Content-Range %q", ErrHTTPStatus, h)
	}

	total = int64(-1) // -1 means "unknown total" (when server sends "*")
	if parts[1] != "*" {
		total, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("%w: invalid Content-Range total: %v", ErrHTTPStatus, err)
		}
	}
	if parts[0] == "*" {
		return 0, -1, total, nil
	}

	se := strings.Split(parts[0], "-") // ["50", "99"]
	if len(se) != 2 {
		return 0, 0, 0, fmt.Errorf("%w: invalid Content-Range range %q", ErrHTTPStatus, h)
	}
	start, err = strconv.ParseInt(se[0], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%w: invalid Content-Range start: %v", ErrHTTPStatus, err)
	}
	end, err = strconv.ParseInt(se[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%w: invalid Content-Range end: %v", ErrHTTPStatus, err)
	}
	if end < start {
		return 0, 0, 0, fmt.Errorf("%w: Content-Range end %d < start %d", ErrHTTPStatus, end, start)
	}
	return start, end, total, nil
}

// retryable decides whether Download should try again.
func retryable(err error) bool {
	if err == nil {
		return false
	}

	// Permanent / user-driven errors: do not retry.
	if errors.Is(err, ErrCancelled) || errors.Is(err, ErrInvalidURL) || errors.Is(err, ErrInvalidDestination) {
		return false
	}
	if errors.Is(err, ErrResumeUnsupported) || errors.Is(err, ErrRangeNotSupported) {
		return false
	}

	// errors.As checks whether any error in the wrap chain has type *StatusError.
	// Pass a pointer to a pointer: &status. If found, status points at that error.
	var status *StatusError
	if errors.As(err, &status) {
		if status.StatusCode == http.StatusTooManyRequests || status.StatusCode >= 500 {
			return true
		}
		if status.StatusCode == http.StatusNotFound || status.StatusCode == http.StatusGone || status.StatusCode == http.StatusForbidden {
			return false
		}
		return false
	}

	// Network / stall / truncated body: worth another attempt (often with Range).
	if errors.Is(err, ErrStalled) || errors.Is(err, ErrIncomplete) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	msg := err.Error()
	if strings.Contains(msg, "request failed") || strings.Contains(msg, "connection") || strings.Contains(msg, "EOF") {
		return true
	}

	// Default for unknown transport failures: retry (Yangon-style flaky networks).
	return true
}
