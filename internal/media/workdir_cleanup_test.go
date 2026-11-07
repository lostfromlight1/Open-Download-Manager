package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPromoteThenRemoveWorkDirLeavesUserFile(t *testing.T) {
	t.Parallel()
	userDest := t.TempDir()
	workDir := filepath.Join(userDest, ".odm-partial", "task-abc")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(workDir, "clip [abc].mp4")
	if err := os.WriteFile(src, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	promoted, err := PromoteMediaFile(src, userDest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(workDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Fatalf("work dir should be gone: %v", err)
	}
	data, err := os.ReadFile(promoted)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ok" {
		t.Fatalf("content %q", data)
	}
}

func TestForceRefreshManagedYtDlpTriggersNeedsEnsure(t *testing.T) {
	t.Parallel()
	data := t.TempDir()
	toolsDir := ToolsDir(data)
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	yt := filepath.Join(toolsDir, ytdlpBinaryName())
	ff := filepath.Join(toolsDir, "ffmpeg"+exeSuffix())
	if err := os.WriteFile(yt, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ff, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolsDir, ytDlpVersionFile), []byte(PinnedYtDlpVersion+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if NeedsEnsure(data, "", "") {
		t.Fatal("should not need ensure when pin matches")
	}
	ForceRefreshManagedYtDlp(data)
	if !NeedsEnsure(data, "", "") {
		t.Fatal("expected NeedsEnsure after ForceRefreshManagedYtDlp")
	}
	if _, err := os.Stat(yt); !os.IsNotExist(err) {
		t.Fatalf("yt-dlp should be removed: %v", err)
	}
}
