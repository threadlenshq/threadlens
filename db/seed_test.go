package db

import (
	"context"
	"path/filepath"
	"testing"
)

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
