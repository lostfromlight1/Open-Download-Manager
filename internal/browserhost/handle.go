package browserhost

import (
	"context"
	"io"
	"path/filepath"
	"strings"

	"github.com/lostfromlight1/Open-Download-Manager/internal/manager"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
)

// Handler enqueues browser-originated downloads through the shared manager.
type Handler struct {
	Store *storage.Store
	Mgr   *manager.Manager
}

// Handle processes one parsed request.
func (h *Handler) Handle(ctx context.Context, req Request) Response {
	if h == nil || h.Store == nil || h.Mgr == nil {
		return errResponse("unavailable", "download host not initialized")
	}
	cfg, err := h.Store.GetSettings(ctx)
	if err != nil {
		return errResponse("settings", err.Error())
	}

	switch req.Action {
	case "ping":
		return Response{OK: true, Enabled: cfg.BrowserIntegrationEnabled}
	case "enqueue":
		return h.enqueue(ctx, cfg, req)
	default:
		return errResponse("unsupported_action", req.Action)
	}
}

func (h *Handler) enqueue(ctx context.Context, cfg storage.Settings, req Request) Response {
	if !cfg.BrowserIntegrationEnabled {
		return errResponse("disabled", "browser integration is disabled in ODM settings")
	}
	dir := strings.TrimSpace(cfg.DownloadDirectory)
	if dir == "" {
		return errResponse("no_directory", "set a default download folder in ODM settings")
	}

	url := strings.TrimSpace(req.URL)
	name := SanitizeFilename(req.Filename)
	if name == "" {
		name = FilenameFromURL(url)
	}
	dest := filepath.Join(dir, name)

	t, err := h.Mgr.Add(ctx, manager.AddOptions{
		URL:         url,
		Destination: dest,
		DisplayName: name,
		Cookie:      strings.TrimSpace(req.Cookie),
		Referer:     strings.TrimSpace(req.Referer),
	})
	if err != nil {
		return errResponse("enqueue_failed", err.Error())
	}
	return Response{OK: true, TaskID: t.ID, Enabled: true}
}

// HandleRaw parses raw JSON then handles it.
func (h *Handler) HandleRaw(ctx context.Context, raw []byte) Response {
	req, err := ParseRequest(raw)
	if err != nil {
		return errResponse("invalid_message", err.Error())
	}
	return h.Handle(ctx, req)
}

// ServeIO reads native-messaging frames until EOF and writes JSON responses.
func (h *Handler) ServeIO(ctx context.Context, r io.Reader, w io.Writer) error {
	for {
		raw, err := ReadMessage(r)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		resp := h.HandleRaw(ctx, raw)
		if err := WriteJSON(w, resp); err != nil {
			return err
		}
	}
}
