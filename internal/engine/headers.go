package engine

import "net/http"

// ApplyExtraHeaders sets allowlisted auth-related headers on an outbound request.
func ApplyExtraHeaders(req *http.Request, extra http.Header) {
	if req == nil || extra == nil {
		return
	}
	if v := extra.Get("Cookie"); v != "" {
		req.Header.Set("Cookie", v)
	}
	if v := extra.Get("Referer"); v != "" {
		req.Header.Set("Referer", v)
	}
}
