package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
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
// the same key and verifies that exactly one call reports "seeded" and the
// rest report "noop" — confirming the INSERT OR IGNORE / ON CONFLICT DO NOTHING
// strategy is race-safe without a transaction.
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

	seededCount := 0
	for i, e := range errs {
		if e != nil {
			t.Errorf("worker %d returned error: %v", i, e)
		}
	}
	for _, r := range results {
		if r.Status == "seeded" {
			seededCount++
		}
	}
	if seededCount != 1 {
		t.Errorf("seededCount = %d, want exactly 1", seededCount)
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

// fakeDriver is a minimal driver.Driver used to test dialectFromDB detection.
type fakeDriver struct{ name string }

func (fakeDriver) Open(_ string) (driver.Conn, error) { return nil, nil }

// TestDialectFromDBSQLite verifies that a *sql.DB backed by the real SQLite
// driver resolves to DialectSQLite.
func TestDialectFromDBSQLite(t *testing.T) {
	database, err := Open(context.Background(), Config{Dialect: DialectSQLite, SQLitePath: filepath.Join(t.TempDir(), "detect.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	got, err := dialectFromDB(database)
	if err != nil {
		t.Fatalf("dialectFromDB: %v", err)
	}
	if got != DialectSQLite {
		t.Errorf("dialectFromDB = %q, want %q", got, DialectSQLite)
	}
}

// TestDialectFromDBUnknownDriver registers a synthetic driver and confirms
// dialectFromDB returns an error (not a silent wrong dialect) when it cannot
// recognise the underlying driver.
func TestDialectFromDBUnknownDriver(t *testing.T) {
	driverName := "fake_driver_for_test_" + t.Name()
	sql.Register(driverName, fakeDriver{name: driverName})

	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = dialectFromDB(db)
	if err == nil {
		t.Error("expected error for unknown driver, got nil")
	}
	if !strings.Contains(err.Error(), "unrecognised driver") {
		t.Errorf("unexpected error message: %v", err)
	}
}

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
