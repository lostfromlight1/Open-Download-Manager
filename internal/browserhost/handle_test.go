package browserhost

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lostfromlight1/Open-Download-Manager/internal/manager"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
)

func TestHandleEnqueueDisabledAndHappyPath(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "odm.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	mgr := manager.New(store, manager.StaticDecisions{
		Cancel:   manager.FileKeep,
		Remove:   manager.FileKeep,
		Mismatch: manager.MismatchAbort,
	})
	h := &Handler{Store: store, Mgr: mgr}
	ctx := context.Background()

	ping := h.Handle(ctx, Request{V: 1, Action: "ping"})
	if !ping.OK || ping.Enabled {
		t.Fatalf("ping while disabled: %+v", ping)
	}

	bad := h.Handle(ctx, Request{V: 1, Action: "enqueue", URL: "https://example.com/a.bin"})
	if bad.OK || bad.Error != "disabled" {
		t.Fatalf("expected disabled: %+v", bad)
	}

	cfg := storage.DefaultSettings()
	cfg.BrowserIntegrationEnabled = true
	cfg.DownloadDirectory = filepath.Join(dir, "dl")
	if err := store.SaveSettings(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	noDir := h.Handle(ctx, Request{V: 1, Action: "enqueue", URL: "https://example.com/a.bin"})
	// directory is set — should succeed; clear dir for no_directory case
	_ = noDir

	cfg.DownloadDirectory = ""
	if err := store.SaveSettings(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	noDir = h.Handle(ctx, Request{V: 1, Action: "enqueue", URL: "https://example.com/a.bin"})
	if noDir.OK || noDir.Error != "no_directory" {
		t.Fatalf("expected no_directory: %+v", noDir)
	}

	cfg.DownloadDirectory = filepath.Join(dir, "dl")
	if err := store.SaveSettings(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	scheme := h.Handle(ctx, Request{V: 1, Action: "enqueue", URL: "ftp://example.com/a.bin"})
	if scheme.OK {
		t.Fatalf("expected scheme reject: %+v", scheme)
	}

	ok := h.Handle(ctx, Request{
		V:        1,
		Action:   "enqueue",
		URL:      "https://example.com/path/file.bin",
		Filename: "mine.bin",
	})
	if !ok.OK || ok.TaskID == "" {
		t.Fatalf("enqueue: %+v", ok)
	}
	tk, err := mgr.Get(ctx, ok.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if tk.DisplayName != "mine.bin" {
		t.Fatalf("name=%q", tk.DisplayName)
	}
	if filepath.Base(tk.DestinationPath) != "mine.bin" {
		t.Fatalf("dest=%q", tk.DestinationPath)
	}
}
