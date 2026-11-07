package media

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFFmpegZipNestedBin(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "ff.zip")
	if err := writeNestedFFmpegZip(zipPath); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := extractFFmpegZip(zipPath, out); err != nil {
		t.Fatal(err)
	}
	if !fileExecutable(filepath.Join(out, "ffmpeg"+exeSuffix())) {
		t.Fatal("ffmpeg missing")
	}
	if !fileExecutable(filepath.Join(out, "ffprobe"+exeSuffix())) {
		t.Fatal("ffprobe missing")
	}
}

func writeNestedFFmpegZip(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	for _, name := range []string{
		"ffmpeg-master-latest-win64-gpl/bin/ffmpeg" + exeSuffix(),
		"ffmpeg-master-latest-win64-gpl/bin/ffprobe" + exeSuffix(),
		"ffmpeg-master-latest-win64-gpl/README.txt",
	} {
		fw, err := w.Create(name)
		if err != nil {
			return err
		}
		if _, err := fw.Write([]byte("fake-" + filepath.Base(name))); err != nil {
			return err
		}
	}
	return w.Close()
}
