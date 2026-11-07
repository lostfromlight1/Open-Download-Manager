package media

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Pinned helper versions downloaded into the app data tools directory.
const (
	// Keep in sync with YouTube extractor fixes; bump when probe fails with
	// "The page needs to be reloaded" on a current yt-dlp release.
	PinnedYtDlpVersion = "2026.10.26"
	PinnedFFmpegTag    = "latest" // yt-dlp/FFmpeg-Builds release tag

	ytDlpVersionFile = "yt-dlp.version"
)

// Tools holds resolved paths to yt-dlp and ffmpeg.
type Tools struct {
	YtDlpPath  string
	FFmpegPath string
	ToolsDir   string
}

// ToolsDir returns <dataDir>/tools.
func ToolsDir(dataDir string) string {
	return filepath.Join(dataDir, "tools")
}

// ResolveTools finds helpers using optional overrides, then tools dir, then PATH.
func ResolveTools(dataDir, ytdlpOverride, ffmpegOverride string) Tools {
	dir := ToolsDir(dataDir)
	t := Tools{ToolsDir: dir}

	t.YtDlpPath = firstExisting(
		strings.TrimSpace(ytdlpOverride),
		filepath.Join(dir, ytdlpBinaryName()),
		lookPath(ytdlpBinaryName()),
		lookPath("yt-dlp"),
	)
	t.FFmpegPath = firstExisting(
		strings.TrimSpace(ffmpegOverride),
		filepath.Join(dir, "ffmpeg"+exeSuffix()),
		lookPath("ffmpeg"),
	)
	return t
}

func (t Tools) YtDlpReady() bool {
	return t.YtDlpPath != "" && fileExecutable(t.YtDlpPath)
}

func (t Tools) FFmpegReady() bool {
	return t.FFmpegPath != "" && fileExecutable(t.FFmpegPath)
}

// ForceRefreshManagedYtDlp removes the managed yt-dlp binary and version pin so
// EnsureTools will re-download PinnedYtDlpVersion (Settings "Check for updates").
func ForceRefreshManagedYtDlp(dataDir string) {
	dir := ToolsDir(dataDir)
	_ = os.Remove(filepath.Join(dir, ytDlpVersionFile))
	_ = os.Remove(filepath.Join(dir, ytdlpBinaryName()))
	_ = os.Remove(filepath.Join(dir, ytdlpBinaryName()+".part"))
}

// NeedsEnsure reports whether managed helpers should be (re)installed:
// missing binaries, or managed yt-dlp not at PinnedYtDlpVersion.
func NeedsEnsure(dataDir, ytdlpOverride, ffmpegOverride string) bool {
	t := ResolveTools(dataDir, ytdlpOverride, ffmpegOverride)
	if !t.YtDlpReady() || !t.FFmpegReady() {
		return true
	}
	managed := filepath.Join(ToolsDir(dataDir), ytdlpBinaryName())
	if !sameFilePath(t.YtDlpPath, managed) {
		// PATH / override copy — do not force-upgrade.
		return false
	}
	return managedYtDlpVersion(ToolsDir(dataDir)) != PinnedYtDlpVersion
}

// EnsureTools downloads missing pinned helpers into dataDir/tools.
// Resumes partial ffmpeg archives when present. httpClient may be nil.
func EnsureTools(ctx context.Context, dataDir string, httpClient *http.Client, onProgress func(string)) (Tools, error) {
	if err := os.MkdirAll(ToolsDir(dataDir), 0o755); err != nil {
		return Tools{}, err
	}
	toolsDir := ToolsDir(dataDir)
	t := ResolveTools(dataDir, "", "")
	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Minute}
	}

	needYt := !t.YtDlpReady() || managedYtDlpVersion(toolsDir) != PinnedYtDlpVersion
	// Only auto-upgrade the copy we manage under tools/; leave PATH overrides alone.
	managedYt := filepath.Join(toolsDir, ytdlpBinaryName())
	if needYt && (t.YtDlpPath == "" || sameFilePath(t.YtDlpPath, managedYt) || !t.YtDlpReady()) {
		if onProgress != nil {
			onProgress("Downloading yt-dlp " + PinnedYtDlpVersion + "…")
		}
		url, err := ytDlpDownloadURL()
		if err != nil {
			return t, err
		}
		// Fresh download (no resume) so a partial old binary cannot linger.
		_ = os.Remove(managedYt)
		_ = os.Remove(managedYt + ".part")
		if err := downloadFile(ctx, client, url, managedYt); err != nil {
			return t, fmt.Errorf("download yt-dlp: %w", err)
		}
		if err := os.Chmod(managedYt, 0o755); err != nil {
			_ = err
		}
		_ = os.WriteFile(filepath.Join(toolsDir, ytDlpVersionFile), []byte(PinnedYtDlpVersion+"\n"), 0o644)
		t.YtDlpPath = managedYt
	}

	if !t.FFmpegReady() {
		if onProgress != nil {
			onProgress("Downloading ffmpeg (large; may take a few minutes)…")
		}
		if err := ensureFFmpeg(ctx, client, ToolsDir(dataDir)); err != nil {
			return t, fmt.Errorf("download ffmpeg: %w", err)
		}
		t = ResolveTools(dataDir, "", "")
		if !t.FFmpegReady() {
			return t, fmt.Errorf("ffmpeg still missing after install")
		}
	}
	return t, nil
}

func ytdlpBinaryName() string {
	if runtime.GOOS == "windows" {
		return "yt-dlp.exe"
	}
	return "yt-dlp"
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func ytDlpDownloadURL() (string, error) {
	base := "https://github.com/yt-dlp/yt-dlp/releases/download/" + PinnedYtDlpVersion + "/"
	switch runtime.GOOS {
	case "windows":
		return base + "yt-dlp.exe", nil
	case "darwin":
		return base + "yt-dlp_macos", nil
	case "linux":
		return base + "yt-dlp_linux", nil
	default:
		return "", fmt.Errorf("unsupported OS %q for managed yt-dlp", runtime.GOOS)
	}
}

func ensureFFmpeg(ctx context.Context, client *http.Client, toolsDir string) error {
	url, name, err := ffmpegDownloadURL()
	if err != nil {
		return err
	}
	archive := filepath.Join(toolsDir, name)
	if err := downloadFile(ctx, client, url, archive); err != nil {
		return err
	}
	lower := strings.ToLower(archive)
	if strings.HasSuffix(lower, ".zip") || strings.HasSuffix(lower, ".tar.xz") ||
		strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		if _, err := os.Stat(archive); err == nil {
			if err := extractFFmpegArchive(archive, toolsDir); err != nil {
				return err
			}
			_ = os.Remove(archive)
		}
	}
	canonical := filepath.Join(toolsDir, "ffmpeg"+exeSuffix())
	if !fileExecutable(canonical) {
		return fmt.Errorf("ffmpeg binary missing after install (looked for %s)", canonical)
	}
	return nil
}

func ffmpegDownloadURL() (url, filename string, err error) {
	// Static GPL builds from yt-dlp/FFmpeg-Builds (single ffmpeg.exe, no DLL deps).
	// Full win64 zip is ~190MiB — callers should run EnsureTools in the background.
	base := "https://github.com/yt-dlp/FFmpeg-Builds/releases/download/" + PinnedFFmpegTag + "/"
	switch runtime.GOOS {
	case "windows":
		return base + "ffmpeg-master-latest-win64-gpl.zip", "ffmpeg-win.zip", nil
	case "linux":
		return base + "ffmpeg-master-latest-linux64-gpl.tar.xz", "ffmpeg-linux.tar.xz", nil
	case "darwin":
		// FFmpeg-Builds does not ship macOS artifacts; require PATH or override.
		return "", "", fmt.Errorf("managed ffmpeg is not available on macOS; install ffmpeg (e.g. brew) or set media.ffmpeg_path")
	default:
		return "", "", fmt.Errorf("unsupported OS %q for managed ffmpeg", runtime.GOOS)
	}
}

func downloadFile(ctx context.Context, client *http.Client, rawURL, dest string) error {
	tmp := dest + ".part"
	var start int64
	if st, err := os.Stat(tmp); err == nil && st.Mode().IsRegular() {
		start = st.Size()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "ODM-media-helper")
	if start > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", start))
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		// Server ignored Range or first attempt — rewrite from scratch.
		start = 0
	case resp.StatusCode == http.StatusPartialContent && start > 0:
		// Resume.
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
	}

	flag := os.O_CREATE | os.O_WRONLY
	if start > 0 && resp.StatusCode == http.StatusPartialContent {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
		start = 0
	}
	f, err := os.OpenFile(tmp, flag, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	_ = os.Remove(dest)
	if err := os.Rename(tmp, dest); err != nil {
		return err
	}
	lower := strings.ToLower(dest)
	if strings.HasSuffix(lower, ".zip") || strings.HasSuffix(lower, ".tar.xz") ||
		strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		dir := filepath.Dir(dest)
		if err := extractFFmpegArchive(dest, dir); err != nil {
			return err
		}
		_ = os.Remove(dest)
	}
	return nil
}

func firstExisting(paths ...string) string {
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if fileExecutable(p) {
			return p
		}
	}
	return ""
}

func lookPath(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

func fileExecutable(path string) bool {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false
	}
	return st.Mode().IsRegular()
}

func managedYtDlpVersion(toolsDir string) string {
	b, err := os.ReadFile(filepath.Join(toolsDir, ytDlpVersionFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func sameFilePath(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return strings.EqualFold(aa, bb)
}

// youtubeExtractorArgSets are tried in order for probe/download.
// Avoids tv_downgraded "page needs to be reloaded" (yt-dlp #17389 / #17405).
func youtubeExtractorArgSets() [][]string {
	// Prefer defaults on current yt-dlp (richer format lists). Fallbacks for
	// intermittent "page needs to be reloaded" from tv_downgraded clients.
	return [][]string{
		{},
		{"--extractor-args", "youtube:player_client=android,web_embedded,-tv_downgraded"},
		{"--extractor-args", "youtube:player_client=web_safari,web_embedded,-tv_downgraded"},
		{"--extractor-args", "youtube:player_client=ios,web_embedded,-tv_downgraded"},
	}
}

func youtubeExtractorArgs() []string {
	return youtubeExtractorArgSets()[0]
}
