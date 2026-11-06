package engine

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// validateURL parses and checks that we only accept http/https.
//
// Returning (*url.URL, error) is the common Go pair: useful value + error.
func validateURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		// %w wraps ErrInvalidURL so errors.Is(err, ErrInvalidURL) still works.
		return nil, fmt.Errorf("%w: empty", ErrInvalidURL)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	// Only allow schemes we deliberately support in Phase 1.
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%w: unsupported scheme %q", ErrInvalidURL, u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("%w: missing host", ErrInvalidURL)
	}
	return u, nil
}

// resolveDestination turns a user path (or empty) into an absolute file path.
func resolveDestination(dest string, u *url.URL) (string, error) {
	dest = strings.TrimSpace(dest)
	if dest == "" {
		// filepath.Join(".", name) means "name in the current directory".
		name := filenameFromURL(u)
		dest = filepath.Join(".", name)
	}

	// Reject paths that look like directories (trailing / or \).
	// Check the raw input BEFORE Abs/Clean: on Windows, filepath.Abs("C:\\tmp\\")
	// often returns "C:\\tmp" without the trailing separator, which would hide
	// the mistake if we only inspected the cleaned absolute path.
	if strings.HasSuffix(dest, `/`) || strings.HasSuffix(dest, `\`) {
		return "", fmt.Errorf("%w: destination must be a file path", ErrInvalidDestination)
	}

	// Abs converts relative paths to absolute using the process working directory.
	abs, err := filepath.Abs(dest)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidDestination, err)
	}

	// Clean removes redundant . and .. elements.
	clean := filepath.Clean(abs)
	return clean, nil
}

// filenameFromURL picks a safe default filename from the URL path.
func filenameFromURL(u *url.URL) string {
	// path.Base is for slash-separated URL paths (not OS-specific filepath).
	base := path.Base(u.Path)
	base = strings.TrimSpace(base)
	if base == "" || base == "." || base == "/" {
		return "download.bin"
	}

	// Replace characters that are illegal or awkward in Windows filenames.
	replacer := strings.NewReplacer(
		"<", "_", ">", "_", ":", "_", "\"", "_",
		"/", "_", "\\", "_", "|", "_", "?", "_", "*", "_",
	)
	base = replacer.Replace(base)
	if base == "" {
		return "download.bin"
	}
	return base
}

// partPath returns the temporary partial-file path used during download.
//
// Example: "video.mp4" → "video.mp4.part"
func partPath(dest string) string {
	return dest + ".part"
}
