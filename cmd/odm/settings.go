package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
)

// runSettings handles: odm settings show|set
func runSettings(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: odm settings <show|set>")
	}
	switch args[0] {
	case "show", "get", "list":
		return runSettingsShow(ctx, args[1:])
	case "set":
		return runSettingsSet(ctx, args[1:])
	default:
		return fmt.Errorf("unknown settings subcommand %q", args[0])
	}
}

func runSettingsShow(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("settings show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var dataDir string
	fs.StringVar(&dataDir, "data-dir", "", "ODM data directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	_, store, dataPath, closeFn, err := openManagerFromFlags(dataDir, nil)
	if err != nil {
		return err
	}
	defer closeFn()

	cfg, err := store.GetSettings(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "data_dir=%s\n", dataPath)
	fmt.Fprintf(os.Stdout, "proxy.mode=%s\n", cfg.ProxyMode)
	fmt.Fprintf(os.Stdout, "proxy.url=%s\n", cfg.ProxyURL)
	fmt.Fprintf(os.Stdout, "queue.max_active=%d\n", cfg.MaxActiveDownloads)
	fmt.Fprintf(os.Stdout, "download.max_connections=%d\n", cfg.MaxConnectionsPerDownload)
	fmt.Fprintf(os.Stdout, "download.max_retries=%d\n", cfg.MaxRetries)
	fmt.Fprintf(os.Stdout, "download.connect_timeout_sec=%d\n", cfg.ConnectTimeoutSec)
	fmt.Fprintf(os.Stdout, "download.stall_timeout_sec=%d\n", cfg.StallTimeoutSec)
	fmt.Fprintf(os.Stdout, "download.directory=%s\n", cfg.DownloadDirectory)
	fmt.Fprintf(os.Stdout, "ui.theme=%s\n", cfg.Theme)
	fmt.Fprintf(os.Stdout, "ui.notify_on_complete=%v\n", cfg.NotifyOnComplete)
	return nil
}

func runSettingsSet(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("settings set", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var dataDir string
	fs.StringVar(&dataDir, "data-dir", "", "ODM data directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("usage: odm settings set <key> <value>")
	}
	key, value := fs.Arg(0), fs.Arg(1)

	_, store, _, closeFn, err := openManagerFromFlags(dataDir, nil)
	if err != nil {
		return err
	}
	defer closeFn()

	cfg, err := store.GetSettings(ctx)
	if err != nil {
		return err
	}
	if err := applySetting(&cfg, key, value); err != nil {
		return err
	}
	if err := store.SaveSettings(ctx, cfg); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "set %s\n", key)
	return nil
}

func applySetting(cfg *storage.Settings, key, value string) error {
	key = strings.TrimSpace(key)
	switch key {
	case storage.SettingProxyMode, "proxy":
		mode, proxyURL, err := netx.ParseProxyMode(value)
		if err != nil {
			return err
		}
		cfg.ProxyMode = mode
		if mode == netx.ProxyManual {
			cfg.ProxyURL = proxyURL
		}
		if mode != netx.ProxyManual {
			// Keep URL stored but unused unless mode is manual.
		}
		return nil
	case storage.SettingProxyURL:
		cfg.ProxyURL = value
		return nil
	case storage.SettingMaxActive, "max-active":
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return fmt.Errorf("max active must be a positive integer")
		}
		cfg.MaxActiveDownloads = n
		return nil
	case storage.SettingMaxConnections, "max-connections", "connections":
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return fmt.Errorf("max connections must be a positive integer")
		}
		cfg.MaxConnectionsPerDownload = n
		return nil
	case storage.SettingMaxRetries, "max-retries":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("max retries must be >= 0")
		}
		cfg.MaxRetries = n
		return nil
	case storage.SettingConnectTimeoutSec:
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return fmt.Errorf("connect timeout must be > 0 seconds")
		}
		cfg.ConnectTimeoutSec = n
		return nil
	case storage.SettingStallTimeoutSec:
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return fmt.Errorf("stall timeout must be > 0 seconds")
		}
		cfg.StallTimeoutSec = n
		return nil
	case storage.SettingDownloadDir, "dir":
		cfg.DownloadDirectory = value
		return nil
	case storage.SettingTheme, "theme":
		v := strings.ToLower(strings.TrimSpace(value))
		switch v {
		case "system", "light", "dark":
			cfg.Theme = v
			return nil
		default:
			return fmt.Errorf("theme must be system, light, or dark")
		}
	case storage.SettingNotifyComplete, "notify-on-complete", "notify":
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true", "1", "yes", "on":
			cfg.NotifyOnComplete = true
		case "false", "0", "no", "off":
			cfg.NotifyOnComplete = false
		default:
			return fmt.Errorf("notify_on_complete must be true or false")
		}
		return nil
	default:
		return fmt.Errorf("unknown settings key %q", key)
	}
}
