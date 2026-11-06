package engine

import (
	// context carries cancellation into the Range probe request.
	"context"

	// errors matches sentinel / wrapped errors.
	"errors"

	// fmt builds Range headers and error messages.
	"fmt"

	// io drains a tiny probe body so connections can be reused.
	"io"

	// net/http is used for the probe GET.
	"net/http"

	// strconv parses Content-Length when a server ignores Range.
	"strconv"

	// strings trims and inspects header values.
	"strings"

	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
)

// RangeCapability is the result of verifying real Range support.
//
// We never trust Accept-Ranges alone. Supported is true only after a 206 with
// a Content-Range whose boundaries and total length check out.
type RangeCapability struct {
	Supported    bool
	Total        int64  // known remote size; 0 when unknown / unsupported
	ETag         string // may be empty
	LastModified string // may be empty
}

// VerifyRangeSupport probes the URL with "Range: bytes=0-0".
//
// Learning notes:
//   - A probe asks for ONE byte. If the server truly supports ranges it replies
//     206 Partial Content and Content-Range: bytes 0-0/TOTAL.
//   - If it ignores Range and returns 200, multi-connection mode is unsafe.
//   - Malformed Content-Range must be treated as "not supported".
func VerifyRangeSupport(ctx context.Context, client *http.Client, rawURL string, userAgent string, extra http.Header) (RangeCapability, error) {
	if client == nil {
		return RangeCapability{}, fmt.Errorf("http client is required")
	}
	if err := ctx.Err(); err != nil {
		return RangeCapability{}, fmt.Errorf("%w: %v", ErrCancelled, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return RangeCapability{}, err
	}
	netx.ApplyUserAgent(req, userAgent)
	ApplyExtraHeaders(req, extra)
	// Ask for the first byte only — cheap probe that still exercises Range.
	req.Header.Set("Range", "bytes=0-0")

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return RangeCapability{}, fmt.Errorf("%w: %v", ErrCancelled, err)
		}
		return RangeCapability{}, fmt.Errorf("range probe failed: %w", err)
	}
	defer resp.Body.Close()
	// Drain a tiny body so the connection can be reused; ignore short-read errors.
	_, _ = io.CopyN(io.Discard, resp.Body, 8)

	etag := resp.Header.Get("ETag")
	lm := resp.Header.Get("Last-Modified")

	switch resp.StatusCode {
	case http.StatusPartialContent: // 206
		start, end, total, perr := parseContentRange(resp.Header.Get("Content-Range"))
		if perr != nil {
			return RangeCapability{ETag: etag, LastModified: lm}, fmt.Errorf("%w: %v", ErrRangeNotSupported, perr)
		}
		if start != 0 || end != 0 {
			return RangeCapability{ETag: etag, LastModified: lm}, fmt.Errorf("%w: probe Content-Range want 0-0, got %d-%d", ErrRangeNotSupported, start, end)
		}
		if total <= 0 {
			// "*" total is not enough to plan non-overlapping segments safely.
			return RangeCapability{ETag: etag, LastModified: lm}, fmt.Errorf("%w: probe Content-Range missing total size", ErrRangeNotSupported)
		}
		return RangeCapability{
			Supported:    true,
			Total:        total,
			ETag:         etag,
			LastModified: lm,
		}, nil

	case http.StatusOK: // 200 — server ignored Range
		// Optional: learn size from Content-Length for callers that still want it.
		var total int64
		if cl := resp.Header.Get("Content-Length"); cl != "" {
			if n, perr := strconv.ParseInt(cl, 10, 64); perr == nil && n >= 0 {
				total = n
			}
		}
		return RangeCapability{
			Supported:    false,
			Total:        total,
			ETag:         etag,
			LastModified: lm,
		}, nil

	case http.StatusRequestedRangeNotSatisfiable: // 416
		// Empty file edge case: some servers 416 a bytes=0-0 probe.
		_, _, total, perr := parseContentRange(resp.Header.Get("Content-Range"))
		if perr == nil && total == 0 {
			return RangeCapability{Supported: true, Total: 0, ETag: etag, LastModified: lm}, nil
		}
		return RangeCapability{ETag: etag, LastModified: lm}, fmt.Errorf("%w: HTTP 416 on range probe", ErrRangeNotSupported)

	default:
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			return RangeCapability{}, &StatusError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(resp.Status)}
		}
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusForbidden {
			return RangeCapability{}, &StatusError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(resp.Status)}
		}
		return RangeCapability{ETag: etag, LastModified: lm}, fmt.Errorf("%w: unexpected probe status HTTP %d", ErrRangeNotSupported, resp.StatusCode)
	}
}
