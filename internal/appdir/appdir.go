// Package appdir resolves where ODM stores its local database and config.
package appdir

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolve returns the data directory for ODM.
//
// Priority:
//  1. explicit override (CLI --data-dir)
//  2. environment variable ODM_DATA_DIR
//  3. os.UserConfigDir()/odm  (e.g. %AppData%\odm on Windows)
func Resolve(override string) (string, error) {
	if override = strings.TrimSpace(override); override != "" {
		return filepath.Abs(override)
	}
	if env := strings.TrimSpace(os.Getenv("ODM_DATA_DIR")); env != "" {
		return filepath.Abs(env)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(base, "odm"), nil
}

// DBPath joins the data directory with the default database filename.
func DBPath(dataDir string) string {
	return filepath.Join(dataDir, "odm.db")
}
