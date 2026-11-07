package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProbeJSONVideo(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
  "_type": "video",
  "id": "abc",
  "title": "Hello",
  "webpage_url": "https://www.youtube.com/watch?v=abc",
  "formats": [
    {"format_id": "18", "ext": "mp4", "height": 360, "vcodec": "avc1", "acodec": "mp4a", "format_note": "360p"},
    {"format_id": "140", "ext": "m4a", "vcodec": "none", "acodec": "mp4a", "format_note": "audio only"}
  ]
}`)
	res, err := ParseProbeJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != "video" || res.Title != "Hello" {
		t.Fatalf("%+v", res)
	}
	if len(res.Formats) < 3 {
		t.Fatalf("expected synthetic + formats, got %d", len(res.Formats))
	}
	if res.Formats[0].ID != "best" {
		t.Fatalf("first should be best: %+v", res.Formats[0])
	}
}

func TestParseProbeJSONPlaylist(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
  "_type": "playlist",
  "id": "PLx",
  "title": "Mix",
  "entries": [
    {"id": "a1", "title": "One", "url": "https://www.youtube.com/watch?v=a1"},
    {"id": "a2", "title": "Two"}
  ]
}`)
	res, err := ParseProbeJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != "playlist" || len(res.Entries) != 2 {
		t.Fatalf("%+v", res)
	}
	if res.Entries[1].URL == "" {
		t.Fatal("expected synthesized watch URL")
	}
}

func TestParseProgressLine(t *testing.T) {
	t.Parallel()
	ev, ok := ParseProgressLine(`[download]  12.3% of 10.00MiB at  1.50MiB/s ETA 00:05`)
	if !ok {
		t.Fatal("expected parse")
	}
	if ev.Total <= 0 || ev.BytesPerSec <= 0 || ev.Downloaded <= 0 {
		t.Fatalf("%+v", ev)
	}
}

func TestParseProgressLineApproxTotal(t *testing.T) {
	t.Parallel()
	// yt-dlp DASH lines often use "of ~  NNiB" with spaces after the tilde.
	ev, ok := ParseProgressLine(`[download]  45.2% of ~  3.00GiB at  1.90MiB/s ETA 00:05`)
	if !ok {
		t.Fatal("expected parse")
	}
	if ev.Total < 3*1024*1024*1024 {
		t.Fatalf("total not parsed from approx size: %+v", ev)
	}
	if ev.Downloaded <= 0 || ev.BytesPerSec <= 0 {
		t.Fatalf("%+v", ev)
	}
}

func TestParseProgressLineZeroPercent(t *testing.T) {
	t.Parallel()
	ev, ok := ParseProgressLine(`[download]   0.0% of  118.62MiB at  250.71KiB/s ETA 08:05`)
	if !ok {
		t.Fatal("expected parse")
	}
	if ev.Total <= 0 || ev.BytesPerSec <= 0 {
		t.Fatalf("%+v", ev)
	}
	if ev.Downloaded != 0 {
		t.Fatalf("want 0 downloaded at 0%%, got %d", ev.Downloaded)
	}
}

func TestEstimateMediaBytesOnDisk(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite := func(name string, n int) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), make([]byte, n), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("clip [abc123].f399.mp4.part", 1000)
	mustWrite("clip [abc123].f140.m4a", 200)
	mustWrite("other [zzzz].mp4", 9999)
	got := estimateMediaBytesOnDisk(dir, "abc123")
	if got != 1200 {
		t.Fatalf("got %d want 1200", got)
	}
}

func TestProgressAccumulatorAcrossStreams(t *testing.T) {
	t.Parallel()
	var acc progressAccumulator

	mid := acc.Accept(ProgressEvent{Downloaded: 350_000_000, Total: 700_000_000, BytesPerSec: 5e6})
	if mid.Downloaded != 350_000_000 || mid.Total != 700_000_000 {
		t.Fatalf("mid video %+v", mid)
	}
	endVideo := acc.Accept(ProgressEvent{Downloaded: 700_000_000, Total: 700_000_000, BytesPerSec: 4e6})
	if endVideo.Downloaded != 700_000_000 {
		t.Fatalf("end video %+v", endVideo)
	}

	// Audio stream starts — raw counters reset (as observed on Big Buck Bunny).
	audio := acc.Accept(ProgressEvent{Downloaded: 4_000_000, Total: 10_000_000, BytesPerSec: 2e6})
	if audio.Downloaded < 700_000_000 {
		t.Fatalf("downloaded must stay monotonic after stream switch: %+v", audio)
	}
	if audio.Downloaded != 704_000_000 {
		t.Fatalf("want 704_000_000 got %+v", audio)
	}
	if audio.Total != 710_000_000 {
		t.Fatalf("want cumulative total 710_000_000 got %+v", audio)
	}

	done := acc.Accept(ProgressEvent{Downloaded: 10_000_000, Total: 10_000_000, BytesPerSec: 1e6})
	if done.Downloaded != 710_000_000 || done.Total != 710_000_000 {
		t.Fatalf("audio complete %+v", done)
	}
}
