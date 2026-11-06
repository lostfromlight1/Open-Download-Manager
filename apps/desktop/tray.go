package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// ShouldHideOnClose reports whether a window-close should hide to tray.
func ShouldHideOnClose(closeToTray, forceQuit bool) bool {
	return closeToTray && !forceQuit
}

// FormatTrayTooltip builds the system-tray hover text from aggregate speed (bytes/sec).
func FormatTrayTooltip(bytesPerSec float64) string {
	if bytesPerSec <= 0 {
		return "Open Download Manager"
	}
	return fmt.Sprintf("ODM — %s/s", formatByteRate(bytesPerSec))
}

func formatByteRate(bps float64) string {
	const (
		kb = 1024
		mb = 1024 * 1024
		gb = 1024 * 1024 * 1024
	)
	switch {
	case bps >= gb:
		return fmt.Sprintf("%.1f GB", bps/gb)
	case bps >= mb:
		return fmt.Sprintf("%.1f MB", bps/mb)
	case bps >= kb:
		return fmt.Sprintf("%.1f KB", bps/kb)
	default:
		return fmt.Sprintf("%.0f B", bps)
	}
}

// trayTaskActions is implemented by DownloadService for Pause all / Resume paused.
type trayTaskActions interface {
	PauseAllActive()
	ResumeAllPaused()
}

// trayController owns tray/window lifecycle for close-to-tray behavior.
type trayController struct {
	mu          sync.Mutex
	app         *application.App
	window      *application.WebviewWindow
	tray        *application.SystemTray
	actions     trayTaskActions
	closeToTray bool
	forceQuit   bool
	lastTipAt   time.Time
	lastTip     string
}

func newTrayController(app *application.App, closeToTray bool) *trayController {
	return &trayController{app: app, closeToTray: closeToTray}
}

func (t *trayController) setActions(actions trayTaskActions) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.actions = actions
}

// SetCloseToTray updates the persisted preference's runtime effect.
// Disabling while hidden restores the window; enabling ensures the tray icon exists.
func (t *trayController) SetCloseToTray(enabled bool) {
	t.mu.Lock()
	t.closeToTray = enabled
	win := t.window
	t.mu.Unlock()
	if !enabled {
		if win != nil {
			win.Show().Focus()
		}
		return
	}
	t.ensureTray()
	t.mu.Lock()
	tray := t.tray
	t.mu.Unlock()
	if tray != nil {
		tray.Show()
	}
}

func (t *trayController) CloseToTrayEnabled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closeToTray
}

// SetLaunchAtLogin syncs the OS autostart registration with the settings toggle.
func (t *trayController) SetLaunchAtLogin(enabled bool) {
	t.mu.Lock()
	app := t.app
	t.mu.Unlock()
	syncLaunchAtLogin(app, enabled)
}

func (t *trayController) ShowMainWindow() {
	t.mu.Lock()
	win := t.window
	t.mu.Unlock()
	if win != nil {
		win.Show().Focus()
	}
}

// StartHidden ensures the tray exists and hides the main window (login autostart).
func (t *trayController) StartHidden() {
	t.ensureTray()
	t.mu.Lock()
	win := t.window
	t.mu.Unlock()
	if win != nil {
		win.Hide()
	}
}

func (t *trayController) Quit() {
	t.mu.Lock()
	t.forceQuit = true
	app := t.app
	t.mu.Unlock()
	if app != nil {
		app.Quit()
	}
}

// SetAggregateSpeed updates the tray tooltip (throttled to ~1s).
func (t *trayController) SetAggregateSpeed(bytesPerSec float64) {
	tip := FormatTrayTooltip(bytesPerSec)
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	if now.Sub(t.lastTipAt) < time.Second {
		return
	}
	t.lastTip = tip
	t.lastTipAt = now
	if t.tray != nil {
		t.tray.SetTooltip(tip)
	}
}

// attachWindow wires close-to-tray hide behavior and creates the tray when enabled.
func (t *trayController) attachWindow(win *application.WebviewWindow) {
	t.mu.Lock()
	t.window = win
	closeToTray := t.closeToTray
	t.mu.Unlock()

	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		t.mu.Lock()
		hide := ShouldHideOnClose(t.closeToTray, t.forceQuit)
		alreadyQuitting := t.forceQuit
		app := t.app
		t.mu.Unlock()
		if hide {
			win.Hide()
			e.Cancel()
			return
		}
		// App options keep DisableQuitOnLastWindowClosed true so tray mode can
		// hide the last window. When close-to-tray is off (or Quit was chosen),
		// we must Quit explicitly or the process would keep running headless.
		if !alreadyQuitting && app != nil {
			t.mu.Lock()
			t.forceQuit = true
			t.mu.Unlock()
			go app.Quit()
		}
	})

	if closeToTray {
		t.ensureTray()
	}
}

// ensureTray creates the system tray once (Show / pause-resume / Quit).
func (t *trayController) ensureTray() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.tray != nil || t.app == nil {
		return
	}
	tray := t.app.SystemTray.New()
	tray.SetIcon(trayIconPNG)
	tray.SetTooltip("Open Download Manager")
	tray.SetLabel("ODM")

	menu := t.app.NewMenu()
	menu.Add("Show Open Download Manager").OnClick(func(*application.Context) {
		t.ShowMainWindow()
	})
	menu.Add("Pause all active").OnClick(func(*application.Context) {
		t.mu.Lock()
		actions := t.actions
		t.mu.Unlock()
		if actions != nil {
			actions.PauseAllActive()
		}
	})
	menu.Add("Resume paused").OnClick(func(*application.Context) {
		t.mu.Lock()
		actions := t.actions
		t.mu.Unlock()
		if actions != nil {
			actions.ResumeAllPaused()
		}
	})
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(*application.Context) {
		t.Quit()
	})
	tray.SetMenu(menu)
	if t.window != nil {
		tray.AttachWindow(t.window)
	}
	tray.OnClick(func() {
		t.ShowMainWindow()
	})
	tray.Run()
	t.tray = tray
}
