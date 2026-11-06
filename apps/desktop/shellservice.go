package main

// ShellService exposes product metadata to the React frontend.
type ShellService struct {
	downloads *DownloadService
}

// NewShellService constructs a ShellService.
func NewShellService(downloads *DownloadService) *ShellService {
	return &ShellService{downloads: downloads}
}

// ProductName returns the user-facing product name.
func (s *ShellService) ProductName() string {
	return "Open Download Manager"
}

// DownloadsAvailable reports whether download management is ready.
func (s *ShellService) DownloadsAvailable() bool {
	if s.downloads == nil {
		return false
	}
	return s.downloads.Ready()
}

// StatusMessage is a short human-readable shell status for the UI.
func (s *ShellService) StatusMessage() string {
	if s.downloads == nil {
		return "Download manager not initialized"
	}
	return s.downloads.StatusMessage()
}
