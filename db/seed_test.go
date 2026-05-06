package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"path/filepath"
	"strings"
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

// ---------------------------------------------------------------------------
// Portability unit tests — validate SQL shape and driver detection without
// requiring a live Postgres instance.
// ---------------------------------------------------------------------------

// TestSeedInsertSQLShapePostgres verifies that the Postgres variant of the
// INSERT statement uses $N positional placeholders (required by pgx/stdlib)
// and that none of the SQLite-only '?' markers appear.
func TestSeedInsertSQLShapePostgres(t *testing.T) {
	q, err := seedInsertSQL(DialectPostgres)
	if err != nil {
		t.Fatalf("seedInsertSQL(Postgres): %v", err)
	}
	if !strings.Contains(q, "$1") || !strings.Contains(q, "$2") || !strings.Contains(q, "$3") {
		t.Errorf("postgres SQL missing $N placeholders:\n%s", q)
	}
	if strings.Contains(q, "?") {
		t.Errorf("postgres SQL must not contain '?' placeholders:\n%s", q)
	}
	if !strings.Contains(q, "WHERE NOT EXISTS") {
		t.Errorf("postgres SQL missing WHERE NOT EXISTS guard:\n%s", q)
	}
}

// TestSeedInsertSQLShapeSQLite verifies the SQLite variant uses '?' markers
// and contains no $N positional parameters.
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
	if !strings.Contains(q, "WHERE NOT EXISTS") {
		t.Errorf("sqlite SQL missing WHERE NOT EXISTS guard:\n%s", q)
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
// recognise the underlying driver type.
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
	if !strings.Contains(err.Error(), "unrecognised driver type") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestSeedMarkerParamCountConsistency is a cross-dialect consistency check
// that ensures both SQL variants use exactly 3 parameter slots — matching the
// 3 arguments (key, value, key) passed by EnsureCoreSeedMarker.
func TestSeedMarkerParamCountConsistency(t *testing.T) {
	tests := []struct {
		dialect     Dialect
		wantCount   int
		countFn     func(string) int
	}{
		{
			dialect:   DialectSQLite,
			wantCount: 3,
			countFn:   func(q string) int { return strings.Count(q, "?") },
		},
		{
			dialect:   DialectPostgres,
			wantCount: 3,
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
