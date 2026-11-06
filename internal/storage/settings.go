package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
)

// Settings keys stored in the settings table.
const (
	SettingProxyMode         = "proxy.mode"               // system | off | manual
	SettingProxyURL          = "proxy.url"                // used when mode=manual
	SettingMaxActive         = "queue.max_active"         // concurrent downloads
	SettingMaxConnections    = "download.max_connections" // per-download Range workers
	SettingMaxRetries        = "download.max_retries"
	SettingConnectTimeoutSec = "download.connect_timeout_sec"
	SettingStallTimeoutSec   = "download.stall_timeout_sec"
	SettingDownloadDir       = "download.directory"    // default folder hint
	SettingTheme             = "ui.theme"              // system | light | dark
	SettingNotifyComplete    = "ui.notify_on_complete" // true | false
)

// Settings is the typed view of persisted configuration.
type Settings struct {
	ProxyMode                 netx.ProxyMode
	ProxyURL                  string
	MaxActiveDownloads        int
	MaxConnectionsPerDownload int
	MaxRetries                int
	ConnectTimeoutSec         int
	StallTimeoutSec           int
	DownloadDirectory         string
	Theme                     string // system | light | dark
	NotifyOnComplete          bool
}

// DefaultSettings returns Phase 2/3 defaults.
func DefaultSettings() Settings {
	return Settings{
		ProxyMode:                 netx.ProxySystem,
		ProxyURL:                  "",
		MaxActiveDownloads:        2,
		MaxConnectionsPerDownload: 4, // Phase 3: try segmented when Range verifies
		MaxRetries:                5,
		ConnectTimeoutSec:         30,
		StallTimeoutSec:           30,
		DownloadDirectory:         "",
		Theme:                     "system",
		NotifyOnComplete:          false,
	}
}

// GetSettings loads settings, filling defaults for missing keys.
func (s *Store) GetSettings(ctx context.Context) (Settings, error) {
	out := DefaultSettings()

	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return out, fmt.Errorf("load settings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return out, err
		}
		switch k {
		case SettingProxyMode:
			out.ProxyMode = netx.ProxyMode(strings.TrimSpace(v))
		case SettingProxyURL:
			out.ProxyURL = v
		case SettingMaxActive:
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				out.MaxActiveDownloads = n
			}
		case SettingMaxConnections:
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				out.MaxConnectionsPerDownload = n
			}
		case SettingMaxRetries:
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				out.MaxRetries = n
			}
		case SettingConnectTimeoutSec:
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				out.ConnectTimeoutSec = n
			}
		case SettingStallTimeoutSec:
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				out.StallTimeoutSec = n
			}
		case SettingDownloadDir:
			out.DownloadDirectory = v
		case SettingTheme:
			switch strings.TrimSpace(strings.ToLower(v)) {
			case "light", "dark", "system":
				out.Theme = strings.TrimSpace(strings.ToLower(v))
			}
		case SettingNotifyComplete:
			out.NotifyOnComplete = strings.EqualFold(strings.TrimSpace(v), "true") || v == "1"
		}
	}
	return out, rows.Err()
}

// SaveSettings writes all known settings keys in one transaction.
func (s *Store) SaveSettings(ctx context.Context, cfg Settings) error {
	if cfg.MaxActiveDownloads <= 0 {
		return fmt.Errorf("max active downloads must be > 0")
	}
	if cfg.MaxConnectionsPerDownload <= 0 {
		return fmt.Errorf("max connections per download must be > 0")
	}
	if cfg.ProxyMode == netx.ProxyManual && strings.TrimSpace(cfg.ProxyURL) == "" {
		return fmt.Errorf("manual proxy requires a URL")
	}
	theme := strings.TrimSpace(strings.ToLower(cfg.Theme))
	if theme == "" {
		theme = "system"
	}
	switch theme {
	case "system", "light", "dark":
	default:
		return fmt.Errorf("theme must be system, light, or dark")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	pairs := map[string]string{
		SettingProxyMode:         string(cfg.ProxyMode),
		SettingProxyURL:          cfg.ProxyURL,
		SettingMaxActive:         strconv.Itoa(cfg.MaxActiveDownloads),
		SettingMaxConnections:    strconv.Itoa(cfg.MaxConnectionsPerDownload),
		SettingMaxRetries:        strconv.Itoa(cfg.MaxRetries),
		SettingConnectTimeoutSec: strconv.Itoa(cfg.ConnectTimeoutSec),
		SettingStallTimeoutSec:   strconv.Itoa(cfg.StallTimeoutSec),
		SettingDownloadDir:       cfg.DownloadDirectory,
		SettingTheme:             theme,
		SettingNotifyComplete:    strconv.FormatBool(cfg.NotifyOnComplete),
	}
	for k, v := range pairs {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO settings(key, value) VALUES(?, ?)
ON CONFLICT(key) DO UPDATE SET value=excluded.value`, k, v); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("save setting %s: %w", k, err)
		}
	}
	return tx.Commit()
}

// SetSetting upserts a single key (used by CLI `settings set`).
func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("settings key must not be empty")
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO settings(key, value) VALUES(?, ?)
ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	if err != nil {
		return fmt.Errorf("set setting: %w", err)
	}
	return nil
}

// GetSetting returns one raw value or "" if missing.
func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}
