// Package main is the Wails desktop entry point.
//
// Even though this is also package "main", it is a *different program* from
// cmd/odm because it lives in a different directory. `go build ./apps/desktop`
// and `go build ./cmd/odm` produce two separate executables.
package main

import (
	"context"
	"embed"
	"log"
	"os"

	"github.com/lostfromlight1/Open-Download-Manager/internal/appdir"
	"github.com/lostfromlight1/Open-Download-Manager/internal/manager"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	startHidden := hasArg("--hidden")

	downloadSvc := NewDownloadService()
	shellSvc := NewShellService(downloadSvc)

	app := application.New(application.Options{
		Name:        "ODM",
		Description: "Open Download Manager desktop shell",
		Services: []application.Service{
			application.NewService(shellSvc),
			application.NewService(downloadSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		// Keep the process alive when the window is hidden to the tray.
		// Real exit is tray Quit / close-to-tray off (see trayController).
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
		},
		Linux: application.LinuxOptions{
			DisableQuitOnLastWindowClosed: true,
		},
	})

	sink := wailsSink{app: app}
	decisions := NewInteractiveDecisions(sink)

	queueCtx, queueCancel := context.WithCancel(context.Background())
	defer queueCancel()

	closeToTray := true
	launchAtLogin := false
	dataDir, err := appdir.Resolve("")
	if err != nil {
		downloadSvc.markFailed(err)
		log.Printf("resolve data dir: %v", err)
	} else {
		store, err := storage.Open(appdir.DBPath(dataDir))
		if err != nil {
			downloadSvc.markFailed(err)
			log.Printf("open database: %v", err)
		} else {
			cfg, cfgErr := store.GetSettings(context.Background())
			if cfgErr == nil {
				closeToTray = cfg.CloseToTray
				launchAtLogin = cfg.LaunchAtLogin
			}
			mgr := manager.New(store, decisions)
			downloadSvc.bind(mgr, store, decisions, sink, dataDir)
			go func() {
				_ = mgr.RunQueue(queueCtx, true)
			}()
			defer func() {
				queueCancel()
				_ = store.Close()
			}()
		}
	}

	tray := newTrayController(app, closeToTray)
	shellSvc.bindTray(tray)
	downloadSvc.setTrayController(tray)
	showUI := func() {
		tray.ShowMainWindow()
	}
	decisions.SetNeedUI(showUI)
	downloadSvc.setNeedUI(showUI)
	// Re-apply autostart on each launch so the OS entry stays in sync.
	syncLaunchAtLogin(app, launchAtLogin)

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Open Download Manager",
		Width:            1040,
		Height:           720,
		MinWidth:         800,
		MinHeight:        560,
		BackgroundColour: application.NewRGB(247, 248, 250),
		URL:              "/",
	})
	tray.attachWindow(win)
	if startHidden {
		tray.StartHidden()
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
	queueCancel()
}

func hasArg(name string) bool {
	for _, a := range os.Args[1:] {
		if a == name {
			return true
		}
	}
	return false
}
