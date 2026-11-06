// Package main is the Wails desktop entry point.
//
// Even though this is also package "main", it is a *different program* from
// cmd/odm because it lives in a different directory. `go build ./apps/desktop`
// and `go build ./cmd/odm` produce two separate executables.
package main

import (
	// embed is a special package that lets us include files inside the binary.
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// //go:embed is a compiler directive (note: no space between // and go).
// It packs everything under frontend/dist into the variable below at build time.
// That means users do not need a separate web folder next to the .exe.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// application.New creates the Wails app object (windowing + IPC + assets).
	app := application.New(application.Options{
		Name:        "ODM",
		Description: "Open Download Manager desktop shell",
		// Services are Go structs whose exported methods become callable from JS.
		Services: []application.Service{
			application.NewService(NewShellService()),
		},
		Assets: application.AssetOptions{
			// Serve the embedded frontend files to the webview.
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Open Download Manager",
		Width:            960,
		Height:           640,
		BackgroundColour: application.NewRGB(255, 255, 255),
		URL:              "/",
	})

	// Run blocks until the application exits.
	if err := app.Run(); err != nil {
		log.Fatal(err) // log and exit with non-zero status
	}
}
