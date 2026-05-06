package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// Open opens and configures a database connection for the given Config,
// applies all pending migrations, and returns the ready-to-use *sql.DB.
func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	switch cfg.Dialect {
	case DialectSQLite:
		return openSQLite(ctx, cfg)
	case DialectPostgres:
		return openPostgres(ctx, cfg)
	default:
		return nil, fmt.Errorf("db.Open: unsupported dialect %q", cfg.Dialect)
	}
}

func openSQLite(ctx context.Context, cfg Config) (*sql.DB, error) {
	if cfg.SQLitePath == "" {
		return nil, errors.New("db.Open: SQLitePath is required for sqlite dialect")
	}

	// Encode PRAGMAs in the DSN so modernc.org/sqlite applies them on every
	// new connection it opens (they are connection-scoped in SQLite).
	dsn := buildSQLiteDSN(cfg.SQLitePath)

	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db.Open: sql.Open sqlite: %w", err)
	}

	// SQLite works best with a single writer connection.
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)

	if err := Migrate(ctx, database, DialectSQLite); err != nil {
		database.Close()
		return nil, fmt.Errorf("db.Open: migrate: %w", err)
	}

	return database, nil
}

// buildSQLiteDSN returns a DSN with the required PRAGMAs embedded as
// _pragma query parameters. modernc.org/sqlite applies these on every new
// connection, which is required because foreign_keys and busy_timeout are
// connection-scoped settings in SQLite.
func buildSQLiteDSN(path string) string {
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	return "file:" + path + "?" + q.Encode()
}

func openPostgres(ctx context.Context, cfg Config) (*sql.DB, error) {
	if cfg.DatabaseURL == "" {
		return nil, errors.New("db.Open: DatabaseURL is required for postgres dialect")
	}

	database, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("db.Open: sql.Open pgx: %w", err)
	}

	if err := database.PingContext(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("db.Open: ping postgres: %w", err)
	}

	if err := Migrate(ctx, database, DialectPostgres); err != nil {
		database.Close()
		return nil, fmt.Errorf("db.Open: migrate: %w", err)
	}

	return database, nil
}
