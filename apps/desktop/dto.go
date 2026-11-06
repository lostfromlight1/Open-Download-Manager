package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
	"github.com/lostfromlight1/Open-Download-Manager/internal/task"
)

// DownloadDTO is the frontend-safe view of a task.
// Percent / ETA are computed in React from these raw fields.
type DownloadDTO struct {
	ID              string   `json:"id"`
	URL             string   `json:"url"`
	FileName        string   `json:"fileName"`
	DestinationPath string   `json:"destinationPath"`
	Status          string   `json:"status"`
	DownloadedBytes int64    `json:"downloadedBytes"`
	TotalBytes      *int64   `json:"totalBytes"`
	BytesPerSecond  *float64 `json:"bytesPerSecond"`
	ErrorCode       string   `json:"errorCode"`
	ErrorMessage    string   `json:"errorMessage"`
	CreatedAt       string   `json:"createdAt"`
	UpdatedAt       string   `json:"updatedAt"`
}

// AddDownloadRequest is the add-download form payload from React.
type AddDownloadRequest struct {
	URL         string `json:"url"`
	Destination string `json:"destination"`
}

// SettingsDTO is the frontend settings form.
type SettingsDTO struct {
	DownloadDirectory         string `json:"downloadDirectory"`
	MaxActiveDownloads        int    `json:"maxActiveDownloads"`
	MaxConnectionsPerDownload int    `json:"maxConnectionsPerDownload"`
	ProxyMode                 string `json:"proxyMode"`
	ProxyURL                  string `json:"proxyURL"`
	Theme                     string `json:"theme"`            // system | light | dark
	NotifyOnComplete          bool   `json:"notifyOnComplete"` // opt-in completion toasts
	BrowserIntegrationEnabled bool   `json:"browserIntegrationEnabled"`
	ChromeExtensionID         string `json:"chromeExtensionId"`
	BrowserHostRegistered     bool   `json:"browserHostRegistered"` // status only; not persisted
}

// ProgressEventDTO is emitted on odm:download-progress.
type ProgressEventDTO struct {
	TaskID          string   `json:"taskId"`
	DownloadedBytes int64    `json:"downloadedBytes"`
	TotalBytes      *int64   `json:"totalBytes"`
	BytesPerSecond  *float64 `json:"bytesPerSecond"`
	Status          string   `json:"status"`
}

// ChangedEventDTO is emitted on odm:download-changed.
type ChangedEventDTO struct {
	TaskID   string       `json:"taskId"`
	Download *DownloadDTO `json:"download"` // nil when removed
}

// DecisionRequiredDTO is emitted on odm:decision-required.
type DecisionRequiredDTO struct {
	DecisionID      string `json:"decisionId"`
	TaskID          string `json:"taskId"`
	DestinationPath string `json:"destinationPath"`
	SavedETag       string `json:"savedETag"`
	SavedModified   string `json:"savedModified"`
	RemoteETag      string `json:"remoteETag"`
	RemoteModified  string `json:"remoteModified"`
	Kind            string `json:"kind"` // "resume_mismatch"
}

func taskToDTO(t task.Task) DownloadDTO {
	return DownloadDTO{
		ID:              t.ID,
		URL:             t.URL,
		FileName:        t.DisplayName,
		DestinationPath: t.DestinationPath,
		Status:          string(t.State),
		DownloadedBytes: t.DownloadedBytes,
		TotalBytes:      t.TotalBytes,
		ErrorCode:       t.ErrorCode,
		ErrorMessage:    t.ErrorMessage,
		CreatedAt:       t.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:       t.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func settingsToDTO(s storage.Settings) SettingsDTO {
	theme := s.Theme
	if theme == "" {
		theme = "system"
	}
	return SettingsDTO{
		DownloadDirectory:         s.DownloadDirectory,
		MaxActiveDownloads:        s.MaxActiveDownloads,
		MaxConnectionsPerDownload: s.MaxConnectionsPerDownload,
		ProxyMode:                 string(s.ProxyMode),
		ProxyURL:                  s.ProxyURL,
		Theme:                     theme,
		NotifyOnComplete:          s.NotifyOnComplete,
		BrowserIntegrationEnabled: s.BrowserIntegrationEnabled,
		ChromeExtensionID:         s.ChromeExtensionID,
	}
}

func settingsFromDTO(d SettingsDTO) (storage.Settings, error) {
	cfg := storage.DefaultSettings()
	cfg.DownloadDirectory = d.DownloadDirectory
	cfg.MaxActiveDownloads = d.MaxActiveDownloads
	cfg.MaxConnectionsPerDownload = d.MaxConnectionsPerDownload
	cfg.ProxyURL = d.ProxyURL
	cfg.Theme = d.Theme
	cfg.NotifyOnComplete = d.NotifyOnComplete
	cfg.BrowserIntegrationEnabled = d.BrowserIntegrationEnabled
	cfg.ChromeExtensionID = strings.TrimSpace(d.ChromeExtensionID)

	switch netx.ProxyMode(d.ProxyMode) {
	case netx.ProxySystem, "":
		cfg.ProxyMode = netx.ProxySystem
	case netx.ProxyOff:
		cfg.ProxyMode = netx.ProxyOff
	case netx.ProxyManual:
		cfg.ProxyMode = netx.ProxyManual
		if cfg.ProxyURL == "" {
			return cfg, fmt.Errorf("manual proxy requires a URL")
		}
	default:
		mode, url, err := netx.ParseProxyMode(d.ProxyMode)
		if err != nil {
			return cfg, err
		}
		cfg.ProxyMode = mode
		if mode == netx.ProxyManual {
			cfg.ProxyURL = url
		}
	}
	return cfg, nil
}
