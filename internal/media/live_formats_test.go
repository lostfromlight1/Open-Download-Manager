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

// Live matrix: audio-only + a couple heights through media.Download.
// ODM_LIVE_YT=1 go test ./internal/media -run LiveFormatMatrix -count=1 -timeout 45m -v
func TestLiveFormatMatrix(t *testing.T) {
	if os.Getenv("ODM_LIVE_YT") == "" {
		t.Skip("set ODM_LIVE_YT=1")
	}
	dataDir := filepath.Join(os.Getenv("APPDATA"), "odm")
	tools := media.ResolveTools(dataDir, "", "")
	if !tools.YtDlpReady() || !tools.FFmpegReady() {
		t.Fatal("helpers missing")
	}

	const pageURL = "https://www.youtube.com/watch?v=aqz-KE-bpKQ"
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()

	probe, err := media.Probe(ctx, tools, pageURL)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}

	format360 := ""
	format1080 := ""
	for _, f := range probe.Formats {
		if f.IsAudio || f.ID == "best" || f.ID == "ba" {
			continue
		}
		if format360 == "" && f.Height > 0 && f.Height <= 360 {
			format360 = f.ID
		}
		if format1080 == "" && f.Height >= 1000 && f.Height <= 1100 {
			format1080 = f.ID
		}
	}
	if format360 == "" {
		t.Fatal("no <=360p format in probe")
	}
	if format1080 == "" {
		t.Log("no ~1080p format; will skip 1080 case")
	}
	t.Logf("formats: ba, 360=%s, 1080=%s", format360, format1080)

	cases := []struct {
		name     string
		formatID string
		wantAudio bool
		wantVideo bool
		minBytes int64
	}{
		{name: "audio-ba", formatID: "ba", wantAudio: true, wantVideo: false, minBytes: 100_000},
		{name: "h360", formatID: format360, wantAudio: true, wantVideo: true, minBytes: 1_000_000},
	}
	if format1080 != "" {
		cases = append(cases, struct {
			name     string
			formatID string
			wantAudio bool
			wantVideo bool
			minBytes int64
		}{name: "h1080", formatID: format1080, wantAudio: true, wantVideo: true, minBytes: 5_000_000})
	}

	root := filepath.Join(`D:\Download`, "odm-yt-format-smoke", "via-odm")
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dest := filepath.Join(root, tc.name)
			_ = os.RemoveAll(dest)
			if err := os.MkdirAll(dest, 0o755); err != nil {
				t.Fatal(err)
			}
			final, err := media.Download(ctx, media.DownloadOptions{
				WebpageURL: pageURL,
				FormatID:   tc.formatID,
				DestDir:    dest,
				Tools:      tools,
			})
			if err != nil {
				t.Fatalf("download format=%s: %v", tc.formatID, err)
			}
			info, err := os.Stat(final)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("final=%s size=%d selector=%s", final, info.Size(), media.ResolveFormatSelector(tc.formatID))
			if info.Size() < tc.minBytes {
				t.Fatalf("too small: %d", info.Size())
			}
			base := filepath.Base(final)
			if strings.Contains(base, ".f3") || strings.Contains(base, ".f2") {
				t.Fatalf("looks like fragment: %s", base)
			}
			// leftover fragments?
			entries, _ := os.ReadDir(dest)
			for _, e := range entries {
				n := e.Name()
				if n == base {
					continue
				}
				low := strings.ToLower(n)
				if strings.Contains(low, ".f") && (strings.HasSuffix(low, ".mp4") || strings.HasSuffix(low, ".webm") || strings.HasSuffix(low, ".m4a")) {
					t.Fatalf("leftover fragment: %s", n)
				}
			}
		})
	}
}
