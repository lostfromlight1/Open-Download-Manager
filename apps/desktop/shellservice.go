package main

// ShellService exposes a minimal desktop status API to the React frontend.
//
// Wails can call exported methods on this struct from TypeScript.
// Phase 1 added the Go download engine for the CLI; the desktop UI still does
// not wire download actions yet, so DownloadsAvailable stays false.
type ShellService struct{}

// NewShellService constructs a ShellService.
//
// Returning a pointer (*ShellService) is common so methods share one instance.
func NewShellService() *ShellService {
	return &ShellService{}
}

// ProductName returns the user-facing product name.
func (s *ShellService) ProductName() string {
	return "Open Download Manager"
}

// DownloadsAvailable reports whether download management is implemented in the UI.
func (s *ShellService) DownloadsAvailable() bool {
	return false
}

// StatusMessage is a short human-readable shell status for the UI.
func (s *ShellService) StatusMessage() string {
	return "Download management is not available in this build."
}
