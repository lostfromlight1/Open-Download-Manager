// Package testserver provides a local, deterministic HTTP server for tests.
//
// Why not download from the public internet in tests?
//   - Public servers change, go offline, or rate-limit us.
//   - We need exact failure modes (drop mid-body, stall, 429, ...).
//
// httptest.Server listens on a random localhost port for the duration of a test.
package testserver

import (
	// fmt formats strings (Sprintf) and scans text (Sscanf).
	"fmt"

	// net/http is the standard HTTP server and client API.
	"net/http"

	// httptest builds servers that listen on localhost with a random free port —
	// perfect for unit/integration tests that need a real TCP HTTP server.
	"net/http/httptest"

	// strconv converts between strings and numbers (Itoa = int → string).
	"strconv"

	// strings has helpers like HasPrefix / TrimSpace.
	"strings"

	// sync provides Mutex for protecting shared variables.
	"sync"

	// sync/atomic provides lock-free counters safe across goroutines.
	"sync/atomic"

	// time is used for SlowDelay sleeps.
	"time"
)

// Behavior selects how the fixture server responds.
//
// Using a named string type (instead of plain string) documents intent at
// call sites: testserver.BehaviorOK vs a magic "ok" string.
type Behavior string

// const groups related Behavior values. Exported names start with Uppercase
// so tests in other packages can write testserver.BehaviorOK.
const (
	BehaviorOK              Behavior = "ok"              // normal 200 + full body
	BehaviorRedirect        Behavior = "redirect"        // 302 to /target
	BehaviorNotFound        Behavior = "404"             // permanent 404
	BehaviorServerError     Behavior = "500"             // N times 500, then OK
	BehaviorTooManyRequests Behavior = "429"             // N times 429, then OK
	BehaviorSlowFirstByte   Behavior = "slow-first-byte" // sleep before body
	BehaviorStall           Behavior = "stall"           // headers only, no body
	BehaviorDropMidBody     Behavior = "drop-mid-body"   // write some bytes, then RST
	BehaviorNoRange         Behavior = "no-range"        // always 200, ignores Range
	BehaviorRequireRange    Behavior = "require-range"   // supports resume with 206
)

// Options configures the fixture server.
//
// Zero values are replaced with safe defaults inside Start().
type Options struct {
	Behavior     Behavior      // which failure/success mode to use
	Body         []byte        // payload bytes served to clients
	SlowDelay    time.Duration // used by BehaviorSlowFirstByte
	DropAfter    int           // bytes written before dropping (DropMidBody)
	RedirectTo   string        // optional absolute URL for redirects
	ETag         string        // ETag response header value
	LastModified string        // optional Last-Modified header
	FailTimes    int           // for 429/500: fail this many times then succeed
}

// Server is a local deterministic HTTP fixture.
//
// Fields that start with Uppercase are exported (visible outside the package).
// Lowercase fields (requests, mu, ...) are package-private.
type Server struct {
	HTTP *httptest.Server // underlying server; tests rarely need this directly
	URL  string           // convenience base URL ending with "/"

	// atomic.Int64 is safe to increment from many goroutines without a mutex.
	requests atomic.Int64

	// mu protects failLeft (a regular int is not concurrency-safe by itself).
	mu       sync.Mutex
	failLeft int
	behavior Behavior
	opts     Options
}

// Start launches an httptest server with the given behavior.
//
// Pattern for tests:
//
//	srv := testserver.Start(...)
//	t.Cleanup(srv.Close) // always close, even if the test fails
func Start(opts Options) *Server {
	// Replace zero values so callers can omit fields they do not care about.
	if opts.Behavior == "" {
		opts.Behavior = BehaviorOK
	}
	if opts.Body == nil {
		// nil vs empty slice: both have length 0, but we want a non-empty default.
		opts.Body = []byte("hello-odm-fixture")
	}
	if opts.SlowDelay == 0 {
		opts.SlowDelay = 2 * time.Second
	}
	if opts.DropAfter <= 0 {
		opts.DropAfter = 8
	}
	if opts.ETag == "" {
		// Backticks are not needed; quoted string includes the quote characters
		// that real ETag headers often contain.
		opts.ETag = `"fixture-etag"`
	}

	// &Server{...} allocates a Server on the heap and returns its address.
	// Methods with pointer receivers (*Server) can then mutate this instance.
	s := &Server{
		opts:     opts,
		behavior: opts.Behavior,
		failLeft: opts.FailTimes,
	}

	// ServeMux routes URL paths to handler functions.
	mux := http.NewServeMux()
	// HandleFunc registers a path. "/" matches the root (and, depending on Go
	// version/mux rules, often acts as a catch-all for unmatched paths here).
	mux.HandleFunc("/", s.handle)
	mux.HandleFunc("/target", s.handleOK) // redirect destination

	// NewServer starts listening immediately on 127.0.0.1:<random-port>.
	s.HTTP = httptest.NewServer(mux)
	// Append "/" so callers can treat URL as a directory-like base if needed.
	s.URL = s.HTTP.URL + "/"
	return s
}

// Close shuts down the server and frees the port.
//
// CloseClientConnections aborts idle/active client sockets first so handlers
// blocked on r.Context() (for example BehaviorStall) can finish promptly.
func (s *Server) Close() {
	s.HTTP.CloseClientConnections()
	s.HTTP.Close()
}

// RequestCount returns how many requests were received.
//
// Load() is the atomic read; Add(1) is the atomic increment used in handle().
func (s *Server) RequestCount() int64 {
	return s.requests.Load()
}

// handle is the main route. Method values with pointer receivers can be used
// as http.HandlerFunc because Go automatically adapts func(w, r) methods.
//
// Signature must match: func(http.ResponseWriter, *http.Request)
func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	s.requests.Add(1)

	// switch on the configured behavior. Each case models one network scenario.
	switch s.behavior {
	case BehaviorRedirect:
		target := s.opts.RedirectTo
		if target == "" {
			// Default: bounce to /target on the same server.
			target = s.HTTP.URL + "/target"
		}
		// StatusFound = 302. The Go http.Client follows redirects by default.
		http.Redirect(w, r, target, http.StatusFound)

	case BehaviorNotFound:
		http.Error(w, "missing", http.StatusNotFound)

	case BehaviorServerError:
		if s.consumeFail() {
			http.Error(w, "boom", http.StatusInternalServerError)
			return // return ends this handler early
		}
		s.handleOK(w, r)

	case BehaviorTooManyRequests:
		if s.consumeFail() {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		s.handleOK(w, r)

	case BehaviorSlowFirstByte:
		// Sleep blocks this handler goroutine; the client waits for headers/body.
		time.Sleep(s.opts.SlowDelay)
		s.handleOK(w, r)

	case BehaviorStall:
		// Send headers, then never send the body. Client stall detection should fire.
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(s.opts.Body)))
		w.WriteHeader(http.StatusOK)
		// Flusher pushes buffered headers to the wire immediately.
		// The type assertion `w.(http.Flusher)` asks: "does w implement Flusher?"
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Block until the client disconnects or the server shuts down.
		// Do NOT use bare `select {}` here: that never wakes up, so
		// httptest.Server.Close would hang forever waiting for this handler.
		<-r.Context().Done()

	case BehaviorDropMidBody:
		s.handleDropMidBody(w, r)

	case BehaviorNoRange:
		s.handleNoRange(w, r)

	case BehaviorRequireRange:
		s.handleRange(w, r)

	default:
		s.handleOK(w, r)
	}
}

// consumeFail returns true and decrements the remaining forced failures.
//
// Lock/Unlock (via defer) make this safe if two requests arrive at once.
func (s *Server) consumeFail() bool {
	s.mu.Lock()
	defer s.mu.Unlock() // runs when the function returns, even on early return
	if s.failLeft <= 0 {
		return false
	}
	s.failLeft--
	return true
}

// handleOK serves the full body, or delegates to handleRange when Range is set.
func (s *Server) handleOK(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.Header.Get("Range"), "bytes=") {
		s.handleRange(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("ETag", s.opts.ETag)
	if s.opts.LastModified != "" {
		w.Header().Set("Last-Modified", s.opts.LastModified)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.Itoa(len(s.opts.Body)))
	// WriteHeader must come AFTER setting headers and BEFORE writing the body.
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(s.opts.Body) // ignore byte count and error in this fixture
}

// handleNoRange intentionally ignores Range and always returns HTTP 200 + full body.
// That lets us test "resume not supported" client behavior.
func (s *Server) handleNoRange(w http.ResponseWriter, r *http.Request) {
	_ = r // explicitly unused; keeps the HandlerFunc signature
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("ETag", s.opts.ETag)
	w.Header().Set("Content-Length", strconv.Itoa(len(s.opts.Body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(s.opts.Body)
}

// handleRange implements a minimal bytes=START- responder with HTTP 206.
func (s *Server) handleRange(w http.ResponseWriter, r *http.Request) {
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		s.writeFull(w, http.StatusOK)
		return
	}

	var start int64
	// Sscanf parses a formatted string into variables (like C's sscanf).
	_, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &start)
	if err != nil || start < 0 || int(start) > len(s.opts.Body) {
		http.Error(w, "bad range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// Slice from start to the end: body[start:]
	body := s.opts.Body[start:]
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("ETag", s.opts.ETag)
	w.Header().Set("Accept-Ranges", "bytes")
	// Content-Range format: bytes START-END/TOTAL
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(s.opts.Body)-1, len(s.opts.Body)))
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusPartialContent) // 206
	_, _ = w.Write(body)
}

// writeFull writes the entire Body with the given status code.
func (s *Server) writeFull(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("ETag", s.opts.ETag)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.Itoa(len(s.opts.Body)))
	w.WriteHeader(status)
	_, _ = w.Write(s.opts.Body)
}

// handleDropMidBody writes DropAfter bytes, then forcibly closes the TCP connection.
//
// On a later request with Range, it serves the remainder with 206 so the client
// can prove resume-after-disconnect works.
func (s *Server) handleDropMidBody(w http.ResponseWriter, r *http.Request) {
	body := s.opts.Body
	start := int64(0)
	if strings.HasPrefix(r.Header.Get("Range"), "bytes=") {
		_, _ = fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-", &start)
		if start > 0 && int(start) < len(body) {
			// After a disconnect, honor resume and finish the rest.
			rest := body[start:]
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("ETag", s.opts.ETag)
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(body)-1, len(body)))
			w.Header().Set("Content-Length", strconv.Itoa(len(rest)))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(rest)
			return
		}
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("ETag", s.opts.ETag)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	n := s.opts.DropAfter
	if n > len(body) {
		n = len(body)
	}
	// Write a prefix only: body[:n] means indices 0 .. n-1.
	_, _ = w.Write(body[:n])
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Hijack takes ownership of the TCP connection so we can close it abruptly
	// without sending a clean HTTP end. That simulates a mid-transfer drop.
	hj, ok := w.(http.Hijacker)
	if !ok {
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		return
	}
	_ = conn.Close()
}
