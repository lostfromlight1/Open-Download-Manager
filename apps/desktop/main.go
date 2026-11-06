// Package main is the Wails desktop entry point.
//
// Even though this is also package "main", it is a *different program* from
// cmd/odm because it lives in a different directory. `go build ./apps/desktop`
// and `go build ./cmd/odm` produce two separate executables.
package main

import (
	// embed is a special package that lets us include files inside the binary.
	"context"
	"embed"
	"log"

	"github.com/lostfromlight1/Open-Download-Manager/internal/appdir"
	"github.com/lostfromlight1/Open-Download-Manager/internal/manager"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// //go:embed is a compiler directive (note: no space between // and go).
// It packs everything under frontend/dist into the variable below at build time.
// That means users do not need a separate web folder next to the .exe.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	downloadSvc := NewDownloadService()
	shellSvc := NewShellService(downloadSvc)

	// application.New creates the Wails app object (windowing + IPC + assets).
	app := application.New(application.Options{
		Name:        "ODM",
		Description: "Open Download Manager desktop shell",
		// Services are Go structs whose exported methods become callable from JS.
		Services: []application.Service{
			application.NewService(shellSvc),
			application.NewService(downloadSvc),
		},
		Assets: application.AssetOptions{
			// Serve the embedded frontend files to the webview.
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	sink := wailsSink{app: app}
	decisions := NewInteractiveDecisions(sink)

	queueCtx, queueCancel := context.WithCancel(context.Background())
	defer queueCancel()

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
			mgr := manager.New(store, decisions)
			downloadSvc.bind(mgr, store, decisions, sink, dataDir)
			go func() {
				_ = mgr.RunQueue(queueCtx, true)
			}()
			// Close DB after queue stops (when app exits and defer cancel runs
			// before process end — best-effort).
			defer func() {
				queueCancel()
				_ = store.Close()
			}()
		}
	}

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Open Download Manager",
		Width:            1040,
		Height:           720,
		BackgroundColour: application.NewRGB(247, 248, 250),
		URL:              "/",
	})

	// Run blocks until the application exits.
	if err := app.Run(); err != nil {
		log.Fatal(err) // log and exit with non-zero status
	}
	queueCancel()
}
