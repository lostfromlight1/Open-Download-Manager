package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lostfromlight1/Open-Download-Manager/internal/engine"
	"github.com/lostfromlight1/Open-Download-Manager/internal/manager"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
	"github.com/lostfromlight1/Open-Download-Manager/internal/task"
)

// DownloadService is the thin Wails facade over internal/manager.
//
// It must not implement transfer logic. React calls these methods via bindings.
type DownloadService struct {
	mu        sync.Mutex
	mgr       *manager.Manager
	store     *storage.Store
	decisions *InteractiveDecisions
	sink      EventSink
	dataDir   string
	ready     bool
	initErr   string
}

// NewDownloadService constructs an unready service (tests / late bind).
func NewDownloadService() *DownloadService {
	return &DownloadService{}
}

// bind wires the opened manager/store and starts listening for task updates.
// Unexported so Wails does not generate bindings for this bootstrap method.
func (s *DownloadService) bind(mgr *manager.Manager, store *storage.Store, decisions *InteractiveDecisions, sink EventSink, dataDir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mgr = mgr
	s.store = store
	s.decisions = decisions
	s.sink = sink
	s.dataDir = dataDir
	s.ready = mgr != nil && store != nil
	s.initErr = ""
	if s.ready {
		mgr.SetListener(managerListener{svc: s})
	}
}

// markFailed records a startup failure for the UI.
func (s *DownloadService) markFailed(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ready = false
	if err != nil {
		s.initErr = err.Error()
	} else {
		s.initErr = "download service unavailable"
	}
}

// Ready reports whether the manager is usable.
func (s *DownloadService) Ready() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ready
}

// StatusMessage is a short human-readable status for the shell.
func (s *DownloadService) StatusMessage() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ready {
		return "Download manager ready"
	}
	if s.initErr != "" {
		return s.initErr
	}
	return "Download manager not initialized"
}

// DataDir returns the resolved ODM data directory.
func (s *DownloadService) DataDir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dataDir
}

func (s *DownloadService) requireReady() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready || s.mgr == nil {
		if s.initErr != "" {
			return fmt.Errorf("%s", s.initErr)
		}
		return fmt.Errorf("download service not ready")
	}
	return nil
}

// ListDownloads returns an authoritative snapshot of all tasks.
func (s *DownloadService) ListDownloads() ([]DownloadDTO, error) {
	if err := s.requireReady(); err != nil {
		return nil, err
	}
	tasks, err := s.mgr.List(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]DownloadDTO, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, taskToDTO(t))
	}
	return out, nil
}

// AddDownload validates and enqueues a new task.
func (s *DownloadService) AddDownload(req AddDownloadRequest) (DownloadDTO, error) {
	if err := s.requireReady(); err != nil {
		return DownloadDTO{}, err
	}
	url := strings.TrimSpace(req.URL)
	dest := strings.TrimSpace(req.Destination)
	if dest == "" {
		cfg, err := s.mgr.Settings(context.Background())
		if err != nil {
			return DownloadDTO{}, err
		}
		if cfg.DownloadDirectory == "" {
			return DownloadDTO{}, fmt.Errorf("%w: destination required", engine.ErrInvalidDestination)
		}
		// Derive filename from URL path when only a directory setting exists.
		base := filepath.Base(strings.TrimRight(url, "/"))
		if base == "" || base == "." || base == "/" {
			base = "download.bin"
		}
		dest = filepath.Join(cfg.DownloadDirectory, base)
	}
	t, err := s.mgr.Add(context.Background(), manager.AddOptions{
		URL:         url,
		Destination: dest,
	})
	if err != nil {
		return DownloadDTO{}, err
	}
	return taskToDTO(t), nil
}

// PauseDownload pauses an active or queued task.
func (s *DownloadService) PauseDownload(id string) error {
	if err := s.requireReady(); err != nil {
		return err
	}
	return s.mgr.Pause(context.Background(), id)
}

// ResumeDownload re-queues a paused/failed/cancelled task.
func (s *DownloadService) ResumeDownload(id string) error {
	if err := s.requireReady(); err != nil {
		return err
	}
	return s.mgr.Resume(context.Background(), id)
}

// CancelDownload stops a task. action must be "keep" or "delete".
func (s *DownloadService) CancelDownload(id, action string) error {
	if err := s.requireReady(); err != nil {
		return err
	}
	files, err := manager.ParseFileAction(action)
	if err != nil {
		return err
	}
	return s.mgr.Cancel(context.Background(), id, files)
}

// RemoveDownload deletes task metadata. action must be "keep" or "delete".
func (s *DownloadService) RemoveDownload(id, action string) error {
	if err := s.requireReady(); err != nil {
		return err
	}
	files, err := manager.ParseFileAction(action)
	if err != nil {
		return err
	}
	return s.mgr.Remove(context.Background(), id, files)
}

// GetSettings loads persisted settings.
func (s *DownloadService) GetSettings() (SettingsDTO, error) {
	if err := s.requireReady(); err != nil {
		return SettingsDTO{}, err
	}
	cfg, err := s.mgr.Settings(context.Background())
	if err != nil {
		return SettingsDTO{}, err
	}
	return settingsToDTO(cfg), nil
}

// UpdateSettings validates and persists settings.
func (s *DownloadService) UpdateSettings(dto SettingsDTO) (SettingsDTO, error) {
	if err := s.requireReady(); err != nil {
		return SettingsDTO{}, err
	}
	cfg, err := settingsFromDTO(dto)
	if err != nil {
		return SettingsDTO{}, err
	}
	// Preserve timeout/retry fields not shown in the UI.
	existing, err := s.mgr.Settings(context.Background())
	if err != nil {
		return SettingsDTO{}, err
	}
	cfg.MaxRetries = existing.MaxRetries
	cfg.ConnectTimeoutSec = existing.ConnectTimeoutSec
	cfg.StallTimeoutSec = existing.StallTimeoutSec
	if err := s.mgr.SaveSettings(context.Background(), cfg); err != nil {
		return SettingsDTO{}, err
	}
	return settingsToDTO(cfg), nil
}

// ResolveDecision answers a pending resume-mismatch prompt.
func (s *DownloadService) ResolveDecision(decisionID, choice string) error {
	s.mu.Lock()
	d := s.decisions
	s.mu.Unlock()
	if d == nil {
		return fmt.Errorf("decision bridge not available")
	}
	return d.Resolve(decisionID, choice)
}

// managerListener adapts DownloadService to manager.Listener without exporting
// those methods on the Wails service surface.
type managerListener struct {
	svc *DownloadService
}

func (l managerListener) OnProgress(taskID string, ev engine.ProgressEvent) {
	s := l.svc
	s.mu.Lock()
	sink := s.sink
	mgr := s.mgr
	s.mu.Unlock()
	if sink == nil {
		return
	}
	var total *int64
	if ev.Total > 0 {
		t := ev.Total
		total = &t
	}
	var bps *float64
	if ev.BytesPerSec > 0 {
		v := ev.BytesPerSec
		bps = &v
	}
	status := string(task.StateDownloading)
	if mgr != nil {
		if cur, err := mgr.Get(context.Background(), taskID); err == nil {
			status = string(cur.State)
		}
	}
	sink.Emit(EventDownloadProgress, ProgressEventDTO{
		TaskID:          taskID,
		DownloadedBytes: ev.Downloaded,
		TotalBytes:      total,
		BytesPerSecond:  bps,
		Status:          status,
	})
}

func (l managerListener) OnChanged(taskID string) {
	s := l.svc
	s.mu.Lock()
	sink := s.sink
	mgr := s.mgr
	s.mu.Unlock()
	if sink == nil || mgr == nil {
		return
	}
	cur, err := mgr.Get(context.Background(), taskID)
	if err != nil {
		sink.Emit(EventDownloadChanged, ChangedEventDTO{TaskID: taskID, Download: nil})
		return
	}
	dto := taskToDTO(cur)
	sink.Emit(EventDownloadChanged, ChangedEventDTO{TaskID: taskID, Download: &dto})
}
