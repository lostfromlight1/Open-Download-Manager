// Package netx (short for "network extras") configures HTTP clients for ODM.
//
// Keeping proxy/timeout setup here means the download engine does not hard-code
// how connections are dialed. Later we can plug in optional tunneling without
// rewriting download logic.
package netx

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ProxyMode is a custom string type.
//
// Defining a type (instead of plain string) makes intent clearer in APIs and
// lets the compiler catch some mistakes when you pass the wrong kind of string.
type ProxyMode string

// const blocks group related constants.
//
// Using ProxyMode("system") style values keeps assignments type-safe.
const (
	// ProxySystem uses environment / OS proxy settings via http.ProxyFromEnvironment.
	ProxySystem ProxyMode = "system"

	// ProxyOff disables proxying (direct connections).
	ProxyOff ProxyMode = "off"

	// ProxyManual uses an explicit HTTP or HTTPS proxy URL.
	ProxyManual ProxyMode = "manual"
)

// ProxyConfig configures outbound proxy behavior.
//
// A struct groups related fields. Other packages set these fields by name.
type ProxyConfig struct {
	Mode ProxyMode
	URL  string // required when Mode is ProxyManual (http:// or https://)
}

// ParseProxyMode accepts:
//   - "system" / "" → use OS/env proxy
//   - "off" / "none" / "direct" → no proxy
//   - a full proxy URL like "http://127.0.0.1:8080" → manual mode
func ParseProxyMode(value string) (ProxyMode, string, error) {
	// TrimSpace removes leading/trailing whitespace; ToLower normalizes case.
	v := strings.TrimSpace(strings.ToLower(value))

	switch v {
	case "", string(ProxySystem):
		// string(ProxySystem) converts the typed constant back to a plain string
		// for comparison with v.
		return ProxySystem, "", nil

	case string(ProxyOff), "none", "direct":
		return ProxyOff, "", nil

	case string(ProxyManual):
		// Bare "manual" is incomplete because we still need the proxy address.
		return ProxyManual, "", fmt.Errorf("proxy mode manual requires --proxy-url")

	default:
		// If the user passed a URL, treat it as manual proxy.
		if strings.Contains(v, "://") {
			// Keep original casing/spacing for the URL (except we used lower for detection).
			// Re-trim the original value for the returned URL.
			return ProxyManual, strings.TrimSpace(value), nil
		}
		return "", "", fmt.Errorf("unknown proxy mode %q (use system, off, or a proxy URL)", value)
	}
}

// proxyFunc returns the function http.Transport.Proxy expects.
//
// Signature of that function: func(*http.Request) (*url.URL, error)
// Returning nil means "no proxy".
func proxyFunc(cfg ProxyConfig) (func(*http.Request) (*url.URL, error), error) {
	switch cfg.Mode {
	case ProxySystem, "":
		// Built-in helper reads HTTP_PROXY / HTTPS_PROXY / NO_PROXY, etc.
		return http.ProxyFromEnvironment, nil

	case ProxyOff:
		return nil, nil

	case ProxyManual:
		if strings.TrimSpace(cfg.URL) == "" {
			return nil, fmt.Errorf("manual proxy requires a URL")
		}

		// url.Parse splits a URL into scheme/host/path/query.
		u, err := url.Parse(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("parse proxy URL: %w", err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("proxy URL scheme must be http or https, got %q", u.Scheme)
		}
		if u.Host == "" {
			return nil, fmt.Errorf("proxy URL must include a host")
		}

		// http.ProxyURL returns a closure that always returns this proxy URL.
		return http.ProxyURL(u), nil

	default:
		return nil, fmt.Errorf("unknown proxy mode %q", cfg.Mode)
	}
}
