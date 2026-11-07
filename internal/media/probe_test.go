package media

import "testing"

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
	if ev.Total <= 0 || ev.BytesPerSec <= 0 {
		t.Fatalf("%+v", ev)
	}
}
