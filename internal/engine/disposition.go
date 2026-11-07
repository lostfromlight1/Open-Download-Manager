package engine

import (
	"mime"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var reOpaqueCDNName = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$|^[0-9a-f]{32,}$`)

// FilenameFromContentDisposition extracts a basename from a Content-Disposition
// header. Returns "" when absent or unsafe.
func FilenameFromContentDisposition(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(header)
	if err != nil {
		// Common servers omit the disposition-type cleanly; try a loose match.
		return filenameFromDispositionLoose(header)
	}
	name := strings.TrimSpace(params["filename"])
	if name == "" {
		name = strings.TrimSpace(params["filename*"])
		if i := strings.Index(name, "''"); i >= 0 {
			name = name[i+2:]
		}
	}
	return sanitizeDispositionBasename(name)
}

func filenameFromDispositionLoose(header string) string {
	lower := strings.ToLower(header)
	for _, key := range []string{"filename*=", "filename="} {
		idx := strings.Index(lower, key)
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(header[idx+len(key):])
		if strings.HasPrefix(rest, `"`) {
			rest = rest[1:]
			if end := strings.Index(rest, `"`); end >= 0 {
				rest = rest[:end]
			}
		} else {
			if end := strings.IndexAny(rest, "; \t"); end >= 0 {
				rest = rest[:end]
			}
		}
		if key == "filename*=" {
			if i := strings.Index(rest, "''"); i >= 0 {
				rest = rest[i+2:]
			}
		}
		if got := sanitizeDispositionBasename(rest); got != "" {
			return got
		}
	}
	return ""
}

func sanitizeDispositionBasename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Trim(name, " .")
	if name == "" || name == "." || name == ".." {
		return ""
	}
	replacer := strings.NewReplacer(
		"<", "_", ">", "_", ":", "_", "\"", "_",
		"|", "_", "?", "_", "*", "_",
	)
	name = replacer.Replace(name)
	if name == "" || LooksLikeOpaqueCDNName(name) {
		return ""
	}
	return name
}

// LooksLikeOpaqueCDNName reports UUID / hex object ids used by CDNs such as
// release-assets.githubusercontent.com instead of a real filename.
func LooksLikeOpaqueCDNName(name string) bool {
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

// PreferReadableFilename returns preferred when it is a usable non-CDN name,
// otherwise fallback (also sanitized for CDN opacity).
func PreferReadableFilename(preferred, fallback string) string {
	preferred = strings.TrimSpace(preferred)
	fallback = strings.TrimSpace(fallback)
	if preferred != "" && !LooksLikeOpaqueCDNName(preferred) {
		return filepath.Base(preferred)
	}
	if fallback != "" && !LooksLikeOpaqueCDNName(fallback) {
		return filepath.Base(fallback)
	}
	if preferred != "" {
		return filepath.Base(preferred)
	}
	return filepath.Base(fallback)
}
