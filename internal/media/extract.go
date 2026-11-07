package media

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// extractFFmpegArchive unpacks an ffmpeg build archive and places ffmpeg(.exe) in destDir.
func extractFFmpegArchive(archivePath, destDir string) error {
	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractFFmpegZip(archivePath, destDir)
	case strings.HasSuffix(lower, ".tar.xz"):
		return extractFFmpegTarXZ(archivePath, destDir)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractFFmpegTarGZ(archivePath, destDir)
	default:
		return fmt.Errorf("unsupported archive type: %s", archivePath)
	}
}

func extractFFmpegZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	want := "ffmpeg" + exeSuffix()
	var copiedFFmpeg, copiedProbe bool
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		base := filepath.Base(f.Name)
		var outName string
		switch {
		case strings.EqualFold(base, want) || strings.EqualFold(base, "ffmpeg.exe") || base == "ffmpeg":
			outName = "ffmpeg" + exeSuffix()
		case strings.EqualFold(base, "ffprobe"+exeSuffix()) || strings.EqualFold(base, "ffprobe.exe") || base == "ffprobe":
			outName = "ffprobe" + exeSuffix()
		default:
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		outPath := filepath.Join(destDir, outName)
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			_ = rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		_ = out.Close()
		_ = rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if strings.HasPrefix(outName, "ffmpeg") {
			copiedFFmpeg = true
		} else {
			copiedProbe = true
		}
		if copiedFFmpeg && copiedProbe {
			return nil
		}
	}
	if !copiedFFmpeg {
		return fmt.Errorf("ffmpeg binary not found inside zip")
	}
	return nil
}

func extractFFmpegTarGZ(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	return extractTarSeekingFFmpeg(tar.NewReader(gz), destDir)
}

func extractFFmpegTarXZ(archivePath, destDir string) error {
	// stdlib has no xz; use system tar when available.
	tmp := filepath.Join(destDir, "ffmpeg-extract")
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	cmd := exec.Command("tar", "-xJf", archivePath, "-C", tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tar extract ffmpeg: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	var found string
	_ = filepath.Walk(tmp, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		base := filepath.Base(path)
		if base == "ffmpeg" || base == "ffmpeg.exe" {
			found = path
			return io.EOF
		}
		return nil
	})
	if found == "" {
		return fmt.Errorf("ffmpeg binary not found inside tar.xz")
	}
	dest := filepath.Join(destDir, "ffmpeg"+exeSuffix())
	in, err := os.Open(found)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func extractTarSeekingFFmpeg(tr *tar.Reader, destDir string) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		base := filepath.Base(hdr.Name)
		if base != "ffmpeg" && base != "ffmpeg.exe" {
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		outPath := filepath.Join(destDir, "ffmpeg"+exeSuffix())
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, tr)
		_ = out.Close()
		return copyErr
	}
	return fmt.Errorf("ffmpeg binary not found inside tar")
}
