package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// Functional tests — run against a real SQLite in-memory DB
// ---------------------------------------------------------------------------

func TestCoreSeedMarkerIsIdempotent(t *testing.T) {
	database, err := Open(context.Background(), Config{Dialect: DialectSQLite, SQLitePath: filepath.Join(t.TempDir(), "seed.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	first, err := EnsureCoreSeedMarker(context.Background(), database, "demo", 1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EnsureCoreSeedMarker(context.Background(), database, "demo", 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != "seeded" {
		t.Fatalf("first status = %q, want seeded", first.Status)
	}
	if first.Name != "demo" {
		t.Fatalf("first Name = %q, want demo", first.Name)
	}
	if first.Version != 1 {
		t.Fatalf("first Version = %d, want 1", first.Version)
	}
	if second.Status != "noop" {
		t.Fatalf("second status = %q, want noop", second.Status)
	}
}

func TestCoreSeedMarkerDistinctNamesAreIndependent(t *testing.T) {
	database, err := Open(context.Background(), Config{Dialect: DialectSQLite, SQLitePath: filepath.Join(t.TempDir(), "seed.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	r1, err := EnsureCoreSeedMarker(context.Background(), database, "demo", 1)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := EnsureCoreSeedMarker(context.Background(), database, "other", 2)
	if err != nil {
		t.Fatal(err)
	}
	if r1.Status != "seeded" {
		t.Fatalf("r1.Status = %q, want seeded", r1.Status)
	}
	if r2.Status != "seeded" {
		t.Fatalf("r2.Status = %q, want seeded", r2.Status)
	}
}

// TestCoreSeedMarkerConcurrentIdempotency fires multiple concurrent calls for
// the same key and verifies that:
//  1. Exactly one call reports "seeded" (the row was inserted once).
//  2. All remaining calls report "noop" (the INSERT OR IGNORE was a no-op).
//  3. The database contains exactly one row for the key after all workers finish.
//
// This confirms the INSERT OR IGNORE / ON CONFLICT DO NOTHING strategy is
// race-safe without an additional transaction or read-then-write gap.
func TestCoreSeedMarkerConcurrentIdempotency(t *testing.T) {
	database, err := Open(context.Background(), Config{Dialect: DialectSQLite, SQLitePath: filepath.Join(t.TempDir(), "seed_concurrent.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	const workers = 20
	results := make([]SeedResult, workers)
	errs := make([]error, workers)

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := range workers {
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = EnsureCoreSeedMarker(context.Background(), database, "concurrent", 1)
		}(i)
	}
	wg.Wait()

	// All workers must succeed.
	for i, e := range errs {
		if e != nil {
			t.Errorf("worker %d returned error: %v", i, e)
		}
	}

	// Exactly one worker must have seeded the row; all others must be noop.
	seededCount := 0
	noopCount := 0
	for _, r := range results {
		switch r.Status {
		case "seeded":
			seededCount++
		case "noop":
			noopCount++
		default:
			t.Errorf("unexpected status %q", r.Status)
		}
	}
	if seededCount != 1 {
		t.Errorf("seededCount = %d, want exactly 1", seededCount)
	}
	if noopCount != workers-1 {
		t.Errorf("noopCount = %d, want %d", noopCount, workers-1)
	}

	// Confirm exactly one row in the DB for this key.
	var rowCount int
	err = database.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM app_settings WHERE key = ?`, "seed.core.concurrent").
		Scan(&rowCount)
	if err != nil {
		t.Fatalf("counting rows: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("row count in app_settings = %d, want exactly 1", rowCount)
	}
}

// ---------------------------------------------------------------------------
// Portability unit tests — validate SQL shape and driver detection without
// requiring a live Postgres instance.
// ---------------------------------------------------------------------------

// TestSeedInsertSQLShapePostgres verifies that the Postgres variant uses
// $N positional placeholders and ON CONFLICT DO NOTHING (not WHERE NOT EXISTS).
func TestSeedInsertSQLShapePostgres(t *testing.T) {
	q, err := seedInsertSQL(DialectPostgres)
	if err != nil {
		t.Fatalf("seedInsertSQL(Postgres): %v", err)
	}
	if !strings.Contains(q, "$1") || !strings.Contains(q, "$2") {
		t.Errorf("postgres SQL missing $N placeholders:\n%s", q)
	}
	if strings.Contains(q, "?") {
		t.Errorf("postgres SQL must not contain '?' placeholders:\n%s", q)
	}
	if !strings.Contains(strings.ToUpper(q), "ON CONFLICT") {
		t.Errorf("postgres SQL missing ON CONFLICT guard:\n%s", q)
	}
	if strings.Contains(strings.ToUpper(q), "WHERE NOT EXISTS") {
		t.Errorf("postgres SQL must not use WHERE NOT EXISTS (has race window):\n%s", q)
	}
}

// TestSeedInsertSQLShapeSQLite verifies the SQLite variant uses INSERT OR IGNORE
// with '?' markers and no $N parameters.
func TestSeedInsertSQLShapeSQLite(t *testing.T) {
	q, err := seedInsertSQL(DialectSQLite)
	if err != nil {
		t.Fatalf("seedInsertSQL(SQLite): %v", err)
	}
	if !strings.Contains(q, "?") {
		t.Errorf("sqlite SQL missing '?' placeholders:\n%s", q)
	}
	if strings.Contains(q, "$") {
		t.Errorf("sqlite SQL must not contain '$N' placeholders:\n%s", q)
	}
	upperQ := strings.ToUpper(q)
	if !strings.Contains(upperQ, "INSERT OR IGNORE") {
		t.Errorf("sqlite SQL missing INSERT OR IGNORE guard:\n%s", q)
	}
	if strings.Contains(upperQ, "WHERE NOT EXISTS") {
		t.Errorf("sqlite SQL must not use WHERE NOT EXISTS (has race window):\n%s", q)
	}
}

// TestSeedInsertSQLShapeUnsupportedDialect ensures an unknown dialect returns
// an error rather than silently producing incorrect SQL.
func TestSeedInsertSQLShapeUnsupportedDialect(t *testing.T) {
	_, err := seedInsertSQL("mysql")
	if err == nil {
		t.Error("expected error for unsupported dialect, got nil")
	}
}

// ---------------------------------------------------------------------------
// dialectFromDB detection tests
// ---------------------------------------------------------------------------

// fakeDriver is a minimal driver.Driver used to test that dialectFromDB
// returns an error for databases that don't respond to either probe query.
type fakeDriver struct{ name string }

func (fakeDriver) Open(_ string) (driver.Conn, error) { return nil, nil }

// TestDialectFromDBSQLite verifies that a *sql.DB backed by the real SQLite
// driver resolves to DialectSQLite via the probe-query mechanism.
func TestDialectFromDBSQLite(t *testing.T) {
	database, err := Open(context.Background(), Config{Dialect: DialectSQLite, SQLitePath: filepath.Join(t.TempDir(), "detect.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	got, err := dialectFromDB(context.Background(), database)
	if err != nil {
		t.Fatalf("dialectFromDB: %v", err)
	}
	if got != DialectSQLite {
		t.Errorf("dialectFromDB = %q, want %q", got, DialectSQLite)
	}
}

// TestDialectFromDBUnknownDriver registers a synthetic driver and confirms
// dialectFromDB returns an error (not a silent wrong dialect) when the
// connection cannot execute either probe query.
//
// The fakeConn returns an error for all queries, which simulates a driver
// that does not understand the dialect-probe SQL.
func TestDialectFromDBUnknownDriver(t *testing.T) {
	driverName := "fake_probe_driver_" + t.Name()
	sql.Register(driverName, fakeConnDriver{})

	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = dialectFromDB(context.Background(), db)
	if err == nil {
		t.Error("expected error for unrecognised driver, got nil")
	}
	if !strings.Contains(err.Error(), "unable to identify dialect") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ---------------------------------------------------------------------------
// fakeConnDriver — a driver whose connections fail every query, used by
// TestDialectFromDBUnknownDriver to exercise the probe-failure path.
// ---------------------------------------------------------------------------

type fakeConnDriver struct{}

func (fakeConnDriver) Open(_ string) (driver.Conn, error) { return fakeConn{}, nil }

type fakeConn struct{}

func (fakeConn) Prepare(_ string) (driver.Stmt, error) {
	return nil, fmt.Errorf("fakeConn: not implemented")
}
func (fakeConn) Close() error              { return nil }
func (fakeConn) Begin() (driver.Tx, error) { return nil, fmt.Errorf("fakeConn: not implemented") }

// ---------------------------------------------------------------------------
// Parameter count consistency
// ---------------------------------------------------------------------------

// TestSeedMarkerParamCountConsistency ensures both SQL variants use exactly 2
// parameter slots — matching the 2 arguments (key, value) passed by
// EnsureCoreSeedMarker after switching from the 3-arg WHERE NOT EXISTS form.
func TestSeedMarkerParamCountConsistency(t *testing.T) {
	tests := []struct {
		dialect   Dialect
		wantCount int
		countFn   func(string) int
	}{
		{
			dialect:   DialectSQLite,
			wantCount: 2,
			countFn:   func(q string) int { return strings.Count(q, "?") },
		},
		{
			dialect:   DialectPostgres,
			wantCount: 2,
			countFn:   func(q string) int { return strings.Count(q, "$") },
		},
	}

	for _, tc := range tests {
		t.Run(string(tc.dialect), func(t *testing.T) {
			q, err := seedInsertSQL(tc.dialect)
			if err != nil {
				t.Fatalf("seedInsertSQL(%s): %v", tc.dialect, err)
			}
			got := tc.countFn(q)
			if got != tc.wantCount {
				t.Errorf("parameter count = %d, want %d in:\n%s", got, tc.wantCount, q)
			}
		})
	}
}
