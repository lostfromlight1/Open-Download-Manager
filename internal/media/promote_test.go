package media

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromoteMediaFileMovesIntoDest(t *testing.T) {
	t.Parallel()
	work := t.TempDir()
	dest := t.TempDir()
	src := filepath.Join(work, "clip [abc].mp4")
	if err := os.WriteFile(src, []byte("merged"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := PromoteMediaFile(src, dest)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dest, "clip [abc].mp4")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	data, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "merged" {
		t.Fatalf("content %q", data)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("work file should be gone: %v", err)
	}
}

func TestPromoteMediaFileUniqueOnConflict(t *testing.T) {
	t.Parallel()
	work := t.TempDir()
	dest := t.TempDir()
	existing := filepath.Join(dest, "clip.mp4")
	if err := os.WriteFile(existing, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(work, "clip.mp4")
	if err := os.WriteFile(src, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := PromoteMediaFile(src, dest)
	if err != nil {
		t.Fatal(err)
	}
	if got == existing {
		t.Fatal("expected unique destination name")
	}
	if filepath.Dir(got) != dest {
		t.Fatalf("dest dir %q", got)
	}
	data, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("content %q", data)
	}
}

func TestFormatUserMediaErrorMissingFragment(t *testing.T) {
	t.Parallel()
	raw := `ERROR: Unable to download video: [Errno 2] No such file or directory: 'D:\\Download\\clip [Hlr8vgRllxc].f616.mp4.part-Frag484'`
	got := FormatUserMediaError(raw)
	if !strings.Contains(strings.ToLower(got), "fragment") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(strings.ToLower(got), "duplicate") {
		t.Fatalf("want duplicate hint, got %q", got)
	}
}

func TestEstimateMediaBytesOnDiskPartFrag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	name := "clip [abc123].f616.mp4.part-Frag484"
	if err := os.WriteFile(filepath.Join(dir, name), make([]byte, 500), 0o644); err != nil {
		t.Fatal(err)
	}
	got := estimateMediaBytesOnDisk(dir, "abc123")
	if got != 500 {
		t.Fatalf("got %d want 500", got)
	}
}
