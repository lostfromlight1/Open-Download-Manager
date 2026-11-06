package netx

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/version"
)

// ClientOptions configures a download-oriented HTTP client.
type ClientOptions struct {
	Proxy ProxyConfig

	// ConnectTimeout bounds how long TCP dial may take.
	ConnectTimeout time.Duration

	// ResponseHeaderTimeout is how long to wait for response headers
	// after the request has been written.
	ResponseHeaderTimeout time.Duration

	// TLSHandshakeTimeout bounds the TLS handshake for https:// URLs.
	TLSHandshakeTimeout time.Duration

	// MaxRedirects stops infinite redirect loops.
	MaxRedirects int

	// UserAgent identifies ODM to servers (polite + debuggable).
	UserAgent string
}

// DefaultClientOptions returns sensible defaults for unreliable networks.
//
// Returning a value (not a pointer) makes it easy to copy and tweak:
//
//	opts := netx.DefaultClientOptions()
//	opts.Proxy.Mode = netx.ProxyOff
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		Proxy: ProxyConfig{
			Mode: ProxySystem,
		},
		ConnectTimeout:        30 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		MaxRedirects:          10,
		UserAgent:             "ODM/" + version.Version,
	}
}

// NewClient builds an *http.Client.
//
// Important security note: we never set InsecureSkipVerify. TLS certificate
// verification stays enabled.
func NewClient(opts ClientOptions) (*http.Client, error) {
	// Fill any zero/negative timeouts with defaults.
	if opts.ConnectTimeout <= 0 {
		opts.ConnectTimeout = DefaultClientOptions().ConnectTimeout
	}
	if opts.ResponseHeaderTimeout <= 0 {
		opts.ResponseHeaderTimeout = DefaultClientOptions().ResponseHeaderTimeout
	}
	if opts.TLSHandshakeTimeout <= 0 {
		opts.TLSHandshakeTimeout = DefaultClientOptions().TLSHandshakeTimeout
	}
	if opts.MaxRedirects <= 0 {
		opts.MaxRedirects = DefaultClientOptions().MaxRedirects
	}
	if opts.UserAgent == "" {
		opts.UserAgent = DefaultClientOptions().UserAgent
	}

	proxy, err := proxyFunc(opts.Proxy)
	if err != nil {
		return nil, err
	}

	// http.Transport is the low-level connection pool / dialer / proxy layer.
	transport := &http.Transport{
		Proxy: proxy,
		// net.Dialer creates TCP connections. Timeout here is connect timeout.
		DialContext: (&net.Dialer{
			Timeout:   opts.ConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   opts.TLSHandshakeTimeout,
		ResponseHeaderTimeout: opts.ResponseHeaderTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Capture maxRedirects in a local variable for the closure below.
	maxRedirects := opts.MaxRedirects

	client := &http.Client{
		Transport: transport,
		// CheckRedirect is called before following an HTTP redirect.
		// Returning an error stops redirect following.
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			// via is the list of requests already made in this redirect chain.
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil // allow this redirect
		},
		// Client.Timeout would cover the whole request including body download.
		// We leave it 0 and use context + stall detection instead, so large
		// files are not killed by a single global timer.
		Timeout: 0,
	}
	return client, nil
}

// ApplyUserAgent sets the User-Agent header when the caller did not set one.
func ApplyUserAgent(req *http.Request, userAgent string) {
	if req.Header.Get("User-Agent") == "" {
		if userAgent == "" {
			userAgent = DefaultClientOptions().UserAgent
		}
		req.Header.Set("User-Agent", userAgent)
	}
}
