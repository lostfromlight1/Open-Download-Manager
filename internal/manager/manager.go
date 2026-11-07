// Package manager owns task lifecycle, the download queue, and persistence
// orchestration. It calls internal/engine for byte transfer and must not import
// Wails — the CLI and a future desktop UI both use this package.
package manager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/engine"
	"github.com/lostfromlight1/Open-Download-Manager/internal/media"
	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
	"github.com/lostfromlight1/Open-Download-Manager/internal/task"
)

// AddOptions configures a newly enqueued download.
type AddOptions struct {
	URL         string
	Destination string // final file path (required for Phase 2 CLI)
	DisplayName string // optional; derived from destination when empty

	// Cookie / Referer are ephemeral request headers (browser handoff).
	// They are kept in memory only for this process — never written to SQLite.
	Cookie  string
	Referer string
}

// ephemeralAuth is per-task Cookie/Referer held only until a terminal state.
type ephemeralAuth struct {
	Cookie  string
	Referer string
}

// Listener receives optional UI-facing updates (desktop). CLI leaves it nil.
// Implementations must not block for long and must not call back into Manager
// methods that take m.mu (to avoid deadlocks).
type Listener interface {
	OnProgress(taskID string, ev engine.ProgressEvent)
	OnChanged(taskID string)
}

// Manager coordinates queued downloads against a Store.
type Manager struct {
	store     *storage.Store
	decisions DecisionHandler
	dataDir   string // app data dir for managed yt-dlp/ffmpeg

	mu       sync.Mutex
	running  map[string]context.CancelFunc // taskID → cancel for in-flight work
	auth     map[string]ephemeralAuth      // taskID → Cookie/Referer (memory only)
	active   int                           // count of workers holding a slot
	wake     chan struct{}                 // signal scheduler to look for work
	closed   bool
	listener Listener
}

// SetDataDir sets the ODM data directory (tools live under dataDir/tools).
func (m *Manager) SetDataDir(dir string) {
	m.mu.Lock()
	m.dataDir = dir
	m.mu.Unlock()
}

// New constructs a Manager. decisions may be nil (defaults to keep + abort).
func New(store *storage.Store, decisions DecisionHandler) *Manager {
	if decisions == nil {
		decisions = StaticDecisions{Cancel: FileKeep, Remove: FileKeep, Mismatch: MismatchAbort}
	}
	return &Manager{
		store:     store,
		decisions: decisions,
		running:   make(map[string]context.CancelFunc),
		auth:      make(map[string]ephemeralAuth),
		wake:      make(chan struct{}, 1),
	}
}

// SetListener attaches an optional progress/change listener (desktop UI).
func (m *Manager) SetListener(l Listener) {
	m.mu.Lock()
	m.listener = l
	m.mu.Unlock()
}

func (m *Manager) notifyChanged(taskID string) {
	m.mu.Lock()
	l := m.listener
	m.mu.Unlock()
	if l != nil {
		l.OnChanged(taskID)
	}
}

func (m *Manager) notifyProgress(taskID string, ev engine.ProgressEvent) {
	m.mu.Lock()
	l := m.listener
	m.mu.Unlock()
	if l != nil {
		l.OnProgress(taskID, ev)
	}
}

// AddMediaOptions configures a YouTube / media download task.
type AddMediaOptions struct {
	WebpageURL  string
	FormatID    string
	DestDir     string
	DisplayName string
	PlaylistID  string
}

// Add inserts a queued task and nudges the scheduler.
func (m *Manager) Add(ctx context.Context, opts AddOptions) (task.Task, error) {
	if media.IsYouTubeURL(opts.URL) {
		return task.Task{}, fmt.Errorf("%w: YouTube URLs require the media download flow (enable in Settings)", engine.ErrInvalidURL)
	}
	if err := validateHTTPURL(opts.URL); err != nil {
		return task.Task{}, err
	}

	dest := filepath.Clean(opts.Destination)
	if dest == "" || dest == "." {
		return task.Task{}, fmt.Errorf("%w: destination required", engine.ErrInvalidDestination)
	}
	abs, err := filepath.Abs(dest)
	if err != nil {
		return task.Task{}, fmt.Errorf("%w: %v", engine.ErrInvalidDestination, err)
	}

	name := opts.DisplayName
	if name == "" {
		name = filepath.Base(abs)
	}

	now := time.Now().UTC()
	t := task.Task{
		ID:              task.NewID(),
		URL:             opts.URL,
		DisplayName:     name,
		DestinationPath: abs,
		State:           task.StateQueued,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertTask(ctx, t); err != nil {
		return task.Task{}, err
	}
	if opts.Cookie != "" || opts.Referer != "" {
		m.mu.Lock()
		m.auth[t.ID] = ephemeralAuth{Cookie: opts.Cookie, Referer: opts.Referer}
		m.mu.Unlock()
	}
	m.notifyChanged(t.ID)
	m.nudge()
	return t, nil
}

// AddMedia enqueues a YouTube download (requires media.enabled at execute time).
func (m *Manager) AddMedia(ctx context.Context, opts AddMediaOptions) (task.Task, error) {
	pageURL, err := media.ValidateYouTubeURL(opts.WebpageURL)
	if err != nil {
		return task.Task{}, err
	}
	dir := filepath.Clean(strings.TrimSpace(opts.DestDir))
	if dir == "" || dir == "." {
		return task.Task{}, fmt.Errorf("%w: destination directory required", engine.ErrInvalidDestination)
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return task.Task{}, fmt.Errorf("%w: %v", engine.ErrInvalidDestination, err)
	}
	name := strings.TrimSpace(opts.DisplayName)
	if name == "" {
		name = "youtube-video"
	}
	// Provisional path; executeMedia updates DestinationPath to the real file.
	abs := filepath.Join(absDir, name+".mp4")

	format := strings.TrimSpace(opts.FormatID)
	if format == "" {
		format = "best"
	}

	now := time.Now().UTC()
	t := task.Task{
		ID:              task.NewID(),
		URL:             pageURL,
		DisplayName:     name,
		DestinationPath: abs,
		State:           task.StateQueued,
		CreatedAt:       now,
		UpdatedAt:       now,
		MediaSource:     "youtube",
		MediaFormatID:   format,
		MediaWebpageURL: pageURL,
		MediaPlaylistID: strings.TrimSpace(opts.PlaylistID),
	}
	if err := m.store.InsertTask(ctx, t); err != nil {
		return task.Task{}, err
	}
	m.notifyChanged(t.ID)
	m.nudge()
	return t, nil
}

func (m *Manager) takeAuthHeaders(taskID string) http.Header {
	m.mu.Lock()
	auth, ok := m.auth[taskID]
	m.mu.Unlock()
	if !ok {
		return nil
	}
	h := make(http.Header)
	if auth.Cookie != "" {
		h.Set("Cookie", auth.Cookie)
	}
	if auth.Referer != "" {
		h.Set("Referer", auth.Referer)
	}
	return h
}

func (m *Manager) clearAuth(taskID string) {
	m.mu.Lock()
	delete(m.auth, taskID)
	m.mu.Unlock()
}

// List returns all tasks.
func (m *Manager) List(ctx context.Context) ([]task.Task, error) {
	return m.store.ListTasks(ctx)
}

// Get returns one task.
func (m *Manager) Get(ctx context.Context, id string) (task.Task, error) {
	return m.store.GetTask(ctx, id)
}

// Settings loads persisted settings.
func (m *Manager) Settings(ctx context.Context) (storage.Settings, error) {
	return m.store.GetSettings(ctx)
}

// SaveSettings persists settings.
func (m *Manager) SaveSettings(ctx context.Context, cfg storage.Settings) error {
	return m.store.SaveSettings(ctx, cfg)
}

// RunQueue processes queued work until ctx is cancelled.
//
// If wait is false, returns once nothing is queued and no workers are active.
// If wait is true, blocks until ctx ends (used by `odm queue run`).
func (m *Manager) RunQueue(ctx context.Context, wait bool) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		started, err := m.scheduleOnce(ctx)
		if err != nil {
			return err
		}
		if started > 0 {
			continue
		}

		m.mu.Lock()
		active := m.active
		m.mu.Unlock()
		if active == 0 && !wait {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.wake:
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// Pause cancels an in-flight task and marks it paused (keeps files).
func (m *Manager) Pause(ctx context.Context, id string) error {
	t, err := m.store.GetTask(ctx, id)
	if err != nil {
		return err
	}

	m.cancelRunning(id)

	next, err := task.Transition(t.State, task.StatePaused)
	if err != nil {
		return err
	}
	t.State = next
	t.UpdatedAt = time.Now().UTC()
	if err := m.store.UpdateTask(ctx, t); err != nil {
		return err
	}
	m.notifyChanged(id)
	return nil
}

// Resume moves a paused/failed/cancelled task back to queued.
func (m *Manager) Resume(ctx context.Context, id string) error {
	t, err := m.store.GetTask(ctx, id)
	if err != nil {
		return err
	}
	switch t.State {
	case task.StatePaused, task.StateFailed, task.StateCancelled:
		t.State = task.StateQueued
		t.ErrorCode = ""
		t.ErrorMessage = ""
		t.UpdatedAt = time.Now().UTC()
		if err := m.store.UpdateTask(ctx, t); err != nil {
			return err
		}
		m.notifyChanged(id)
		m.nudge()
		return nil
	default:
		return fmt.Errorf("%w: cannot resume from %s", task.ErrInvalidTransition, t.State)
	}
}

// Cancel stops a task. files decides whether to delete .part / destination.
func (m *Manager) Cancel(ctx context.Context, id string, files FileAction) error {
	if files == "" {
		var err error
		files, err = m.decisions.OnCancelFiles(id)
		if err != nil {
			return err
		}
	}

	t, err := m.store.GetTask(ctx, id)
	if err != nil {
		return err
	}
	m.cancelRunning(id)

	next, err := task.Transition(t.State, task.StateCancelled)
	if err != nil {
		return err
	}
	t.State = next
	t.UpdatedAt = time.Now().UTC()
	if err := m.store.UpdateTask(ctx, t); err != nil {
		return err
	}
	if files == FileDelete {
		_ = os.Remove(t.PartPath())
		_ = os.Remove(t.DestinationPath)
	}
	m.clearAuth(id)
	m.notifyChanged(id)
	m.nudge()
	return nil
}

// Remove deletes task metadata. files decides whether to delete files too.
func (m *Manager) Remove(ctx context.Context, id string, files FileAction) error {
	if files == "" {
		var err error
		files, err = m.decisions.OnRemoveFiles(id)
		if err != nil {
			return err
		}
	}

	t, err := m.store.GetTask(ctx, id)
	if err != nil {
		return err
	}
	m.cancelRunning(id)

	if files == FileDelete {
		_ = os.Remove(t.PartPath())
		_ = os.Remove(t.DestinationPath)
	}
	if err := m.store.DeleteTask(ctx, id); err != nil {
		return err
	}
	m.clearAuth(id)
	m.notifyChanged(id)
	m.nudge()
	return nil
}

func (m *Manager) scheduleOnce(ctx context.Context) (int, error) {
	cfg, err := m.store.GetSettings(ctx)
	if err != nil {
		return 0, err
	}

	m.mu.Lock()
	slots := cfg.MaxActiveDownloads - m.active
	m.mu.Unlock()
	if slots <= 0 {
		return 0, nil
	}

	candidates, err := m.store.ListByStates(ctx, task.StateQueued)
	if err != nil {
		return 0, err
	}

	started := 0
	for _, t := range candidates {
		if started >= slots {
			break
		}
		if err := m.startTask(ctx, t, cfg); err != nil {
			return started, err
		}
		started++
	}
	return started, nil
}

func (m *Manager) startTask(parent context.Context, t task.Task, cfg storage.Settings) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return fmt.Errorf("manager closed")
	}
	if _, alive := m.running[t.ID]; alive {
		m.mu.Unlock()
		return nil
	}
	runCtx, cancel := context.WithCancel(parent)
	m.running[t.ID] = cancel
	m.active++
	m.mu.Unlock()

	// go starts a worker goroutine. The task runs concurrently with the scheduler.
	go func() {
		defer func() {
			cancel()
			m.mu.Lock()
			delete(m.running, t.ID)
			m.active--
			m.mu.Unlock()
			m.nudge()
		}()
		_ = m.execute(runCtx, t.ID, cfg)
	}()
	return nil
}

func (m *Manager) execute(ctx context.Context, id string, cfg storage.Settings) error {
	t, err := m.store.GetTask(ctx, id)
	if err != nil {
		return err
	}

	t.State = task.StateInspecting
	t.UpdatedAt = time.Now().UTC()
	if err := m.store.UpdateTask(ctx, t); err != nil {
		return err
	}
	m.notifyChanged(id)

	if t.MediaSource == "youtube" {
		return m.executeMedia(ctx, t, cfg)
	}

	forceRestart, err := m.handleResumeValidators(ctx, &t, cfg)
	if err != nil {
		return m.failTask(ctx, t, "resume_mismatch", err.Error())
	}

	t.State = task.StateDownloading
	t.UpdatedAt = time.Now().UTC()
	if err := m.store.UpdateTask(ctx, t); err != nil {
		return err
	}
	m.notifyChanged(id)

	proxy := netx.ProxyConfig{Mode: cfg.ProxyMode, URL: cfg.ProxyURL}
	connections := cfg.MaxConnectionsPerDownload
	if connections < 1 {
		connections = 1
	}
	result, err := engine.Download(ctx, engine.Options{
		URL:            t.URL,
		Destination:    t.DestinationPath,
		ForceRestart:   forceRestart,
		Connections:    connections,
		Proxy:          proxy,
		MaxRetries:     cfg.MaxRetries,
		ConnectTimeout: time.Duration(cfg.ConnectTimeoutSec) * time.Second,
		StallTimeout:   time.Duration(cfg.StallTimeoutSec) * time.Second,
		ExtraHeaders:   m.takeAuthHeaders(id),
		OnProgress: func(ev engine.ProgressEvent) {
			cur, gerr := m.store.GetTask(context.Background(), id)
			if gerr != nil {
				return
			}
			cur.DownloadedBytes = ev.Downloaded
			if ev.Total > 0 {
				total := ev.Total
				cur.TotalBytes = &total
			}
			cur.UpdatedAt = time.Now().UTC()
			_ = m.store.UpdateTask(context.Background(), cur)
			m.notifyProgress(id, ev)
		},
	})
	if err != nil {
		if errors.Is(err, engine.ErrCancelled) || errors.Is(err, context.Canceled) {
			cur, gerr := m.store.GetTask(context.Background(), id)
			if gerr == nil && task.IsActive(cur.State) {
				// If Cancel already set cancelled, do not overwrite with paused.
				if cur.State != task.StateCancelled {
					cur.State = task.StatePaused
					cur.UpdatedAt = time.Now().UTC()
					_ = m.store.UpdateTask(context.Background(), cur)
					m.notifyChanged(id)
				}
			}
			return err
		}
		return m.failTask(ctx, t, "download_failed", err.Error())
	}

	t, err = m.store.GetTask(ctx, id)
	if err != nil {
		return err
	}
	t.State = task.StateFinalizing
	t.UpdatedAt = time.Now().UTC()
	_ = m.store.UpdateTask(ctx, t)
	m.notifyChanged(id)

	t.State = task.StateCompleted
	t.DownloadedBytes = result.BytesWritten
	total := result.BytesWritten
	t.TotalBytes = &total
	t.ETag = result.ETag
	t.LastModified = result.LastModified
	t.ErrorCode = ""
	t.ErrorMessage = ""
	t.UpdatedAt = time.Now().UTC()
	if err := m.store.UpdateTask(ctx, t); err != nil {
		return err
	}
	m.clearAuth(id)
	m.notifyChanged(id)
	return nil
}

func (m *Manager) executeMedia(ctx context.Context, t task.Task, cfg storage.Settings) error {
	id := t.ID
	if !cfg.MediaEnabled {
		return m.failTask(ctx, t, "media_disabled", "enable YouTube downloads in Settings")
	}

	m.mu.Lock()
	dataDir := m.dataDir
	m.mu.Unlock()
	tools := media.ResolveTools(dataDir, cfg.MediaYtDlpPath, cfg.MediaFFmpegPath)
	if !tools.YtDlpReady() {
		return m.failTask(ctx, t, "media_tools", "yt-dlp helper is missing; re-enable media in Settings to install")
	}
	formatSel := media.ResolveFormatSelector(t.MediaFormatID)
	if strings.Contains(formatSel, "+") && !tools.FFmpegReady() {
		return m.failTask(ctx, t, "media_tools", "ffmpeg is required to merge video+audio; open Settings → Retry helper install")
	}

	t.State = task.StateDownloading
	t.UpdatedAt = time.Now().UTC()
	if err := m.store.UpdateTask(ctx, t); err != nil {
		return err
	}
	m.notifyChanged(id)

	destDir := filepath.Dir(t.DestinationPath)
	final, err := media.Download(ctx, media.DownloadOptions{
		WebpageURL: t.MediaWebpageURL,
		FormatID:   t.MediaFormatID,
		DestDir:    destDir,
		OutputName: "",
		Tools:      tools,
		OnProgress: func(ev media.ProgressEvent) {
			cur, gerr := m.store.GetTask(context.Background(), id)
			if gerr != nil {
				return
			}
			cur.DownloadedBytes = ev.Downloaded
			if ev.Total > 0 {
				total := ev.Total
				cur.TotalBytes = &total
			}
			cur.UpdatedAt = time.Now().UTC()
			_ = m.store.UpdateTask(context.Background(), cur)
			m.notifyProgress(id, engine.ProgressEvent{
				Downloaded:  ev.Downloaded,
				Total:       ev.Total,
				BytesPerSec: ev.BytesPerSec,
			})
		},
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "cancelled") {
			cur, gerr := m.store.GetTask(context.Background(), id)
			if gerr == nil && task.IsActive(cur.State) && cur.State != task.StateCancelled {
				cur.State = task.StatePaused
				cur.UpdatedAt = time.Now().UTC()
				_ = m.store.UpdateTask(context.Background(), cur)
				m.notifyChanged(id)
			}
			return err
		}
		return m.failTask(ctx, t, "media_failed", err.Error())
	}

	st, _ := os.Stat(final)
	var size int64
	if st != nil {
		size = st.Size()
	}
	t, err = m.store.GetTask(ctx, id)
	if err != nil {
		return err
	}
	t.State = task.StateCompleted
	t.DestinationPath = final
	t.DisplayName = filepath.Base(final)
	t.DownloadedBytes = size
	t.TotalBytes = &size
	t.ErrorCode = ""
	t.ErrorMessage = ""
	t.UpdatedAt = time.Now().UTC()
	if err := m.store.UpdateTask(ctx, t); err != nil {
		return err
	}
	m.notifyChanged(id)
	return nil
}

func (m *Manager) handleResumeValidators(ctx context.Context, t *task.Task, cfg storage.Settings) (forceRestart bool, err error) {
	if _, statErr := os.Stat(t.PartPath()); statErr != nil {
		return false, nil
	}
	if t.ETag == "" && t.LastModified == "" {
		return false, nil
	}

	remoteETag, remoteLM, err := probeValidators(ctx, t.URL, cfg)
	if err != nil {
		return false, nil
	}

	etagChanged := t.ETag != "" && remoteETag != "" && t.ETag != remoteETag
	lmChanged := t.LastModified != "" && remoteLM != "" && t.LastModified != remoteLM
	if !etagChanged && !lmChanged {
		return false, nil
	}

	action, err := m.decisions.OnResumeMismatch(ctx, ValidatorMismatch{
		TaskID:          t.ID,
		DestinationPath: t.DestinationPath,
		SavedETag:       t.ETag,
		SavedModified:   t.LastModified,
		RemoteETag:      remoteETag,
		RemoteModified:  remoteLM,
	})
	if err != nil {
		return false, err
	}
	switch action {
	case MismatchRestart:
		t.ETag = ""
		t.LastModified = ""
		t.DownloadedBytes = 0
		t.UpdatedAt = time.Now().UTC()
		_ = m.store.UpdateTask(ctx, *t)
		return true, nil
	case MismatchAbort:
		return false, fmt.Errorf("%w: remote file validators changed", task.ErrDecisionRequired)
	default:
		return false, fmt.Errorf("unknown mismatch action %q", action)
	}
}

func (m *Manager) failTask(ctx context.Context, t task.Task, code, message string) error {
	cur, err := m.store.GetTask(ctx, t.ID)
	if err != nil {
		cur = t
	}
	cur.State = task.StateFailed
	cur.ErrorCode = code
	cur.ErrorMessage = message
	cur.RetryCount++
	cur.UpdatedAt = time.Now().UTC()
	_ = m.store.UpdateTask(ctx, cur)
	m.clearAuth(cur.ID)
	m.notifyChanged(cur.ID)
	return fmt.Errorf("%s: %s", code, message)
}

func (m *Manager) cancelRunning(id string) {
	m.mu.Lock()
	if cancel, ok := m.running[id]; ok {
		cancel()
	}
	m.mu.Unlock()
}

func (m *Manager) nudge() {
	// Non-blocking send: if a wake signal is already pending, skip.
	select {
	case m.wake <- struct{}{}:
	default:
	}
}
