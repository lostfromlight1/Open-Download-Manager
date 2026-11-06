package engine

import (
	"net/http"
	"testing"
)

func TestApplyExtraHeadersAllowlist(t *testing.T) {
	t.Parallel()
	req, err := http.NewRequest(http.MethodGet, "https://example.com/f", nil)
	if err != nil {
		t.Fatal(err)
	}
	extra := make(http.Header)
	extra.Set("Cookie", "a=1")
	extra.Set("Referer", "https://example.com/")
	extra.Set("Authorization", "Bearer secret")
	ApplyExtraHeaders(req, extra)
	if req.Header.Get("Cookie") != "a=1" {
		t.Fatalf("cookie=%q", req.Header.Get("Cookie"))
	}
	if req.Header.Get("Referer") != "https://example.com/" {
		t.Fatalf("referer=%q", req.Header.Get("Referer"))
	}
	if req.Header.Get("Authorization") != "" {
		t.Fatal("Authorization must not be copied from ExtraHeaders")
	}
}
