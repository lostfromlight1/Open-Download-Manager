package main

import (
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const autostartIdentifier = "OpenDownloadManager"

// syncLaunchAtLogin enables or disables OS login autostart via Wails.
// When enabled, the app is registered with --hidden so it starts to the tray.
func syncLaunchAtLogin(app *application.App, enabled bool) {
	if app == nil || app.Autostart == nil {
		return
	}
	if enabled {
		if err := app.Autostart.EnableWithOptions(application.AutostartOptions{
			Identifier: autostartIdentifier,
			Arguments:  []string{"--hidden"},
		}); err != nil {
			log.Printf("launch-at-login enable: %v", err)
		}
		return
	}
	if err := app.Autostart.Disable(); err != nil {
		log.Printf("launch-at-login disable: %v", err)
	}
}
