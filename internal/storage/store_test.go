package storage_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
	"github.com/lostfromlight1/Open-Download-Manager/internal/task"
)

func TestStoreRoundTripTaskAndSettings(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "odm.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	now := time.Now().UTC()
	total := int64(100)
	in := task.Task{
		ID:              "abc",
		URL:             "https://example.com/a.bin",
		DisplayName:     "a.bin",
		DestinationPath: filepath.Join(dir, "a.bin"),
		State:           task.StateQueued,
		TotalBytes:      &total,
		DownloadedBytes: 10,
		ETag:            `"etag-1"`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := store.InsertTask(ctx, in); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetTask(ctx, "abc")
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != in.URL || got.DownloadedBytes != 10 || got.TotalBytes == nil || *got.TotalBytes != 100 {
		t.Fatalf("round-trip mismatch: %#v", got)
	}

	cfg := storage.DefaultSettings()
	cfg.ProxyMode = netx.ProxyOff
	cfg.MaxActiveDownloads = 3
	cfg.BrowserIntegrationEnabled = true
	cfg.ChromeExtensionID = "abcdefghijklmnopqrstuvwxyzabcdef"
	cfg.CloseToTray = false
	if err := store.SaveSettings(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ProxyMode != netx.ProxyOff || loaded.MaxActiveDownloads != 3 {
		t.Fatalf("settings mismatch: %#v", loaded)
	}
	if !loaded.BrowserIntegrationEnabled || loaded.ChromeExtensionID != cfg.ChromeExtensionID {
		t.Fatalf("browser settings mismatch: %#v", loaded)
	}
	if loaded.CloseToTray {
		t.Fatalf("close_to_tray want false: %#v", loaded)
	}

	segs := []storage.Segment{
		{ID: "s1", TaskID: "abc", StartByte: 0, EndByte: 49, State: storage.SegmentPending},
		{ID: "s2", TaskID: "abc", StartByte: 50, EndByte: 99, State: storage.SegmentPending},
	}
	if err := store.ReplaceSegments(ctx, "abc", segs); err != nil {
		t.Fatal(err)
	}
	gotSegs, err := store.ListSegments(ctx, "abc")
	if err != nil {
		t.Fatal(err)
	}
	if len(gotSegs) != 2 || gotSegs[0].EndByte != 49 || gotSegs[1].StartByte != 50 {
		t.Fatalf("segments mismatch: %#v", gotSegs)
	}
	gotSegs[0].DownloadedBytes = 49
	gotSegs[0].State = storage.SegmentCompleted
	if err := store.UpdateSegmentProgress(ctx, gotSegs[0]); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteSegmentsForTask(ctx, "abc"); err != nil {
		t.Fatal(err)
	}
	gotSegs, err = store.ListSegments(ctx, "abc")
	if err != nil {
		t.Fatal(err)
	}
	if len(gotSegs) != 0 {
		t.Fatalf("expected no segments after delete, got %#v", gotSegs)
	}
}
