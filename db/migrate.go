package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// Migrate ensures schema_migrations exists and applies any unapplied core
// migrations for the given dialect. It is safe to call multiple times; already-
// applied migrations are skipped.
func Migrate(ctx context.Context, db *sql.DB, dialect Dialect) error {
	if err := ensureMigrationsTable(ctx, db, dialect); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	migrations, err := loadMigrations(dialect)
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	for _, m := range migrations {
		applied, err := migrationApplied(ctx, db, m.version, dialect)
		if err != nil {
			return fmt.Errorf("check migration %q: %w", m.version, err)
		}
		if applied {
			continue
		}
		if err := applyMigration(ctx, db, dialect, m); err != nil {
			return fmt.Errorf("apply migration %q: %w", m.version, err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal types
// ---------------------------------------------------------------------------

type migration struct {
	version string
	sql     string
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// loadMigrations reads all *.sql files for the given dialect from the embedded
// FS and returns them sorted by filename (i.e. by version).
func loadMigrations(dialect Dialect) ([]migration, error) {
	dir := path.Join("migrations", string(dialect))
	entries, err := fs.ReadDir(migrationFS, dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %q: %w", dir, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var result []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		data, err := migrationFS.ReadFile(path.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read migration file %q: %w", e.Name(), err)
		}
		version := strings.TrimSuffix(e.Name(), ".sql")
		result = append(result, migration{version: version, sql: string(data)})
	}
	return result, nil
}

// migrationApplied returns true if a core migration with the given version has
// already been recorded in schema_migrations.
func migrationApplied(ctx context.Context, db *sql.DB, version string, dialect Dialect) (bool, error) {
	var q string
	switch dialect {
	case DialectPostgres:
		q = `SELECT COUNT(*) FROM schema_migrations WHERE version = $1 AND scope = 'core'`
	case DialectSQLite:
		q = `SELECT COUNT(*) FROM schema_migrations WHERE version = ? AND scope = 'core'`
	default:
		return false, fmt.Errorf("unsupported dialect: %q", dialect)
	}

	var count int
	if err := db.QueryRowContext(ctx, q, version).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// applyMigration runs the migration SQL inside a transaction and records the
// version in schema_migrations.
func applyMigration(ctx context.Context, db *sql.DB, dialect Dialect, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return fmt.Errorf("exec migration SQL: %w", err)
	}

	ins, err2 := insertMigrationSQL(dialect)
	if err2 != nil {
		return fmt.Errorf("insert migration SQL: %w", err2)
	}
	if _, err := tx.ExecContext(ctx, ins, m.version, "core"); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	return tx.Commit()
}

// ensureMigrationsTable creates schema_migrations if it does not yet exist.
func ensureMigrationsTable(ctx context.Context, db *sql.DB, dialect Dialect) error {
	q, err := migrationTableSQL(dialect)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, q)
	return err
}

// migrationTableSQL returns the CREATE TABLE statement for schema_migrations
// appropriate for the given dialect.
func migrationTableSQL(dialect Dialect) (string, error) {
	switch dialect {
	case DialectPostgres:
		return `CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT NOT NULL,
  scope   TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (version, scope)
)`, nil
	case DialectSQLite:
		return `CREATE TABLE IF NOT EXISTS schema_migrations (
  version    TEXT NOT NULL,
  scope      TEXT NOT NULL,
  applied_at DATETIME NOT NULL DEFAULT (datetime('now')),
  PRIMARY KEY (version, scope)
)`, nil
	default:
		return "", fmt.Errorf("unsupported dialect: %q", dialect)
	}
}

// insertMigrationSQL returns a parameterised INSERT statement for the
// schema_migrations table appropriate for the given dialect.
func insertMigrationSQL(dialect Dialect) (string, error) {
	switch dialect {
	case DialectPostgres:
		return `INSERT INTO schema_migrations (version, scope) VALUES ($1, $2)`, nil
	case DialectSQLite:
		return `INSERT INTO schema_migrations (version, scope) VALUES (?, ?)`, nil
	default:
		return "", fmt.Errorf("unsupported dialect: %q", dialect)
	}
}
