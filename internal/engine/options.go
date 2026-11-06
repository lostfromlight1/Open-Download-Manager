package engine

import (
	"io"
	"net/http"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
	"github.com/lostfromlight1/Open-Download-Manager/internal/version"
)

// ProgressEvent is a throttled snapshot of transfer progress.
//
// We send snapshots (not every byte) so UIs/CLIs stay responsive.
type ProgressEvent struct {
	Downloaded  int64   // bytes on disk so far (including resumed bytes)
	Total       int64   // 0 when the server did not tell us the size
	BytesPerSec float64 // rough speed for the current attempt
	Attempt     int     // 1-based attempt counter
}

// Options configures a download (single-stream or segmented).
//
// Passing a struct keeps the Download() signature stable as we add settings.
type Options struct {
	URL         string
	Destination string // final path; we write to Destination+".part" until success

	// ForceRestart deletes any existing .part / destination and starts from byte 0.
	ForceRestart bool

	// Connections is the max parallel HTTP Range workers for one file.
	// 1 (default) keeps the classic single-stream path.
	// Values > 1 enable segmented mode only after Range support is verified.
	Connections int

	// MinSegmentBytes is the smallest segment we bother splitting into.
	// Tiny files always fall back to single-stream even when Connections > 1.
	MinSegmentBytes int64

	// Proxy selects system / off / manual proxy behavior.
	Proxy netx.ProxyConfig

	ConnectTimeout        time.Duration
	ResponseHeaderTimeout time.Duration
	StallTimeout          time.Duration // max time between received byte chunks
	MaxRetries            int           // retries AFTER the first attempt (0 = try once)
	RetryBackoff          time.Duration // base delay; multiplied by attempt number
	ProgressInterval      time.Duration // how often to emit progress
	BufferSize            int           // read buffer size in bytes

	UserAgent  string
	HTTPClient *http.Client // optional; created from Proxy/timeouts when nil

	// ExtraHeaders are applied to outbound GETs (allowlisted: Cookie, Referer).
	// Used for one-shot browser auth handoff; never persisted by the engine.
	ExtraHeaders http.Header

	// OnProgress is an optional callback (function value). nil means "no callback".
	OnProgress func(ProgressEvent)

	// ProgressWriter receives human-readable progress text (CLI uses os.Stderr).
	// io.Writer is an interface with one method: Write([]byte) (int, error).
	ProgressWriter io.Writer
}

// withDefaults returns a copy of Options with zero values replaced by defaults.
//
// Value receiver (o Options) would also work, but pointer receiver lets us
// call it on either Options or *Options. We take a pointer and return a copy
// so the caller's original struct is not mutated unexpectedly.
func (o *Options) withDefaults() Options {
	out := *o // copy the struct

	if out.ConnectTimeout <= 0 {
		out.ConnectTimeout = 30 * time.Second
	}
	if out.ResponseHeaderTimeout <= 0 {
		out.ResponseHeaderTimeout = 60 * time.Second
	}
	if out.StallTimeout <= 0 {
		out.StallTimeout = 30 * time.Second
	}
	if out.MaxRetries < 0 {
		out.MaxRetries = 0
	}
	if out.RetryBackoff <= 0 {
		out.RetryBackoff = time.Second
	}
	if out.ProgressInterval <= 0 {
		out.ProgressInterval = 500 * time.Millisecond
	}
	if out.BufferSize <= 0 {
		// 32 KiB is a common streaming buffer size: small enough for memory,
		// large enough for decent disk/network throughput.
		out.BufferSize = 32 * 1024
	}
	if out.Connections <= 0 {
		// Default stays single-stream so existing callers keep prior behavior.
		out.Connections = 1
	}
	if out.MinSegmentBytes <= 0 {
		// 256 KiB: below this, multi-connection overhead is usually not worth it.
		out.MinSegmentBytes = 256 * 1024
	}
	if out.UserAgent == "" {
		out.UserAgent = "ODM/" + version.Version
	}
	if out.Proxy.Mode == "" {
		out.Proxy.Mode = netx.ProxySystem
	}
	return out
}
