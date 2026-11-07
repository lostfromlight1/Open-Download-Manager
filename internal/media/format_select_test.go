package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFormatSelector(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"":       preferredBestSelector(),
		"best":   preferredBestSelector(),
		"ba":     "ba[ext=m4a]/ba",
		"298":    "298+ba[ext=m4a]/298+bestaudio/298",
		"137":    "137+ba[ext=m4a]/137+bestaudio/137",
		"18":     "18+ba[ext=m4a]/18+bestaudio/18",
		"299+ba": "299+ba",
	}
	for in, want := range cases {
		if got := ResolveFormatSelector(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestFindFinalMediaSkipsFragments(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite := func(name string, size int) {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, make([]byte, size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("video [abc123].f312.mp4", 1000)
	mustWrite("video [abc123].f251.webm", 200)
	mustWrite("video [abc123].temp", 900)
	mustWrite("video [abc123].mp4", 800)

	got := findFinalMediaFile(dir)
	if filepath.Base(got) != "video [abc123].mp4" {
		t.Fatalf("got %q", got)
	}

	cleanupIntermediateMedia(dir, "abc123")
	if _, err := os.Stat(filepath.Join(dir, "video [abc123].f312.mp4")); !os.IsNotExist(err) {
		t.Fatal("expected fragment removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "video [abc123].mp4")); err != nil {
		t.Fatal("final mp4 should remain")
	}
}

func TestIsIntermediateMediaName(t *testing.T) {
	t.Parallel()
	yes := []string{
		"a [id].f312.mp4",
		"a [id].f251-9.webm",
		"a.part",
		"a.temp",
	}
	no := []string{
		"a [id].mp4",
		"a [id].mkv",
		"song.m4a",
	}
	for _, n := range yes {
		if !isIntermediateMediaName(n) {
			t.Fatalf("expected intermediate: %s", n)
		}
	}
	for _, n := range no {
		if isIntermediateMediaName(n) {
			t.Fatalf("expected final: %s", n)
		}
	}
}
