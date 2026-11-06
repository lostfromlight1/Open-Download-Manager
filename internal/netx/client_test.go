package netx

// These tests use `package netx` (same package), so they can call unexported
// helpers if needed and access Transport internals for assertions.

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestNewClientUsesSystemProxyByDefault checks the default proxy function is set.
//
// t.Parallel() allows this test to run at the same time as other parallel tests,
// which speeds up `go test` on multi-core machines.
func TestNewClientUsesSystemProxyByDefault(t *testing.T) {
	t.Parallel()

	client, err := NewClient(DefaultClientOptions())
	if err != nil {
		t.Fatal(err)
	}
	// Type assertion: client.Transport is an interface (http.RoundTripper).
	// transport, ok := x.(*http.Transport) asks "is the concrete type *http.Transport?"
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.Proxy == nil {
		t.Fatal("expected transport with proxy function")
	}
}

func TestNewClientProxyOff(t *testing.T) {
	t.Parallel()

	opts := DefaultClientOptions()
	opts.Proxy = ProxyConfig{Mode: ProxyOff}
	client, err := NewClient(opts)
	if err != nil {
		t.Fatal(err)
	}
	transport := client.Transport.(*http.Transport)
	if transport.Proxy != nil {
		t.Fatal("expected nil proxy function when proxy is off")
	}
}

// TestClientFetchesOverHTTP is a small end-to-end check against httptest.
func TestClientFetchesOverHTTP(t *testing.T) {
	t.Parallel()

	// http.HandlerFunc converts an ordinary function into an http.Handler.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// `_` discards the unused *http.Request parameter.
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close) // close when the test ends

	opts := DefaultClientOptions()
	opts.Proxy = ProxyConfig{Mode: ProxyOff}
	opts.ConnectTimeout = 2 * time.Second
	client, err := NewClient(opts)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	// defer Close so we do not leak the connection if later assertions fail.
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
