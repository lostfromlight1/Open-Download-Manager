package main

// Wails custom event names for the React frontend.
const (
	EventDownloadProgress  = "odm:download-progress"
	EventDownloadChanged   = "odm:download-changed"
	EventDecisionRequired  = "odm:decision-required"
	EventMediaPickRequired = "odm:media-pick-required"
	EventServiceError      = "odm:service-error"
)

// MediaPickRequiredDTO asks the UI to probe a YouTube URL and open the format picker.
type MediaPickRequiredDTO struct {
	URL string `json:"url"`
}

// EventSink publishes desktop events. Production uses Wails; tests use a recorder.
type EventSink interface {
	Emit(name string, data any)
}
