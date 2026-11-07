// Package browserhost implements the Chrome/Firefox native-messaging protocol
// and enqueue helpers used by cmd/odm-nmhost.
package browserhost

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// HostName is the native messaging host identifier registered with browsers.
	HostName = "app.opendownloadmanager.native"

	// FirefoxExtensionID is the fixed gecko id for the ODM WebExtension.
	FirefoxExtensionID = "odm@opendownloadmanager.app"

	ProtocolVersion = 1
	MaxMessageBytes = 256 * 1024
)

// Request is the JSON body sent by the extension.
type Request struct {
	V        int    `json:"v"`
	Action   string `json:"action"`
	URL      string `json:"url,omitempty"`
	Filename string `json:"filename,omitempty"`
	// Cookie / Referer are one-shot auth handoff for intercept (never persisted).
	Cookie  string `json:"cookie,omitempty"`
	Referer string `json:"referer,omitempty"`
}

// Response is returned to the extension.
type Response struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
	TaskID  string `json:"taskId,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
}

// ParseRequest validates and decodes a raw JSON payload.
func ParseRequest(raw []byte) (Request, error) {
	if len(raw) == 0 {
		return Request{}, fmt.Errorf("empty message")
	}
	if len(raw) > MaxMessageBytes {
		return Request{}, fmt.Errorf("message too large")
	}
	var req Request
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return Request{}, fmt.Errorf("invalid json: %w", err)
	}
	if req.V != ProtocolVersion {
		return Request{}, fmt.Errorf("unsupported protocol version %d", req.V)
	}
	switch req.Action {
	case "ping", "enqueue", "media_pick":
	default:
		return Request{}, fmt.Errorf("unsupported action %q", req.Action)
	}
	if err := ValidateAuthHeader("cookie", req.Cookie); err != nil {
		return Request{}, err
	}
	if err := ValidateAuthHeader("referer", req.Referer); err != nil {
		return Request{}, err
	}
	return req, nil
}

func errResponse(code, message string) Response {
	return Response{OK: false, Error: code, Message: message}
}
