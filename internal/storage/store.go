// Package storage persists tasks and settings in SQLite.
//
// Why SQLite?
//   - One local file, no separate database server.
//   - Good enough for thousands of download tasks.
//   - Works offline (important for a download manager).
//
// Why modernc.org/sqlite?
//   - Pure Go driver (no CGO), easier cross-compile on Windows/macOS/Linux.
package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	// Blank import registers the database driver named "sqlite".
	// The `_` means "import only for side effects" (driver registration).
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Store wraps an open *sql.DB plus helpers for tasks/settings.
type Store struct {
	db *sql.DB
}

// Open creates (if needed) and migrates a SQLite database at dbPath.
//
// Example dbPath: C:\Users\You\AppData\Local\odm\odm.db
func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	// The modernc driver name is "sqlite". The DSN is just a file path.
	// `_pragma=busy_timeout` helps when the CLI and a future UI share the DB.
	dsn := dbPath + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite handles concurrency best with a single writer connection from Go.
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the database file handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// DB exposes the underlying *sql.DB for advanced tests.
func (s *Store) DB() *sql.DB {
	return s.db
}

// migrate applies every embedded *.sql file in version order exactly once.
func (s *Store) migrate(ctx context.Context) error {
	// Ensure the migrations bookkeeping table exists even before 001 runs.
	// 001 also creates it; IF NOT EXISTS makes this idempotent.
	if _, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
);`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names) // 001_..., 002_..., ...

	for _, name := range names {
		version, err := parseMigrationVersion(name)
		if err != nil {
			return err
		}

		applied, err := s.isMigrationApplied(ctx, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		// Run each migration in a transaction so a mid-file failure rolls back.
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`,
			version, time.Now().UTC().Format(time.RFC3339Nano),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func parseMigrationVersion(name string) (int, error) {
	// Expect names like "001_init.sql".
	parts := strings.SplitN(name, "_", 2)
	if len(parts) < 1 {
		return 0, fmt.Errorf("bad migration name %q", name)
	}
	v, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("bad migration version in %q: %w", name, err)
	}
	return v, nil
}

func (s *Store) isMigrationApplied(ctx context.Context, version int) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, version,
	).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
