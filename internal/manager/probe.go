package manager

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
	"github.com/lostfromlight1/Open-Download-Manager/internal/version"
)

// probeValidators performs a HEAD request to read ETag / Last-Modified.
//
// HEAD is an optimization: if the server rejects HEAD, we return an error and
// the manager lets the normal GET path proceed without forcing a mismatch.
func probeValidators(ctx context.Context, rawURL string, cfg storage.Settings) (etag, lastModified string, err error) {
	client, err := netx.NewClient(netx.ClientOptions{
		Proxy:                 netx.ProxyConfig{Mode: cfg.ProxyMode, URL: cfg.ProxyURL},
		ConnectTimeout:        time.Duration(cfg.ConnectTimeoutSec) * time.Second,
		ResponseHeaderTimeout: time.Duration(cfg.ConnectTimeoutSec) * time.Second,
	})
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return "", "", err
	}
	netx.ApplyUserAgent(req, "ODM/"+version.Version)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", "", fmt.Errorf("HEAD status %d", resp.StatusCode)
	}
	return resp.Header.Get("ETag"), resp.Header.Get("Last-Modified"), nil
}
