// Package media probes and downloads YouTube media via a managed yt-dlp helper.
package media

import (
	"fmt"
	"net/url"
	"strings"
)

// MaxPlaylistSelect caps how many playlist entries the UI may enqueue at once.
const MaxPlaylistSelect = 100

// IsYouTubeURL reports whether raw is an http(s) YouTube watch/playlist/channel URL.
func IsYouTubeURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	switch host {
	case "youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com", "youtu.be":
		return true
	default:
		return strings.HasSuffix(host, ".youtube.com")
	}
}

// ValidateYouTubeURL returns a normalized URL or an error if not allowed.
func ValidateYouTubeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	if !IsYouTubeURL(raw) {
		return "", fmt.Errorf("only YouTube URLs are supported for media downloads")
	}
	u.Fragment = ""
	return u.String(), nil
}

// LooksLikePlaylist reports playlist/channel-style URLs (vs a single watch video).
func LooksLikePlaylist(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	path := strings.ToLower(u.Path)
	q := u.Query()
	if host == "youtu.be" {
		return false
	}
	if strings.Contains(path, "/playlist") || strings.Contains(path, "/channel/") ||
		strings.Contains(path, "/c/") || strings.Contains(path, "/@") ||
		strings.Contains(path, "/user/") || strings.Contains(path, "/videos") {
		return true
	}
	if q.Get("list") != "" && q.Get("v") == "" {
		return true
	}
	return false
}
