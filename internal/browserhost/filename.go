package browserhost

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var reOpaqueCDNName = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$|^[0-9a-f]{32,}$`)

// SanitizeFilename strips path components and unsafe characters from a
// browser-supplied name. Empty input returns "".
func SanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	// Avoid filepath.Base: on Windows "a:b.bin" is treated as a drive-relative path.
	name = strings.ReplaceAll(name, "\\", "/")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		switch {
		case r == 0 || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|':
			b.WriteByte('_')
		case unicode.IsControl(r):
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), " .")
	if out == "" || out == "." || out == ".." {
		return ""
	}
	return out
}

// FilenameFromURL derives a fallback name from a URL path.
func FilenameFromURL(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	base := ""
	if err == nil {
		base = u.Path
	} else {
		base = rawURL
	}
	base = strings.TrimRight(base, "/")
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	base = SanitizeFilename(base)
	if base == "" || looksLikeOpaqueCDNName(base) {
		return "download.bin"
	}
	return base
}

// PreferFilename picks a readable name over GitHub CDN UUID object ids.
func PreferFilename(preferred, fallback string) string {
	preferred = strings.TrimSpace(preferred)
	fallback = strings.TrimSpace(fallback)
	if preferred != "" && !looksLikeOpaqueCDNName(preferred) {
		return preferred
	}
	if fallback != "" && !looksLikeOpaqueCDNName(fallback) {
		return fallback
	}
	if preferred != "" {
		return preferred
	}
	return fallback
}

func looksLikeOpaqueCDNName(name string) bool {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "" {
		return false
	}
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	if stem == "" {
		stem = base
	}
	return reOpaqueCDNName.MatchString(stem)
}
