package main

import (
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// ShouldHideOnClose reports whether a window-close should hide to tray.
func ShouldHideOnClose(closeToTray, forceQuit bool) bool {
	return closeToTray && !forceQuit
}

// trayController owns tray/window lifecycle for close-to-tray behavior.
type trayController struct {
	mu          sync.Mutex
	app         *application.App
	window      *application.WebviewWindow
	tray        *application.SystemTray
	closeToTray bool
	forceQuit   bool
}

func newTrayController(app *application.App, closeToTray bool) *trayController {
	return &trayController{app: app, closeToTray: closeToTray}
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

func (t *trayController) ShowMainWindow() {
	t.mu.Lock()
	win := t.window
	t.mu.Unlock()
	if win != nil {
		win.Show().Focus()
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

// ensureTray creates the system tray once (Show / Quit menu + click to restore).
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
