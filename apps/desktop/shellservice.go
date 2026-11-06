package main

// ShellService exposes product metadata and window/tray actions to React.
type ShellService struct {
	downloads *DownloadService
	tray      *trayController
}

// NewShellService constructs a ShellService.
func NewShellService(downloads *DownloadService) *ShellService {
	return &ShellService{downloads: downloads}
}

func (s *ShellService) bindTray(tray *trayController) {
	s.tray = tray
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

// ShowMainWindow restores the main window from the tray.
func (s *ShellService) ShowMainWindow() {
	if s.tray != nil {
		s.tray.ShowMainWindow()
	}
}

// Quit exits the application fully (stops the download queue).
func (s *ShellService) Quit() {
	if s.tray != nil {
		s.tray.Quit()
	}
}
