package manager_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/manager"
	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
	"github.com/lostfromlight1/Open-Download-Manager/internal/task"
	"github.com/lostfromlight1/Open-Download-Manager/internal/testserver"
)

func openManager(t *testing.T, decisions manager.DecisionHandler) (*manager.Manager, *storage.Store) {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "odm.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := storage.DefaultSettings()
	cfg.ProxyMode = netx.ProxyOff
	cfg.MaxActiveDownloads = 1
	cfg.MaxRetries = 2
	cfg.ConnectTimeoutSec = 5
	cfg.StallTimeoutSec = 5
	if err := store.SaveSettings(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	return manager.New(store, decisions), store
}

func TestManagerDownloadsQueuedTask(t *testing.T) {
	body := []byte("phase2-queue-body")
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorOK, Body: body})
	t.Cleanup(srv.Close)

	m, _ := openManager(t, manager.StaticDecisions{
		Cancel:   manager.FileKeep,
		Remove:   manager.FileDelete,
		Mismatch: manager.MismatchAbort,
	})

	dest := filepath.Join(t.TempDir(), "file.bin")
	tk, err := m.Add(context.Background(), manager.AddOptions{URL: srv.URL, Destination: dest})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := m.RunQueue(ctx, false); err != nil && ctx.Err() == nil {
		t.Fatal(err)
	}

	// Wait briefly for worker completion.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, err := m.Get(context.Background(), tk.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.State == task.StateCompleted {
			data, err := os.ReadFile(dest)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != string(body) {
				t.Fatalf("got %q", data)
			}
			return
		}
		if got.State == task.StateFailed {
			t.Fatalf("failed: %s %s", got.ErrorCode, got.ErrorMessage)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("timeout waiting for completion")
}

func TestManagerCancelDeleteRemovesPart(t *testing.T) {
	// Slow/stall server so cancel happens mid-flight.
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorStall, Body: []byte("xxxx")})
	t.Cleanup(srv.Close)

	m, _ := openManager(t, nil)
	dest := filepath.Join(t.TempDir(), "stall.bin")
	tk, err := m.Add(context.Background(), manager.AddOptions{URL: srv.URL, Destination: dest})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.RunQueue(ctx, true) }()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		got, _ := m.Get(context.Background(), tk.ID)
		if got.State == task.StateDownloading || got.State == task.StateInspecting {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if err := m.Cancel(context.Background(), tk.ID, manager.FileDelete); err != nil {
		t.Fatal(err)
	}
	got, err := m.Get(context.Background(), tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != task.StateCancelled {
		t.Fatalf("state=%s", got.State)
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatalf("expected part deleted, err=%v", err)
	}
}

func TestManagerResumeMismatchAbort(t *testing.T) {
	body := []byte("hello-validators")
	srv := testserver.Start(testserver.Options{
		Behavior: testserver.BehaviorOK,
		Body:     body,
		ETag:     `"remote-etag"`,
	})
	t.Cleanup(srv.Close)

	m, store := openManager(t, manager.StaticDecisions{Mismatch: manager.MismatchAbort})
	dest := filepath.Join(t.TempDir(), "v.bin")

	// Seed a partial file + saved validators that disagree with the server.
	if err := os.WriteFile(dest+".part", body[:4], 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	tk := task.Task{
		ID:              task.NewID(),
		URL:             srv.URL,
		DisplayName:     "v.bin",
		DestinationPath: dest,
		State:           task.StateQueued,
		ETag:            `"old-etag"`,
		DownloadedBytes: 4,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := store.InsertTask(context.Background(), tk); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = m.RunQueue(ctx, false)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, err := m.Get(context.Background(), tk.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.State == task.StateFailed {
			if got.ErrorCode != "resume_mismatch" {
				t.Fatalf("code=%s msg=%s", got.ErrorCode, got.ErrorMessage)
			}
			return
		}
		time.Sleep(40 * time.Millisecond)
	}
	t.Fatal("expected resume_mismatch failure")
}

func TestManagerRemoveKeepLeavesFile(t *testing.T) {
	body := []byte("keep-me")
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorOK, Body: body})
	t.Cleanup(srv.Close)

	m, _ := openManager(t, nil)
	dest := filepath.Join(t.TempDir(), "keep.bin")
	tk, err := m.Add(context.Background(), manager.AddOptions{URL: srv.URL, Destination: dest})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = m.RunQueue(ctx, false)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, _ := m.Get(context.Background(), tk.ID)
		if got.State == task.StateCompleted {
			break
		}
		time.Sleep(40 * time.Millisecond)
	}

	if err := m.Remove(context.Background(), tk.ID, manager.FileKeep); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Get(context.Background(), tk.ID); err == nil {
		t.Fatal("expected not found")
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("file should remain: %v", err)
	}
}
