package main

import (
	"fmt"

	"github.com/lostfromlight1/Open-Download-Manager/internal/appdir"
	"github.com/lostfromlight1/Open-Download-Manager/internal/manager"
	"github.com/lostfromlight1/Open-Download-Manager/internal/storage"
)

// openManagerFromFlags opens the SQLite DB under the resolved data directory
// and returns a Manager plus a cleanup function (Close).
func openManagerFromFlags(dataDir string, decisions manager.DecisionHandler) (*manager.Manager, *storage.Store, string, func() error, error) {
	dir, err := appdir.Resolve(dataDir)
	if err != nil {
		return nil, nil, "", nil, err
	}
	store, err := storage.Open(appdir.DBPath(dir))
	if err != nil {
		return nil, nil, "", nil, fmt.Errorf("open database in %s: %w", dir, err)
	}
	return manager.New(store, decisions), store, dir, store.Close, nil
}
