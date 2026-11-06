package manager

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/lostfromlight1/Open-Download-Manager/internal/engine"
)

// validateHTTPURL mirrors the engine's http/https rules for enqueue-time checks.
func validateHTTPURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("%w: empty", engine.ErrInvalidURL)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: %v", engine.ErrInvalidURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: unsupported scheme %q", engine.ErrInvalidURL, u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("%w: missing host", engine.ErrInvalidURL)
	}
	return nil
}
