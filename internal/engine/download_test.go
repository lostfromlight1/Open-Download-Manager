package engine_test

// This file uses package engine_test (an *external* test package).
// That means we can only call exported APIs (Download, Options, ...),
// which matches how cmd/odm uses the engine.
//
// Contrast with path_test.go, which uses `package engine` so it can call
// unexported helpers like validateURL.

import (
	// bytes has helpers for comparing and repeating byte slices.
	"bytes"

	// context is used for timeouts (stall test) and Background (no cancel).
	"context"

	// errors for errors.Is / errors.As assertions.
	"errors"

	// os for reading the downloaded file back from disk.
	"os"

	// path/filepath builds OS-correct temp destination paths.
	"path/filepath"

	// testing is Go's built-in test framework (*testing.T).
	"testing"

	// time for short timeouts and backoff in tests.
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/engine"
	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
	"github.com/lostfromlight1/Open-Download-Manager/internal/testserver"
)

// TestDownloadSmallFile is a happy-path integration test:
// local fixture → engine.Download → bytes on disk match.
func TestDownloadSmallFile(t *testing.T) {
	// bytes.Repeat returns a new slice with b repeated n times.
	body := bytes.Repeat([]byte("abc123"), 100)

	// Start a local HTTP server. t.Cleanup registers Close to run at test end
	// (even if t.Fatal is called later).
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorOK, Body: body})
	t.Cleanup(srv.Close)

	// t.TempDir() creates a unique temporary directory cleaned up after the test.
	dest := filepath.Join(t.TempDir(), "small.bin")
	result, err := engine.Download(context.Background(), engine.Options{
		URL:          srv.URL,
		Destination:  dest,
		Proxy:        netx.ProxyConfig{Mode: netx.ProxyOff}, // no system proxy in tests
		MaxRetries:   1,
		StallTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err) // fail immediately and stop this test function
	}
	got, err := os.ReadFile(result.Destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("content mismatch: got %d bytes", len(got))
	}
}

// TestDownloadRedirect checks that the HTTP client follows 302 to /target.
func TestDownloadRedirect(t *testing.T) {
	body := []byte("redirected-body")
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorRedirect, Body: body})
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "redir.bin")
	result, err := engine.Download(context.Background(), engine.Options{
		URL:         srv.URL,
		Destination: dest,
		Proxy:       netx.ProxyConfig{Mode: netx.ProxyOff},
		MaxRetries:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(result.Destination)
	if !bytes.Equal(got, body) {
		t.Fatalf("got %q", got)
	}
}

// TestDownloadRetriesServerErrors expects FailTimes=2 → first requests 500, then 200.
func TestDownloadRetriesServerErrors(t *testing.T) {
	body := []byte("after-500")
	srv := testserver.Start(testserver.Options{
		Behavior:  testserver.BehaviorServerError,
		Body:      body,
		FailTimes: 2,
	})
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "retry.bin")
	result, err := engine.Download(context.Background(), engine.Options{
		URL:          srv.URL,
		Destination:  dest,
		Proxy:        netx.ProxyConfig{Mode: netx.ProxyOff},
		MaxRetries:   5,
		RetryBackoff: 10 * time.Millisecond, // keep the test fast
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Attempts < 2 {
		t.Fatalf("expected retries, attempts=%d", result.Attempts)
	}
}

func TestDownloadRetriesTooManyRequests(t *testing.T) {
	body := []byte("after-429")
	srv := testserver.Start(testserver.Options{
		Behavior:  testserver.BehaviorTooManyRequests,
		Body:      body,
		FailTimes: 1,
	})
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "429.bin")
	_, err := engine.Download(context.Background(), engine.Options{
		URL:          srv.URL,
		Destination:  dest,
		Proxy:        netx.ProxyConfig{Mode: netx.ProxyOff},
		MaxRetries:   3,
		RetryBackoff: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestDownloadNotFoundIsNotRetriedForever asserts 404 is permanent.
func TestDownloadNotFoundIsNotRetriedForever(t *testing.T) {
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorNotFound})
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "missing.bin")
	_, err := engine.Download(context.Background(), engine.Options{
		URL:         srv.URL,
		Destination: dest,
		Proxy:       netx.ProxyConfig{Mode: netx.ProxyOff},
		MaxRetries:  3,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	// errors.As fills status if err (or a wrapped cause) is *engine.StatusError.
	var status *engine.StatusError
	if !errors.As(err, &status) || status.StatusCode != 404 {
		t.Fatalf("expected 404 status error, got %v", err)
	}
	if srv.RequestCount() != 1 {
		t.Fatalf("should not retry 404, requests=%d", srv.RequestCount())
	}
}

// TestDownloadRecoversFromMidBodyDrop proves resume after a TCP drop.
func TestDownloadRecoversFromMidBodyDrop(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 64)
	srv := testserver.Start(testserver.Options{
		Behavior:  testserver.BehaviorDropMidBody,
		Body:      body,
		DropAfter: 10, // first response sends only 10 bytes then closes
	})
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "drop.bin")
	result, err := engine.Download(context.Background(), engine.Options{
		URL:          srv.URL,
		Destination:  dest,
		Proxy:        netx.ProxyConfig{Mode: netx.ProxyOff},
		MaxRetries:   5,
		RetryBackoff: 10 * time.Millisecond,
		StallTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(result.Destination)
	if !bytes.Equal(got, body) {
		t.Fatalf("content mismatch got=%d want=%d attempts=%d", len(got), len(body), result.Attempts)
	}
	if result.Attempts < 2 {
		t.Fatalf("expected more than one attempt, got %d", result.Attempts)
	}
}

// TestDownloadStallDetection expects ErrStalled (or cancel) when no body bytes arrive.
func TestDownloadStallDetection(t *testing.T) {
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorStall, Body: []byte("never")})
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "stall.bin")
	// WithTimeout cancels ctx after 3s as a safety net if stall detection fails.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel() // always call cancel to free timer resources

	_, err := engine.Download(ctx, engine.Options{
		URL:          srv.URL,
		Destination:  dest,
		Proxy:        netx.ProxyConfig{Mode: netx.ProxyOff},
		MaxRetries:   0, // do not retry after stall for this test
		StallTimeout: 200 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected stall or cancel error")
	}
	if !errors.Is(err, engine.ErrStalled) && !errors.Is(err, engine.ErrCancelled) {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDownloadResumeUnsupportedWhenServerIgnoresRange seeds a .part file and
// expects ErrResumeUnsupported when the server answers 200 instead of 206.
func TestDownloadResumeUnsupportedWhenServerIgnoresRange(t *testing.T) {
	body := bytes.Repeat([]byte("z"), 40)
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorNoRange, Body: body})
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	dest := filepath.Join(dir, "norange.bin")
	part := dest + ".part"
	// Pre-create a partial file so the engine sends Range: bytes=10-
	if err := os.WriteFile(part, body[:10], 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := engine.Download(context.Background(), engine.Options{
		URL:         srv.URL,
		Destination: dest,
		Proxy:       netx.ProxyConfig{Mode: netx.ProxyOff},
		MaxRetries:  0,
	})
	if !errors.Is(err, engine.ErrResumeUnsupported) {
		t.Fatalf("expected ErrResumeUnsupported, got %v", err)
	}
}

// TestDownloadForceRestart deletes stale .part data and downloads from byte 0.
func TestDownloadForceRestart(t *testing.T) {
	body := []byte("fresh-content")
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorNoRange, Body: body})
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "force.bin")
	if err := os.WriteFile(dest+".part", []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := engine.Download(context.Background(), engine.Options{
		URL:          srv.URL,
		Destination:  dest,
		ForceRestart: true,
		Proxy:        netx.ProxyConfig{Mode: netx.ProxyOff},
		MaxRetries:   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(result.Destination)
	if !bytes.Equal(got, body) {
		t.Fatalf("got %q", got)
	}
}

// TestDownloadPermissionFailure uses an impossible destination:
// a file path whose parent is itself a file (not a directory).
func TestDownloadPermissionFailure(t *testing.T) {
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorOK, Body: []byte("x")})
	t.Cleanup(srv.Close)

	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// On every OS, opening "not-a-dir/out.bin" fails because not-a-dir is a file.
	dest := filepath.Join(blocker, "out.bin")
	_, err := engine.Download(context.Background(), engine.Options{
		URL:         srv.URL,
		Destination: dest,
		Proxy:       netx.ProxyConfig{Mode: netx.ProxyOff},
		MaxRetries:  0,
	})
	if err == nil {
		t.Fatal("expected permission/path error")
	}
}

func TestDownloadRejectsInvalidURL(t *testing.T) {
	_, err := engine.Download(context.Background(), engine.Options{
		URL:         "ftp://example.com/a", // Phase 1 only allows http/https
		Destination: filepath.Join(t.TempDir(), "a.bin"),
	})
	if !errors.Is(err, engine.ErrInvalidURL) {
		t.Fatalf("got %v", err)
	}
}
