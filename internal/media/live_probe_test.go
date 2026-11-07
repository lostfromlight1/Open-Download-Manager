package media_test

import (
	"context"
	"github.com/lostfromlight1/Open-Download-Manager/internal/media"
	"os"
	"path/filepath"
	"testing"
)

func TestLiveProbeYouTube(t *testing.T) {
	if os.Getenv("ODM_LIVE_YT") == "" {
		t.Skip("set ODM_LIVE_YT=1")
	}
	dir := filepath.Join(os.Getenv("APPDATA"), "odm")
	tools := media.ResolveTools(dir, "", "")
	if !tools.YtDlpReady() {
		t.Fatal("yt-dlp missing")
	}
	res, err := media.Probe(context.Background(), tools, "https://www.youtube.com/watch?v=ujs0_cpAnaw")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("kind=%s title=%q formats=%d", res.Kind, res.Title, len(res.Formats))
	if res.Kind != "video" || len(res.Formats) < 2 {
		t.Fatalf("%+v", res)
	}
}
