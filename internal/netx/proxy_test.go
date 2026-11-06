package netx

import "testing"

// TestParseProxyMode checks the user-facing proxy string parser.
func TestParseProxyMode(t *testing.T) {
	t.Parallel()

	// Multiple return values: mode, url, err.
	mode, url, err := ParseProxyMode("system")
	if err != nil || mode != ProxySystem || url != "" {
		t.Fatalf("system: mode=%q url=%q err=%v", mode, url, err)
	}

	mode, url, err = ParseProxyMode("off")
	if err != nil || mode != ProxyOff || url != "" {
		t.Fatalf("off: mode=%q url=%q err=%v", mode, url, err)
	}

	mode, url, err = ParseProxyMode("http://127.0.0.1:8080")
	if err != nil || mode != ProxyManual || url != "http://127.0.0.1:8080" {
		t.Fatalf("url: mode=%q url=%q err=%v", mode, url, err)
	}

	// Bare "manual" is incomplete — we still need the proxy address.
	if _, _, err := ParseProxyMode("manual"); err == nil {
		t.Fatal("expected error for bare manual")
	}
}

// TestProxyFuncRejectsBadManualURL calls the unexported proxyFunc helper
// (allowed because this file is `package netx`, not netx_test).
func TestProxyFuncRejectsBadManualURL(t *testing.T) {
	t.Parallel()

	_, err := proxyFunc(ProxyConfig{Mode: ProxyManual, URL: "ftp://example.com"})
	if err == nil {
		t.Fatal("expected error for ftp proxy")
	}
}
