package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/browserhost"
	"github.com/lostfromlight1/Open-Download-Manager/internal/engine"
	"github.com/lostfromlight1/Open-Download-Manager/internal/manager"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
	"github.com/lostfromlight1/Open-Download-Manager/internal/task"
)

// DownloadService is the thin Wails facade over internal/manager.
//
// It must not implement transfer logic. React calls these methods via bindings.
type DownloadService struct {
	mu            sync.Mutex
	mgr           *manager.Manager
	store         *storage.Store
	decisions     *InteractiveDecisions
	sink          EventSink
	tray          *trayController
	dataDir       string
	ready         bool
	initErr       string
	knownTaskIDs  map[string]struct{}
	reconcileStop chan struct{}
	reconcileOnce sync.Once
}

// NewDownloadService constructs an unready service (tests / late bind).
func NewDownloadService() *DownloadService {
	return &DownloadService{}
}

// setTrayController lets UpdateSettings apply ui.close_to_tray at runtime.
func (s *DownloadService) setTrayController(tray *trayController) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tray = tray
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
		s.knownTaskIDs = map[string]struct{}{}
		s.startReconcileLocked()
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
	dto := settingsToDTO(cfg)
	dto.BrowserHostRegistered = browserhost.IsInstalled()
	return dto, nil
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
	if cfg.Theme == "" {
		cfg.Theme = existing.Theme
	}
	if cfg.BrowserIntegrationEnabled && strings.TrimSpace(cfg.DownloadDirectory) == "" {
		return SettingsDTO{}, fmt.Errorf("set a default download folder before enabling browser integration")
	}
	if err := s.mgr.SaveSettings(context.Background(), cfg); err != nil {
		return SettingsDTO{}, err
	}

	out := settingsToDTO(cfg)
	wasEnabled := existing.BrowserIntegrationEnabled
	switch {
	case cfg.BrowserIntegrationEnabled:
		result, err := browserhost.Install(browserhost.InstallOptions{
			ChromeExtensionID: cfg.ChromeExtensionID,
		})
		if err != nil {
			out.BrowserHostRegistered = false
			out.BrowserHostMessage = browserhost.FormatInstallError(err)
			return out, fmt.Errorf("%s", out.BrowserHostMessage)
		}
		out.BrowserHostRegistered = true
		out.BrowserHostMessage = browserhost.FormatInstallMessage(result, cfg.ChromeExtensionID)
	case wasEnabled:
		result, err := browserhost.Uninstall()
		if err != nil {
			out.BrowserHostRegistered = browserhost.IsInstalled()
			out.BrowserHostMessage = "Settings saved, but removing the browser host failed: " + err.Error()
			return out, fmt.Errorf("%s", out.BrowserHostMessage)
		}
		out.BrowserHostRegistered = false
		out.BrowserHostMessage = browserhost.FormatUninstallMessage(result)
	default:
		out.BrowserHostRegistered = browserhost.IsInstalled()
		out.BrowserHostMessage = ""
	}

	s.mu.Lock()
	tray := s.tray
	s.mu.Unlock()
	if tray != nil {
		tray.SetCloseToTray(cfg.CloseToTray)
	}
	return out, nil
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

// GetDownload returns one task DTO (for the detail panel).
func (s *DownloadService) GetDownload(id string) (DownloadDTO, error) {
	if err := s.requireReady(); err != nil {
		return DownloadDTO{}, err
	}
	t, err := s.mgr.Get(context.Background(), id)
	if err != nil {
		return DownloadDTO{}, err
	}
	return taskToDTO(t), nil
}

// OpenDownload opens a completed file with the OS default app.
// Path comes from the task row — never from an arbitrary UI string.
func (s *DownloadService) OpenDownload(id string) error {
	if err := s.requireReady(); err != nil {
		return err
	}
	t, err := s.mgr.Get(context.Background(), id)
	if err != nil {
		return err
	}
	if t.State != task.StateCompleted {
		return fmt.Errorf("open is only available for completed downloads")
	}
	return openFile(t.DestinationPath)
}

// RevealDownload shows the file in the OS file manager.
func (s *DownloadService) RevealDownload(id string) error {
	if err := s.requireReady(); err != nil {
		return err
	}
	t, err := s.mgr.Get(context.Background(), id)
	if err != nil {
		return err
	}
	path := t.DestinationPath
	if t.State != task.StateCompleted {
		// Reveal partial when present, else destination parent.
		if _, err := os.Stat(t.PartPath()); err == nil {
			path = t.PartPath()
		}
	}
	return revealInFolder(path)
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
	if s.knownTaskIDs != nil {
		s.knownTaskIDs[taskID] = struct{}{}
	}
	s.mu.Unlock()
	if sink == nil || mgr == nil {
		return
	}
	cur, err := mgr.Get(context.Background(), taskID)
	if err != nil {
		s.mu.Lock()
		if s.knownTaskIDs != nil {
			delete(s.knownTaskIDs, taskID)
		}
		s.mu.Unlock()
		sink.Emit(EventDownloadChanged, ChangedEventDTO{TaskID: taskID, Download: nil})
		return
	}
	dto := taskToDTO(cur)
	sink.Emit(EventDownloadChanged, ChangedEventDTO{TaskID: taskID, Download: &dto})
}

// startReconcileLocked emits download-changed for tasks inserted by odm-nmhost
// (a different Manager process), which the desktop UI would otherwise miss.
// Caller must hold s.mu.
func (s *DownloadService) startReconcileLocked() {
	s.reconcileOnce.Do(func() {
		s.reconcileStop = make(chan struct{})
		stop := s.reconcileStop
		go s.reconcileLoop(stop)
	})
}

func (s *DownloadService) reconcileLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s.reconcileOnceTick()
		}
	}
}

func (s *DownloadService) reconcileOnceTick() {
	s.mu.Lock()
	mgr := s.mgr
	sink := s.sink
	ready := s.ready
	s.mu.Unlock()
	if !ready || mgr == nil || sink == nil {
		return
	}
	tasks, err := mgr.List(context.Background())
	if err != nil {
		return
	}

	s.mu.Lock()
	if s.knownTaskIDs == nil {
		s.knownTaskIDs = map[string]struct{}{}
	}
	// First pass: seed without emitting so startup does not spam the UI.
	if len(s.knownTaskIDs) == 0 && len(tasks) > 0 {
		for _, t := range tasks {
			s.knownTaskIDs[t.ID] = struct{}{}
		}
		s.mu.Unlock()
		return
	}
	var added []task.Task
	var removed []string
	seen := make(map[string]struct{}, len(tasks))
	for _, t := range tasks {
		seen[t.ID] = struct{}{}
		if _, ok := s.knownTaskIDs[t.ID]; !ok {
			s.knownTaskIDs[t.ID] = struct{}{}
			added = append(added, t)
		}
	}
	for id := range s.knownTaskIDs {
		if _, ok := seen[id]; !ok {
			delete(s.knownTaskIDs, id)
			removed = append(removed, id)
		}
	}
	s.mu.Unlock()

	for _, t := range added {
		dto := taskToDTO(t)
		sink.Emit(EventDownloadChanged, ChangedEventDTO{TaskID: t.ID, Download: &dto})
	}
	for _, id := range removed {
		sink.Emit(EventDownloadChanged, ChangedEventDTO{TaskID: id, Download: nil})
	}
}
