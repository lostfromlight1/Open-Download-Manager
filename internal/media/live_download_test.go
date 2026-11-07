package media_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/media"
)

// Live smoke: probe + download a substantial YouTube video (Big Buck Bunny).
// Run: ODM_LIVE_YT=1 go test ./internal/media -run LiveDownload -count=1 -timeout 30m
func TestLiveDownloadLargeYouTube(t *testing.T) {
	if os.Getenv("ODM_LIVE_YT") == "" {
		t.Skip("set ODM_LIVE_YT=1")
	}

	dataDir := filepath.Join(os.Getenv("APPDATA"), "odm")
	tools := media.ResolveTools(dataDir, "", "")
	if !tools.YtDlpReady() {
		t.Fatal("yt-dlp missing — enable YouTube downloads in Settings first")
	}
	if !tools.FFmpegReady() {
		t.Fatal("ffmpeg missing — enable YouTube downloads in Settings first")
	}

	const pageURL = "https://www.youtube.com/watch?v=aqz-KE-bpKQ" // Big Buck Bunny
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	probe, err := media.Probe(ctx, tools, pageURL)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	t.Logf("probe kind=%s title=%q formats=%d", probe.Kind, probe.Title, len(probe.Formats))
	if probe.Kind != "video" {
		t.Fatalf("expected video, got %+v", probe)
	}

	dest := filepath.Join(t.TempDir(), "yt-smoke")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	var last media.ProgressEvent
	var progressHits int
	final, err := media.Download(ctx, media.DownloadOptions{
		WebpageURL: pageURL,
		FormatID:   "best", // prefer merged mp4+m4a
		DestDir:    dest,
		Tools:      tools,
		OnProgress: func(ev media.ProgressEvent) {
			progressHits++
			last = ev
			if progressHits == 1 || progressHits%25 == 0 {
				t.Logf("progress #%d downloaded=%d total=%d bps=%.0f",
					progressHits, ev.Downloaded, ev.Total, ev.BytesPerSec)
			}
		},
	})
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if final == "" {
		t.Fatal("empty final path")
	}
	info, err := os.Stat(final)
	if err != nil {
		t.Fatalf("stat final %s: %v", final, err)
	}
	t.Logf("final=%s size=%d progressHits=%d lastTotal=%d lastBps=%.0f",
		final, info.Size(), progressHits, last.Total, last.BytesPerSec)

	if info.Size() < 5*1024*1024 {
		t.Fatalf("expected a substantial file (>=5MiB), got %d bytes at %s", info.Size(), final)
	}
	base := filepath.Base(final)
	if strings.Contains(strings.ToLower(base), ".f") &&
		(strings.Contains(base, ".f3") || strings.Contains(base, ".f2") || strings.Contains(base, ".f1")) {
		t.Fatalf("final path looks like a DASH fragment, not a merged file: %s", base)
	}
	ext := strings.ToLower(filepath.Ext(final))
	if ext != ".mp4" && ext != ".mkv" && ext != ".webm" && ext != ".m4a" {
		t.Fatalf("unexpected extension %q for %s", ext, final)
	}

	// No leftover fragments for this video id in dest.
	entries, _ := os.ReadDir(dest)
	for _, e := range entries {
		name := e.Name()
		low := strings.ToLower(name)
		if strings.Contains(low, ".f") && (strings.HasSuffix(low, ".mp4") || strings.HasSuffix(low, ".webm") || strings.HasSuffix(low, ".m4a")) {
			if name != base {
				t.Fatalf("leftover fragment still present: %s", name)
			}
		}
		if strings.HasSuffix(low, ".part") || strings.HasSuffix(low, ".ytdl") {
			t.Fatalf("leftover temp file: %s", name)
		}
	}
}
