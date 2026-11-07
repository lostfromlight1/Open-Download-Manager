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
	"github.com/lostfromlight1/Open-Download-Manager/internal/media"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
	"github.com/lostfromlight1/Open-Download-Manager/internal/task"
)

// DownloadService is the thin Wails facade over internal/manager.
//
// It must not implement transfer logic. React calls these methods via bindings.
type DownloadService struct {
	mu             sync.Mutex
	mgr            *manager.Manager
	store          *storage.Store
	decisions      *InteractiveDecisions
	sink           EventSink
	tray           *trayController
	speeds         map[string]float64 // taskID → latest bytes/sec for tray tooltip
	dataDir        string
	ready          bool
	initErr        string
	knownTaskIDs   map[string]struct{}
	reconcileStop  chan struct{}
	reconcileOnce  sync.Once
	needUI         func()
	mediaToolsBusy bool
	mediaToolsErr  string
	mediaToolsMsg  string
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
	if tray != nil {
		tray.setActions(s)
	}
}

// setNeedUI registers a callback to bring the main window forward (media pick, etc.).
func (s *DownloadService) setNeedUI(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.needUI = fn
}

// PauseAllActive pauses queued/active downloads (tray menu).
func (s *DownloadService) PauseAllActive() {
	if err := s.requireReady(); err != nil {
		return
	}
	list, err := s.mgr.List(context.Background())
	if err != nil {
		return
	}
	for _, t := range list {
		if task.IsActive(t.State) || t.State == task.StateQueued {
			_ = s.mgr.Pause(context.Background(), t.ID)
		}
	}
}

// ResumeAllPaused resumes paused tasks (tray menu).
func (s *DownloadService) ResumeAllPaused() {
	if err := s.requireReady(); err != nil {
		return
	}
	list, err := s.mgr.List(context.Background())
	if err != nil {
		return
	}
	for _, t := range list {
		if t.State == task.StatePaused {
			_ = s.mgr.Resume(context.Background(), t.ID)
		}
	}
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
		mgr.SetDataDir(dataDir)
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
	if cfg.MediaEnabled && media.NeedsEnsure(s.dataDir, cfg.MediaYtDlpPath, cfg.MediaFFmpegPath) {
		s.startMediaToolsEnsure()
	}
	s.fillMediaToolsStatus(&dto, cfg)
	return dto, nil
}

func (s *DownloadService) fillMediaToolsStatus(dto *SettingsDTO, cfg storage.Settings) {
	tools := media.ResolveTools(s.dataDir, cfg.MediaYtDlpPath, cfg.MediaFFmpegPath)
	needs := media.NeedsEnsure(s.dataDir, cfg.MediaYtDlpPath, cfg.MediaFFmpegPath)
	dto.MediaToolsReady = tools.YtDlpReady() && tools.FFmpegReady() && !needs

	s.mu.Lock()
	busy := s.mediaToolsBusy
	ensureErr := s.mediaToolsErr
	ensureMsg := s.mediaToolsMsg
	s.mu.Unlock()

	switch {
	case !cfg.MediaEnabled:
		dto.MediaToolsMessage = "YouTube downloads are disabled."
	case busy:
		if ensureMsg != "" {
			dto.MediaToolsMessage = ensureMsg
		} else {
			dto.MediaToolsMessage = "Downloading helpers (ffmpeg is large; leave ODM open)…"
		}
	case ensureErr != "" && needs:
		dto.MediaToolsMessage = "Helper install issue: " + ensureErr + " — click Retry install."
	case needs && tools.YtDlpReady():
		dto.MediaToolsMessage = "yt-dlp update available (pin " + media.PinnedYtDlpVersion + "). Click Retry helper install."
	case tools.YtDlpReady() && tools.FFmpegReady():
		dto.MediaToolsMessage = "yt-dlp and ffmpeg are ready (" + media.PinnedYtDlpVersion + ")."
	case tools.YtDlpReady():
		dto.MediaToolsMessage = "yt-dlp ready; ffmpeg missing (some formats need it). Save again or Retry install — ffmpeg is ~190MB and resumes if interrupted."
	default:
		dto.MediaToolsMessage = "Helpers not installed yet. Save with media enabled to download them."
	}
}

// startMediaToolsEnsure downloads yt-dlp/ffmpeg in the background so Settings
// Save is not blocked on the large ffmpeg zip (which was leaving .part files).
func (s *DownloadService) startMediaToolsEnsure() {
	s.mu.Lock()
	if !s.ready || s.dataDir == "" || s.mediaToolsBusy {
		s.mu.Unlock()
		return
	}
	s.mediaToolsBusy = true
	s.mediaToolsErr = ""
	s.mediaToolsMsg = "Downloading helpers…"
	dataDir := s.dataDir
	s.mu.Unlock()

	go func() {
		_, err := media.EnsureTools(context.Background(), dataDir, nil, func(msg string) {
			s.mu.Lock()
			s.mediaToolsMsg = msg
			s.mu.Unlock()
		})
		s.mu.Lock()
		s.mediaToolsBusy = false
		s.mediaToolsMsg = ""
		if err != nil {
			s.mediaToolsErr = err.Error()
		} else {
			s.mediaToolsErr = ""
		}
		s.mu.Unlock()
	}()
}

// EnsureMediaTools re-downloads pinned yt-dlp (and missing ffmpeg) then returns status.
// Used by Settings “Check for helper updates” / Retry.
func (s *DownloadService) EnsureMediaTools() (SettingsDTO, error) {
	if err := s.requireReady(); err != nil {
		return SettingsDTO{}, err
	}
	cfg, err := s.mgr.Settings(context.Background())
	if err != nil {
		return SettingsDTO{}, err
	}
	if !cfg.MediaEnabled {
		return SettingsDTO{}, fmt.Errorf("enable YouTube downloads first")
	}
	media.ForceRefreshManagedYtDlp(s.dataDir)
	s.startMediaToolsEnsure()
	dto := settingsToDTO(cfg)
	s.fillMediaToolsStatus(&dto, cfg)
	dto.BrowserHostRegistered = browserhost.IsInstalled()
	dto.MediaToolsMessage = "Re-downloading yt-dlp " + media.PinnedYtDlpVersion + "…"
	return dto, nil
}

// ProbeMediaURL runs yt-dlp -J for a YouTube URL (requires media enabled + tools).
func (s *DownloadService) ProbeMediaURL(rawURL string) (MediaProbeDTO, error) {
	if err := s.requireReady(); err != nil {
		return MediaProbeDTO{}, err
	}
	cfg, err := s.mgr.Settings(context.Background())
	if err != nil {
		return MediaProbeDTO{}, err
	}
	if !cfg.MediaEnabled {
		return MediaProbeDTO{}, fmt.Errorf("enable YouTube downloads in Settings first")
	}
	tools := media.ResolveTools(s.dataDir, cfg.MediaYtDlpPath, cfg.MediaFFmpegPath)
	if !tools.YtDlpReady() {
		return MediaProbeDTO{}, fmt.Errorf("yt-dlp is not installed; save Settings with media enabled")
	}
	res, err := media.Probe(context.Background(), tools, rawURL)
	if err != nil {
		return MediaProbeDTO{}, err
	}
	out := MediaProbeDTO{
		Kind:    res.Kind,
		ID:      res.ID,
		Title:   res.Title,
		Webpage: res.Webpage,
	}
	for _, f := range res.Formats {
		out.Formats = append(out.Formats, MediaFormatDTO{
			ID: f.ID, Ext: f.Ext, Resolution: f.Resolution, Note: f.Note,
			IsAudio: f.IsAudio, VideoOnly: f.VideoOnly, Height: f.Height,
		})
	}
	for _, e := range res.Entries {
		out.Entries = append(out.Entries, MediaPlaylistEntryDTO{ID: e.ID, Title: e.Title, URL: e.URL})
	}
	return out, nil
}

// AddMediaDownload enqueues one YouTube video after format selection.
func (s *DownloadService) AddMediaDownload(req AddMediaRequest) (DownloadDTO, error) {
	if err := s.requireReady(); err != nil {
		return DownloadDTO{}, err
	}
	cfg, err := s.mgr.Settings(context.Background())
	if err != nil {
		return DownloadDTO{}, err
	}
	if !cfg.MediaEnabled {
		return DownloadDTO{}, fmt.Errorf("enable YouTube downloads in Settings first")
	}
	dir := strings.TrimSpace(req.DestDir)
	if dir == "" {
		dir = strings.TrimSpace(cfg.DownloadDirectory)
	}
	if dir == "" {
		return DownloadDTO{}, fmt.Errorf("set a default download folder in Settings")
	}
	t, err := s.mgr.AddMedia(context.Background(), manager.AddMediaOptions{
		WebpageURL:  req.WebpageURL,
		FormatID:    req.FormatID,
		DestDir:     dir,
		DisplayName: req.DisplayName,
		PlaylistID:  req.PlaylistID,
	})
	if err != nil {
		return DownloadDTO{}, err
	}
	return taskToDTO(t), nil
}

// IsYouTubeURL reports whether a URL should use the media probe flow.
func (s *DownloadService) IsYouTubeURL(rawURL string) bool {
	return media.IsYouTubeURL(rawURL)
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
	if cfg.MediaEnabled && strings.TrimSpace(cfg.DownloadDirectory) == "" {
		return SettingsDTO{}, fmt.Errorf("set a default download folder before enabling YouTube downloads")
	}
	if err := s.mgr.SaveSettings(context.Background(), cfg); err != nil {
		return SettingsDTO{}, err
	}

	out := settingsToDTO(cfg)
	if cfg.MediaEnabled {
		// ffmpeg zip is large; install asynchronously and resume .part files.
		s.startMediaToolsEnsure()
	}
	s.fillMediaToolsStatus(&out, cfg)

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
		tray.SetLaunchAtLogin(cfg.LaunchAtLogin)
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
	if _, err := os.Stat(t.DestinationPath); err != nil {
		return fmt.Errorf("file not found at destination — it may have been moved, or the merge left unusable fragments; try Show in folder or re-download")
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

	s.mu.Lock()
	if s.speeds == nil {
		s.speeds = make(map[string]float64)
	}
	if ev.BytesPerSec > 0 {
		s.speeds[taskID] = ev.BytesPerSec
	} else {
		delete(s.speeds, taskID)
	}
	var sum float64
	for _, v := range s.speeds {
		sum += v
	}
	tray := s.tray
	s.mu.Unlock()
	if tray != nil {
		tray.SetAggregateSpeed(sum)
	}
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

	if !task.IsActive(cur.State) {
		s.mu.Lock()
		if s.speeds != nil {
			delete(s.speeds, taskID)
		}
		var sum float64
		for _, v := range s.speeds {
			sum += v
		}
		tray := s.tray
		s.mu.Unlock()
		if tray != nil {
			tray.SetAggregateSpeed(sum)
		}
	}
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
	store := s.store
	sink := s.sink
	ready := s.ready
	s.mu.Unlock()
	if !ready || mgr == nil || sink == nil {
		return
	}
	s.claimPendingMediaPick(store, sink)
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

// claimPendingMediaPick reads a one-shot YouTube URL left by odm-nmhost and
// asks the UI to open the format picker (shows the window if hidden).
func (s *DownloadService) claimPendingMediaPick(store *storage.Store, sink EventSink) {
	if store == nil || sink == nil {
		return
	}
	raw, err := store.GetSetting(context.Background(), storage.SettingMediaPendingPickURL)
	if err != nil || strings.TrimSpace(raw) == "" {
		return
	}
	url := strings.TrimSpace(raw)
	_ = store.SetSetting(context.Background(), storage.SettingMediaPendingPickURL, "")

	s.mu.Lock()
	needUI := s.needUI
	s.mu.Unlock()

	if needUI != nil {
		needUI()
	}
	sink.Emit(EventMediaPickRequired, MediaPickRequiredDTO{URL: url})
}
