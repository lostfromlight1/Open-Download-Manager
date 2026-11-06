// Package engine implements ODM's download transfer logic.
//
// This package must NOT import Wails. The CLI and desktop app both call into
// engine so there is one source of truth for downloading.
package engine

import (
	"errors"
	"fmt"
)

// Sentinel errors are package-level variables created with errors.New.
//
// Callers can write:
//
//	if errors.Is(err, engine.ErrInvalidURL) { ... }
//
// This is preferred over comparing error message strings.
var (
	// ErrInvalidURL means the URL is missing, malformed, or uses an unsupported scheme.
	ErrInvalidURL = errors.New("invalid download URL")

	// ErrInvalidDestination means the output path is unsafe or unusable.
	ErrInvalidDestination = errors.New("invalid destination path")

	// ErrHTTPStatus means the server returned a non-success status.
	ErrHTTPStatus = errors.New("unexpected HTTP status")

	// ErrResumeUnsupported means a partial file exists but the server did not honor Range.
	ErrResumeUnsupported = errors.New("resume not supported by server")

	// ErrIncomplete means the transfer finished without all expected bytes.
	ErrIncomplete = errors.New("incomplete download")

	// ErrStalled means no bytes were received within the stall timeout.
	ErrStalled = errors.New("download stalled")

	// ErrCancelled means the caller cancelled the download (context cancelled).
	ErrCancelled = errors.New("download cancelled")

	// ErrRangeNotSupported means we could not verify real HTTP 206 Range behavior.
	// Callers should fall back to single-stream rather than inventing segments.
	ErrRangeNotSupported = errors.New("HTTP Range not verified")

	// ErrSegmentOverlap means a planner bug produced overlapping byte ranges.
	ErrSegmentOverlap = errors.New("segment ranges overlap")
)

// StatusError is a custom error type that also carries an HTTP status code.
//
// Any type that implements Error() string satisfies the built-in error interface.
type StatusError struct {
	StatusCode int
	Message    string
}

// Error makes StatusError implement the error interface.
//
// The receiver is (e *StatusError) — a pointer receiver — so we can call the
// method even when we only have a pointer, and avoid copying the struct.
func (e *StatusError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: HTTP %d: %s", ErrHTTPStatus, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s: HTTP %d", ErrHTTPStatus, e.StatusCode)
}

// Unwrap lets errors.Is(err, ErrHTTPStatus) succeed for StatusError values.
//
// Go's error wrapping convention: if Unwrap returns an inner error, helpers
// like errors.Is walk the chain.
func (e *StatusError) Unwrap() error {
	return ErrHTTPStatus
}
