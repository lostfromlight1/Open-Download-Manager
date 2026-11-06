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
	// Create or re-show the tray when the setting is turned on at runtime.
	t.ensureTray()
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
		// DisableQuitOnLastWindowClosed stays true so tray mode can hide the
		// last window. When the user really closes (setting off or Quit), exit.
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
