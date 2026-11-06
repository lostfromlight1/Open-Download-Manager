package main

import "github.com/wailsapp/wails/v3/pkg/application"

// wailsSink adapts the Wails application event bus to EventSink.
type wailsSink struct {
	app *application.App
}

func (w wailsSink) Emit(name string, data any) {
	if w.app == nil {
		return
	}
	w.app.Event.Emit(name, data)
}
