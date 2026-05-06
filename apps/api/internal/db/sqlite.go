// go:build ignore
//
// This file is excluded from compilation. The legacy SQLite bootstrap path it
// contained has been superseded by github.com/kyle/scout/open-core/db.
// Tests that previously relied on Open() / InitSchema() here have been
// migrated to wire_test.go which uses the shared module directly.
// Retained as a reference; delete when no longer needed.

//go:build ignore

package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Open opens an in-process SQLite database for legacy tests. New callers
// should use github.com/kyle/scout/open-core/db instead.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	if err := InitSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func NowUTCString() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}
