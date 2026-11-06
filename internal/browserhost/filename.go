package browserhost

import (
	"net/url"
	"strings"
	"unicode"
)

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
	if base == "" {
		return "download.bin"
	}
	return base
}
