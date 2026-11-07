package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// Format describes one selectable yt-dlp format.
type Format struct {
	ID         string  `json:"id"`
	Ext        string  `json:"ext"`
	Resolution string  `json:"resolution"`
	Note       string  `json:"note"`
	VCodec     string  `json:"vcodec"`
	ACodec     string  `json:"acodec"`
	FPS        float64 `json:"fps"`
	Filesize   int64   `json:"filesize"`
	Height     int     `json:"height"`
	IsAudio    bool    `json:"isAudio"`
	VideoOnly  bool    `json:"videoOnly"` // adaptive video; download merges bestaudio
}

// PlaylistEntry is one video in a flat playlist/channel listing.
type PlaylistEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// ProbeResult is either a single video (with formats) or a playlist listing.
type ProbeResult struct {
	Kind    string          `json:"kind"` // "video" | "playlist"
	ID      string          `json:"id"`
	Title   string          `json:"title"`
	Webpage string          `json:"webpageUrl"`
	Formats []Format        `json:"formats,omitempty"`
	Entries []PlaylistEntry `json:"entries,omitempty"`
}

// Probe runs yt-dlp -J (or --flat-playlist -J) and parses the result.
func Probe(ctx context.Context, tools Tools, rawURL string) (ProbeResult, error) {
	if !tools.YtDlpReady() {
		return ProbeResult{}, fmt.Errorf("yt-dlp is not available; enable media downloads in Settings")
	}
	pageURL, err := ValidateYouTubeURL(rawURL)
	if err != nil {
		return ProbeResult{}, err
	}

	base := []string{"-J", "--no-playlist"}
	if LooksLikePlaylist(pageURL) {
		base = []string{"-J", "--flat-playlist"}
	}

	var lastErr error
	for _, clientArgs := range youtubeExtractorArgSets() {
		args := append(append([]string{}, base...), clientArgs...)
		args = append(args, "--no-warnings", pageURL)

		cmd := exec.CommandContext(ctx, tools.YtDlpPath, args...)
		hideHelperConsole(cmd)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			lastErr = fmt.Errorf("%s", msg)
			if ctx.Err() != nil {
				return ProbeResult{}, fmt.Errorf("yt-dlp probe cancelled: %w", ctx.Err())
			}
			continue
		}
		return ParseProbeJSON(stdout.Bytes())
	}

	msg := "unknown error"
	if lastErr != nil {
		msg = lastErr.Error()
	}
	if strings.Contains(strings.ToLower(msg), "page needs to be reloaded") {
		return ProbeResult{}, fmt.Errorf("yt-dlp probe failed (YouTube blocked this helper build): %s — open Settings → Retry helper install to upgrade yt-dlp", msg)
	}
	return ProbeResult{}, fmt.Errorf("yt-dlp probe failed: %s", msg)
}

// ParseProbeJSON converts yt-dlp -J output into ProbeResult (also used in tests).
func ParseProbeJSON(raw []byte) (ProbeResult, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return ProbeResult{}, fmt.Errorf("invalid yt-dlp json: %w", err)
	}

	var typ string
	_ = json.Unmarshal(root["_type"], &typ)
	if typ == "" {
		typ = "video"
	}

	if typ == "playlist" {
		return parsePlaylist(root)
	}
	return parseVideo(root)
}

func parseVideo(root map[string]json.RawMessage) (ProbeResult, error) {
	var out ProbeResult
	out.Kind = "video"
	_ = json.Unmarshal(root["id"], &out.ID)
	_ = json.Unmarshal(root["title"], &out.Title)
	_ = json.Unmarshal(root["webpage_url"], &out.Webpage)
	if out.Webpage == "" {
		_ = json.Unmarshal(root["original_url"], &out.Webpage)
	}

	var formats []map[string]any
	if raw, ok := root["formats"]; ok {
		_ = json.Unmarshal(raw, &formats)
	}
	out.Formats = summarizeFormats(formats)
	// Synthetic choices always available.
	out.Formats = append([]Format{
		{ID: "best", Note: "Best available (video+audio)", Ext: "mp4"},
		{ID: "ba", Note: "Audio only (best)", Ext: "m4a", IsAudio: true},
	}, out.Formats...)
	return out, nil
}

func parsePlaylist(root map[string]json.RawMessage) (ProbeResult, error) {
	var out ProbeResult
	out.Kind = "playlist"
	_ = json.Unmarshal(root["id"], &out.ID)
	_ = json.Unmarshal(root["title"], &out.Title)
	_ = json.Unmarshal(root["webpage_url"], &out.Webpage)

	var entries []map[string]any
	if raw, ok := root["entries"]; ok {
		_ = json.Unmarshal(raw, &entries)
	}
	for _, e := range entries {
		if e == nil {
			continue
		}
		id, _ := e["id"].(string)
		title, _ := e["title"].(string)
		u, _ := e["url"].(string)
		if u == "" {
			u, _ = e["webpage_url"].(string)
		}
		if u == "" && id != "" {
			u = "https://www.youtube.com/watch?v=" + id
		}
		if id == "" && u == "" {
			continue
		}
		if title == "" {
			title = id
		}
		out.Entries = append(out.Entries, PlaylistEntry{ID: id, Title: title, URL: u})
	}
	return out, nil
}

func summarizeFormats(raw []map[string]any) []Format {
	seen := map[string]bool{}
	var out []Format
	for _, f := range raw {
		id := stringFromAny(f["format_id"])
		if id == "" || seen[id] {
			continue
		}
		vcodec, _ := f["vcodec"].(string)
		acodec, _ := f["acodec"].(string)
		ext, _ := f["ext"].(string)
		note, _ := f["format_note"].(string)
		res, _ := f["resolution"].(string)
		height := intFromAny(f["height"])
		fps := floatFromAny(f["fps"])
		fs := int64FromAny(f["filesize"])
		if fs == 0 {
			fs = int64FromAny(f["filesize_approx"])
		}
		// Skip storyboards / images.
		if strings.EqualFold(ext, "mhtml") || strings.HasPrefix(strings.ToLower(id), "sb") {
			continue
		}
		isAudio := (vcodec == "none" || vcodec == "") && acodec != "" && acodec != "none"
		videoOnly := !isAudio && (acodec == "none" || acodec == "") && vcodec != "" && vcodec != "none"
		// Prefer progressive or common heights for the picker (avoid tiny duplicates).
		if !isAudio && height > 0 && height < 360 {
			continue
		}
		// Dubbed auto tracks clutter the picker; keep primary audio via "ba".
		if isAudio && strings.Contains(strings.ToLower(note), "dubbed") {
			continue
		}
		seen[id] = true
		label := note
		if res != "" && label == "" {
			label = res
		}
		if videoOnly {
			if label != "" {
				label += " · video+audio"
			} else {
				label = "video+audio"
			}
		}
		out = append(out, Format{
			ID: id, Ext: ext, Resolution: res, Note: label,
			VCodec: vcodec, ACodec: acodec, FPS: fps, Filesize: fs,
			Height: height, IsAudio: isAudio, VideoOnly: videoOnly,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IsAudio != out[j].IsAudio {
			return !out[i].IsAudio && out[j].IsAudio
		}
		return out[i].Height > out[j].Height
	})
	// Cap list for UI.
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

func stringFromAny(v any) string {
	switch n := v.(type) {
	case string:
		return n
	case float64:
		// JSON numbers for format_id (e.g. 298).
		if n == float64(int64(n)) {
			return strconv.FormatInt(int64(n), 10)
		}
		return strconv.FormatFloat(n, 'f', -1, 64)
	case int:
		return strconv.Itoa(n)
	case int64:
		return strconv.FormatInt(n, 10)
	case json.Number:
		return n.String()
	default:
		return ""
	}
}

func floatFromAny(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		return 0
	}
}

func int64FromAny(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}
