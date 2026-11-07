package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNeedsEnsureWhenVersionMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	toolsDir := ToolsDir(dir)
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pretend managed yt-dlp exists but pin file is missing → needs ensure.
	managed := filepath.Join(toolsDir, ytdlpBinaryName())
	if err := os.WriteFile(managed, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	ffmpeg := filepath.Join(toolsDir, "ffmpeg"+exeSuffix())
	if err := os.WriteFile(ffmpeg, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !NeedsEnsure(dir, "", "") {
		t.Fatal("expected NeedsEnsure when yt-dlp.version missing")
	}
}
