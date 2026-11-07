package manager_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/manager"
	"github.com/lostfromlight1/Open-Download-Manager/internal/media"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
	"github.com/lostfromlight1/Open-Download-Manager/internal/task"
)

// End-to-end: enqueue via Manager.AddMedia + RunQueue into a real folder.
// Uses 720p-ish concrete format when available to finish faster than 4K best.
// Run: ODM_LIVE_YT=1 go test ./internal/manager -run LiveMediaManager -count=1 -timeout 30m -v
func TestLiveMediaManagerDownload(t *testing.T) {
	if os.Getenv("ODM_LIVE_YT") == "" {
		t.Skip("set ODM_LIVE_YT=1")
	}

	dataDir := filepath.Join(os.Getenv("APPDATA"), "odm")
	tools := media.ResolveTools(dataDir, "", "")
	if !tools.YtDlpReady() || !tools.FFmpegReady() {
		t.Fatal("helpers missing")
	}

	const pageURL = "https://www.youtube.com/watch?v=aqz-KE-bpKQ"
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	probe, err := media.Probe(ctx, tools, pageURL)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	formatID := "best"
	// Prefer an explicit ~720p progressive/DASH id when listed (faster than 4K).
	for _, f := range probe.Formats {
		if f.Height >= 700 && f.Height <= 800 && !f.IsAudio && f.ID != "best" {
			formatID = f.ID
			break
		}
	}
	t.Logf("using formatID=%s title=%q", formatID, probe.Title)

	dest := filepath.Join(`D:\Download`, "odm-yt-smoke")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(t.TempDir(), "live-media.db")
	st, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.SetSetting(ctx, storage.SettingMediaEnabled, "true"); err != nil {
		t.Fatal(err)
	}
	if err := st.SetSetting(ctx, storage.SettingDownloadDir, dest); err != nil {
		t.Fatal(err)
	}

	m := manager.New(st, manager.StaticDecisions{
		Cancel:   manager.FileKeep,
		Remove:   manager.FileKeep,
		Mismatch: manager.MismatchAbort,
	})
	m.SetDataDir(dataDir)

	added, err := m.AddMedia(ctx, manager.AddMediaOptions{
		WebpageURL:  pageURL,
		FormatID:    formatID,
		DisplayName: probe.Title,
		DestDir:     dest,
	})
	if err != nil {
		t.Fatalf("AddMedia: %v", err)
	}
	t.Logf("enqueued task %s", added.ID)

	if err := m.RunQueue(ctx, false); err != nil {
		t.Fatalf("RunQueue: %v", err)
	}

	got, err := m.Get(ctx, added.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != task.StateCompleted {
		t.Fatalf("state=%s err=%s %s", got.State, got.ErrorCode, got.ErrorMessage)
	}
	info, err := os.Stat(got.DestinationPath)
	if err != nil {
		t.Fatalf("missing file %s: %v", got.DestinationPath, err)
	}
	t.Logf("completed path=%s size=%d name=%s", got.DestinationPath, info.Size(), got.DisplayName)
	if info.Size() < 1024*1024 {
		t.Fatalf("file too small: %d", info.Size())
	}
	base := filepath.Base(got.DestinationPath)
	if strings.Contains(base, ".f3") || strings.Contains(base, ".f2") {
		t.Fatalf("destination looks like fragment: %s", base)
	}
	if got.DisplayName != base {
		t.Fatalf("DisplayName=%q path base=%q", got.DisplayName, base)
	}
	workRoot := filepath.Join(dest, ".odm-partial")
	if ents, err := os.ReadDir(workRoot); err == nil && len(ents) > 0 {
		t.Fatalf("leftover work dirs after promote: %v", ents)
	}
}

// Cancel mid-download must remove .odm-partial/<taskID> (no leaked fragments).
// Run: ODM_LIVE_YT=1 go test ./internal/manager -run LiveMediaCancelWorkDir -count=1 -timeout 10m -v
func TestLiveMediaCancelWorkDirCleanup(t *testing.T) {
	if os.Getenv("ODM_LIVE_YT") == "" {
		t.Skip("set ODM_LIVE_YT=1")
	}

	dataDir := filepath.Join(os.Getenv("APPDATA"), "odm")
	tools := media.ResolveTools(dataDir, "", "")
	if !tools.YtDlpReady() {
		t.Fatal("helpers missing")
	}

	const pageURL = "https://www.youtube.com/watch?v=aqz-KE-bpKQ"
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	dest := filepath.Join(t.TempDir(), "yt-cancel-smoke")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(t.TempDir(), "live-cancel.db")
	st, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.SetSetting(ctx, storage.SettingMediaEnabled, "true"); err != nil {
		t.Fatal(err)
	}

	m := manager.New(st, manager.StaticDecisions{
		Cancel:   manager.FileDelete,
		Remove:   manager.FileDelete,
		Mismatch: manager.MismatchAbort,
	})
	m.SetDataDir(dataDir)

	added, err := m.AddMedia(ctx, manager.AddMediaOptions{
		WebpageURL:  pageURL,
		FormatID:    "best",
		DisplayName: "cancel-smoke",
		DestDir:     dest,
	})
	if err != nil {
		t.Fatalf("AddMedia: %v", err)
	}

	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()
	go func() { _ = m.RunQueue(runCtx, true) }()

	workDir := filepath.Join(dest, ".odm-partial", added.ID)
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		got, _ := m.Get(context.Background(), added.ID)
		if got.State == task.StateDownloading || got.State == task.StateInspecting {
			if st, err := os.Stat(workDir); err == nil && st.IsDir() {
				break
			}
		}
		if got.State == task.StateFailed {
			t.Fatalf("failed before cancel: %s %s", got.ErrorCode, got.ErrorMessage)
		}
		time.Sleep(200 * time.Millisecond)
	}
	if _, err := os.Stat(workDir); err != nil {
		t.Fatalf("expected work dir during download: %v", err)
	}

	if err := m.Cancel(context.Background(), added.ID, manager.FileDelete); err != nil {
		t.Fatal(err)
	}
	runCancel()

	cleanupDeadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(cleanupDeadline) {
		if _, err := os.Stat(workDir); os.IsNotExist(err) {
			got, _ := m.Get(context.Background(), added.ID)
			if got.State != task.StateCancelled && got.State != task.StatePaused {
				// Cancel wins; pause path also acceptable if race.
				t.Logf("state after cancel=%s", got.State)
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("work dir still present after cancel: %s", workDir)
}
