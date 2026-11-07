package media

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// ProgressEvent mirrors engine-style progress for media downloads.
type ProgressEvent struct {
	Downloaded  int64
	Total       int64
	BytesPerSec float64
}

// DownloadOptions configures a yt-dlp download into a destination directory.
type DownloadOptions struct {
	WebpageURL string
	FormatID   string // "best", "ba", concrete format_id, or already-merged selector
	DestDir    string
	OutputName string // optional basename without ext; yt-dlp adds extension
	Tools      Tools
	OnProgress func(ProgressEvent)
}

var (
	reProgressPct    = regexp.MustCompile(`\[download\]\s+([0-9.]+)%`)
	reProgressOf     = regexp.MustCompile(`of\s+~?([0-9.]+)([KMG])i?B`)
	reProgressRate   = regexp.MustCompile(`at\s+([0-9.]+)([KMG])i?B/s`)
	reFragmentInName = regexp.MustCompile(`(?i)\.f\d+([.-]|$)`)
	reYTDLPVideoID   = regexp.MustCompile(`\[([A-Za-z0-9_-]{6,})\]`)
)

// ResolveFormatSelector turns a UI format id into a yt-dlp -f expression.
// Prefer MP4 video + M4A audio so ffmpeg can remux to a single playable .mp4
// (Opus/WebM audio + MP4 video often leaves .f312 / .f251 fragments).
func ResolveFormatSelector(formatID string) string {
	id := strings.TrimSpace(formatID)
	if id == "" {
		return preferredBestSelector()
	}
	switch strings.ToLower(id) {
	case "best", "b":
		return preferredBestSelector()
	case "ba", "bestaudio":
		return "ba[ext=m4a]/ba"
	case "bestvideo", "bv":
		return preferredBestSelector()
	}
	if strings.Contains(id, "+") || strings.Contains(id, "/") {
		return id
	}
	// Concrete DASH id: merge m4a audio first, then any bestaudio, else alone.
	return id + "+ba[ext=m4a]/" + id + "+bestaudio/" + id
}

func preferredBestSelector() string {
	return "bv*[ext=mp4]+ba[ext=m4a]/b[ext=mp4]/bv*+ba/b"
}

// Download runs yt-dlp until completion or ctx cancel.
func Download(ctx context.Context, opts DownloadOptions) (finalPath string, err error) {
	if !opts.Tools.YtDlpReady() {
		return "", fmt.Errorf("yt-dlp is not available")
	}
	pageURL, err := ValidateYouTubeURL(opts.WebpageURL)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(opts.DestDir, 0o755); err != nil {
		return "", err
	}

	format := ResolveFormatSelector(opts.FormatID)
	outTmpl := filepath.Join(opts.DestDir, "%(title).80B [%(id)s].%(ext)s")
	if strings.TrimSpace(opts.OutputName) != "" {
		safe := sanitizeBase(opts.OutputName)
		outTmpl = filepath.Join(opts.DestDir, safe+".%(ext)s")
	}

	base := []string{
		"--no-playlist",
		"--newline",
		"-f", format,
		"-o", outTmpl,
		"--no-warnings",
		// Restrict filenames so Windows Explorer can open them (no | 🔥 etc.).
		"--restrict-filenames",
		"--merge-output-format", "mp4",
	}
	if opts.Tools.FFmpegReady() {
		base = append(base, "--ffmpeg-location", filepath.Dir(opts.Tools.FFmpegPath))
	}

	var lastErr error
	for _, clientArgs := range youtubeExtractorArgSets() {
		runArgs := append(append([]string{}, base...), clientArgs...)
		runArgs = append(runArgs, pageURL)
		path, err := runYtDlpDownload(ctx, opts, runArgs)
		if err == nil {
			return path, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return "", fmt.Errorf("cancelled: %w", ctx.Err())
		}
		low := strings.ToLower(err.Error())
		if !strings.Contains(low, "page needs to be reloaded") && !strings.Contains(low, "unplayable") {
			return "", err
		}
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("yt-dlp download failed")
}

func runYtDlpDownload(ctx context.Context, opts DownloadOptions, runArgs []string) (string, error) {
	cmd := exec.CommandContext(ctx, opts.Tools.YtDlpPath, runArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return "", err
	}

	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var errLines []string
	for sc.Scan() {
		line := sc.Text()
		if ev, ok := ParseProgressLine(line); ok {
			if opts.OnProgress != nil {
				opts.OnProgress(ev)
			}
			continue
		}
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		low := strings.ToLower(trim)
		if strings.HasPrefix(low, "error:") || strings.Contains(low, "error") ||
			strings.Contains(low, "ffmpeg") || strings.Contains(low, "unavailable") ||
			strings.Contains(low, "merging") {
			errLines = append(errLines, trim)
			if len(errLines) > 12 {
				errLines = errLines[len(errLines)-12:]
			}
		}
	}
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return "", fmt.Errorf("cancelled: %w", ctx.Err())
	}
	if waitErr != nil {
		detail := strings.Join(errLines, " | ")
		if detail == "" {
			detail = waitErr.Error()
		}
		cleanupIntermediateMedia(opts.DestDir, "")
		return "", fmt.Errorf("yt-dlp download failed: %s", detail)
	}

	finalPath := findFinalMediaFile(opts.DestDir)
	if finalPath == "" {
		cleanupIntermediateMedia(opts.DestDir, "")
		detail := strings.Join(errLines, " | ")
		if detail == "" {
			detail = "video/audio fragments were downloaded but not merged into one file (is ffmpeg ready?)"
		}
		return "", fmt.Errorf("yt-dlp merge incomplete: %s", detail)
	}
	if id := videoIDFromFilename(filepath.Base(finalPath)); id != "" {
		cleanupIntermediateMedia(opts.DestDir, id)
	}
	return finalPath, nil
}

// ParseProgressLine extracts progress from a yt-dlp --newline download line.
func ParseProgressLine(line string) (ProgressEvent, bool) {
	m := reProgressPct.FindStringSubmatch(line)
	if m == nil {
		return ProgressEvent{}, false
	}
	pct, _ := strconv.ParseFloat(m[1], 64)
	var total int64
	if tm := reProgressOf.FindStringSubmatch(line); tm != nil {
		total = parseSize(tm[1], tm[2])
	}
	var rate float64
	if rm := reProgressRate.FindStringSubmatch(line); rm != nil {
		rate = float64(parseSize(rm[1], rm[2]))
	}
	var downloaded int64
	if total > 0 && pct > 0 {
		downloaded = int64(float64(total) * pct / 100.0)
	}
	return ProgressEvent{Downloaded: downloaded, Total: total, BytesPerSec: rate}, true
}

func parseSize(num, unit string) int64 {
	v, _ := strconv.ParseFloat(num, 64)
	mult := 1.0
	switch strings.ToUpper(unit) {
	case "K":
		mult = 1024
	case "M":
		mult = 1024 * 1024
	case "G":
		mult = 1024 * 1024 * 1024
	}
	return int64(v * mult)
}

func sanitizeBase(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	for _, r := range name {
		switch {
		case r < 32, r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' ||
			r == '"' || r == '<' || r == '>' || r == '|' || r == '#' || r == '%':
			b.WriteByte('_')
		case unicode.IsControl(r):
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	name = strings.TrimSpace(b.String())
	if name == "" {
		return "video"
	}
	return name
}

func isIntermediateMediaName(name string) bool {
	low := strings.ToLower(name)
	switch {
	case strings.HasSuffix(low, ".part"),
		strings.HasSuffix(low, ".ytdl"),
		strings.HasSuffix(low, ".temp"),
		strings.HasSuffix(low, ".tmp"):
		return true
	}
	// yt-dlp DASH fragments: …[id].f312.mp4 or …[id].f251-9.webm
	return reFragmentInName.MatchString(name)
}

func isPlayableMediaExt(name string) bool {
	low := strings.ToLower(name)
	for _, ext := range []string{".mp4", ".mkv", ".webm", ".m4a", ".mp3", ".opus", ".ogg"} {
		if strings.HasSuffix(low, ext) {
			return true
		}
	}
	return false
}

func findFinalMediaFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var best string
	var bestMod int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if isIntermediateMediaName(name) || !isPlayableMediaExt(name) {
			continue
		}
		info, err := e.Info()
		if err != nil || info.Size() == 0 {
			continue
		}
		if info.ModTime().UnixNano() >= bestMod {
			bestMod = info.ModTime().UnixNano()
			best = filepath.Join(dir, name)
		}
	}
	return best
}

func videoIDFromFilename(name string) string {
	m := reYTDLPVideoID.FindStringSubmatch(name)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// cleanupIntermediateMedia removes yt-dlp fragment/temp files for a video id
// (or all intermediates in dir when id is empty — used after hard failure).
func cleanupIntermediateMedia(dir, videoID string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isIntermediateMediaName(name) {
			continue
		}
		if videoID != "" && !strings.Contains(name, videoID) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, name))
	}
}
