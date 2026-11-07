package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveToolsOverrideAndManaged(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	toolsDir := ToolsDir(dir)
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	managed := filepath.Join(toolsDir, ytdlpBinaryName())
	if err := os.WriteFile(managed, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := ResolveTools(dir, "", "")
	if got.YtDlpPath != managed {
		t.Fatalf("expected managed path %q, got %q", managed, got.YtDlpPath)
	}
	if !got.YtDlpReady() {
		t.Fatal("expected yt-dlp ready")
	}

	override := filepath.Join(dir, "custom"+exeSuffix())
	if err := os.WriteFile(override, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	got2 := ResolveTools(dir, override, "")
	if got2.YtDlpPath != override {
		t.Fatalf("override: got %q want %q", got2.YtDlpPath, override)
	}
}

func TestResolveToolsMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	got := ResolveTools(dir, "", "")
	// PATH may still find yt-dlp; only assert override empty and tools dir empty path preference.
	if got.ToolsDir != ToolsDir(dir) {
		t.Fatalf("tools dir %q", got.ToolsDir)
	}
}
