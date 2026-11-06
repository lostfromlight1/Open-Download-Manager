package main

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/manager"
	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
	"github.com/lostfromlight1/Open-Download-Manager/internal/testserver"
)

type recordSink struct {
	mu     sync.Mutex
	events []struct {
		name string
		data any
	}
}

func (r *recordSink) Emit(name string, data any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, struct {
		name string
		data any
	}{name: name, data: data})
}

func (r *recordSink) count(name string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.events {
		if e.name == name {
			n++
		}
	}
	return n
}

func setupService(t *testing.T) (*DownloadService, *recordSink) {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "odm.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := storage.DefaultSettings()
	cfg.ProxyMode = netx.ProxyOff
	cfg.MaxActiveDownloads = 1
	cfg.DownloadDirectory = t.TempDir()
	if err := store.SaveSettings(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	sink := &recordSink{}
	decisions := NewInteractiveDecisions(sink)
	mgr := manager.New(store, decisions)
	svc := NewDownloadService()
	svc.bind(mgr, store, decisions, sink, t.TempDir())
	return svc, sink
}

func TestDownloadServiceListAddSettings(t *testing.T) {
	svc, _ := setupService(t)

	list, err := svc.ListDownloads()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("want empty list, got %d", len(list))
	}

	body := []byte("desktop-svc-body")
	srv := testserver.Start(testserver.Options{Body: body})
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "a.bin")
	dto, err := svc.AddDownload(AddDownloadRequest{URL: srv.URL, Destination: dest})
	if err != nil {
		t.Fatal(err)
	}
	if dto.ID == "" || dto.Status != "queued" {
		t.Fatalf("bad dto: %#v", dto)
	}

	list, err = svc.ListDownloads()
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}

	settings, err := svc.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	settings.MaxActiveDownloads = 3
	settings.ProxyMode = "off"
	saved, err := svc.UpdateSettings(settings)
	if err != nil {
		t.Fatal(err)
	}
	if saved.MaxActiveDownloads != 3 {
		t.Fatalf("settings not saved: %#v", saved)
	}
}

func TestDownloadServiceCancelKeep(t *testing.T) {
	svc, sink := setupService(t)
	body := []byte("cancel-me-please-long-enough")
	srv := testserver.Start(testserver.Options{Behavior: testserver.BehaviorSlowFirstByte, Body: body, SlowDelay: 2 * time.Second})
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "c.bin")
	dto, err := svc.AddDownload(AddDownloadRequest{URL: srv.URL, Destination: dest})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = svc.mgr.RunQueue(ctx, true) }()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		got, _ := svc.mgr.Get(context.Background(), dto.ID)
		if got.State == "downloading" || got.State == "inspecting" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if err := svc.CancelDownload(dto.ID, "keep"); err != nil {
		t.Fatal(err)
	}
	if sink.count(EventDownloadChanged) < 1 {
		t.Fatalf("expected change events")
	}
}

func TestInteractiveDecisionResolve(t *testing.T) {
	sink := &recordSink{}
	d := NewInteractiveDecisions(sink)
	needUI := make(chan struct{}, 1)
	d.SetNeedUI(func() {
		select {
		case needUI <- struct{}{}:
		default:
		}
	})

	done := make(chan manager.MismatchAction, 1)
	go func() {
		action, err := d.OnResumeMismatch(context.Background(), manager.ValidatorMismatch{TaskID: "t1"})
		if err != nil {
			t.Errorf("mismatch wait: %v", err)
			return
		}
		done <- action
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && sink.count(EventDecisionRequired) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if sink.count(EventDecisionRequired) != 1 {
		t.Fatal("expected decision-required event")
	}
	select {
	case <-needUI:
	case <-time.After(2 * time.Second):
		t.Fatal("expected NeedUI callback for resume-mismatch decision")
	}

	sink.mu.Lock()
	ev := sink.events[0].data.(DecisionRequiredDTO)
	sink.mu.Unlock()

	if err := d.Resolve(ev.DecisionID, "restart"); err != nil {
		t.Fatal(err)
	}
	select {
	case a := <-done:
		if a != manager.MismatchRestart {
			t.Fatalf("got %q", a)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for decision")
	}

	if err := d.Resolve(ev.DecisionID, "abort"); err == nil {
		t.Fatal("expected stale decision error")
	}
}
