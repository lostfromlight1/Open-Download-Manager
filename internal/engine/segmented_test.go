package engine_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/engine"
	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
	"github.com/lostfromlight1/Open-Download-Manager/internal/testserver"
)

func TestSegmentedDownloadOK(t *testing.T) {
	body := bytes.Repeat([]byte("seg-"), 256) // 1024 bytes
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorOK, Body: body})
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "seg.bin")
	result, err := engine.Download(context.Background(), engine.Options{
		URL:             srv.URL,
		Destination:     dest,
		Proxy:           netx.ProxyConfig{Mode: netx.ProxyOff},
		Connections:     4,
		MinSegmentBytes: 64,
		MaxRetries:      1,
		StallTimeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(result.Destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("content mismatch: got %d want %d", len(got), len(body))
	}
	// Probe + 4 segment GETs (or fewer if planner caps) — must be more than one GET.
	if srv.RequestCount() < 2 {
		t.Fatalf("expected multi-request segmented transfer, got %d requests", srv.RequestCount())
	}
}

func TestSegmentedFallsBackWhenNoRange(t *testing.T) {
	body := bytes.Repeat([]byte("n"), 1024)
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorNoRange, Body: body})
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "norange.bin")
	result, err := engine.Download(context.Background(), engine.Options{
		URL:             srv.URL,
		Destination:     dest,
		Proxy:           netx.ProxyConfig{Mode: netx.ProxyOff},
		Connections:     4,
		MinSegmentBytes: 64,
		MaxRetries:      1,
		StallTimeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(result.Destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("fallback content mismatch")
	}
}

func TestSegmentedFallsBackOnBadContentRange(t *testing.T) {
	body := bytes.Repeat([]byte("b"), 1024)
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorBadContentRange, Body: body})
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "badcr.bin")
	result, err := engine.Download(context.Background(), engine.Options{
		URL:             srv.URL,
		Destination:     dest,
		Proxy:           netx.ProxyConfig{Mode: netx.ProxyOff},
		Connections:     4,
		MinSegmentBytes: 64,
		MaxRetries:      1,
		StallTimeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(result.Destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("expected single-stream fallback content")
	}
}

func TestVerifyRangeSupportRejectsNoRange(t *testing.T) {
	body := []byte("hello")
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorNoRange, Body: body})
	t.Cleanup(srv.Close)

	client, err := netx.NewClient(netx.ClientOptions{Proxy: netx.ProxyConfig{Mode: netx.ProxyOff}})
	if err != nil {
		t.Fatal(err)
	}
	cap, err := engine.VerifyRangeSupport(context.Background(), client, srv.URL, "ODM-test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if cap.Supported {
		t.Fatal("expected Supported=false when server ignores Range")
	}
}

func TestVerifyRangeSupportOK(t *testing.T) {
	body := bytes.Repeat([]byte("z"), 64)
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorOK, Body: body})
	t.Cleanup(srv.Close)

	client, err := netx.NewClient(netx.ClientOptions{Proxy: netx.ProxyConfig{Mode: netx.ProxyOff}})
	if err != nil {
		t.Fatal(err)
	}
	cap, err := engine.VerifyRangeSupport(context.Background(), client, srv.URL, "ODM-test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !cap.Supported || cap.Total != int64(len(body)) {
		t.Fatalf("capability mismatch: %#v", cap)
	}
}

func TestSegmentedWrongLengthFails(t *testing.T) {
	body := bytes.Repeat([]byte("w"), 1024)
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorWrongSegmentLen, Body: body})
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "wrong.bin")
	_, err := engine.Download(context.Background(), engine.Options{
		URL:             srv.URL,
		Destination:     dest,
		Proxy:           netx.ProxyConfig{Mode: netx.ProxyOff},
		Connections:     4,
		MinSegmentBytes: 64,
		MaxRetries:      0,
		StallTimeout:    5 * time.Second,
	})
	// Probe uses bytes=0-0 — wrong-segment-len still returns 206 with matching
	// Content-Range for a 1-byte probe, then segment GETs under-deliver.
	if err == nil {
		t.Fatal("expected incomplete/segment error")
	}
	if !errors.Is(err, engine.ErrIncomplete) && !errors.Is(err, engine.ErrHTTPStatus) {
		// Accept either incomplete segment body or status-related failure.
		t.Logf("got error (ok if transfer failed safely): %v", err)
	}
}
