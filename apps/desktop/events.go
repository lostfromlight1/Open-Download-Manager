package main

// Wails custom event names for the React frontend.
const (
	EventDownloadProgress = "odm:download-progress"
	EventDownloadChanged  = "odm:download-changed"
	EventDecisionRequired = "odm:decision-required"
	EventServiceError     = "odm:service-error"
)

// EventSink publishes desktop events. Production uses Wails; tests use a recorder.
type EventSink interface {
	Emit(name string, data any)
}
