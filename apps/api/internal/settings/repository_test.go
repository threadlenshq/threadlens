package settings_test

import (
	"context"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/settings"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func TestGet_NotFound(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := settings.NewRepository(db)

	_, found, err := repo.Get(context.Background(), "missing.key")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("want not found, got found")
	}
}

func TestSet_And_Get(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := settings.NewRepository(db)

	if err := repo.Set(context.Background(), "onboarding.complete", "true"); err != nil {
		t.Fatal(err)
	}

	val, found, err := repo.Get(context.Background(), "onboarding.complete")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("want found, got not found")
	}
	if val != "true" {
		t.Fatalf("want %q, got %q", "true", val)
	}
}

func TestSet_Upsert(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := settings.NewRepository(db)

	if err := repo.Set(context.Background(), "k", "v1"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Set(context.Background(), "k", "v2"); err != nil {
		t.Fatal(err)
	}

	val, found, err := repo.Get(context.Background(), "k")
	if err != nil {
		t.Fatal(err)
	}
	if !found || val != "v2" {
		t.Fatalf("want v2, got %q (found=%v)", val, found)
	}
}

func TestDelete_Existing(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := settings.NewRepository(db)

	if err := repo.Set(context.Background(), "k", "v"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(context.Background(), "k"); err != nil {
		t.Fatal(err)
	}

	_, found, err := repo.Get(context.Background(), "k")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("want not found after delete, got found")
	}
}

func TestDelete_Missing_NoError(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := settings.NewRepository(db)

	// Deleting a key that doesn't exist should not return an error.
	if err := repo.Delete(context.Background(), "nonexistent"); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
}

func TestListPrefix(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := settings.NewRepository(db)

	keys := map[string]string{
		"model.scoring":  "gpt-4",
		"model.analysis": "claude",
		"onboarding.done": "true",
	}
	for k, v := range keys {
		if err := repo.Set(context.Background(), k, v); err != nil {
			t.Fatal(err)
		}
	}

	results, err := repo.ListPrefix(context.Background(), "model.")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d: %v", len(results), results)
	}
	for k := range results {
		if k != "model.scoring" && k != "model.analysis" {
			t.Fatalf("unexpected key %q", k)
		}
	}
}

func TestListPrefix_Empty(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := settings.NewRepository(db)

	results, err := repo.ListPrefix(context.Background(), "nope.")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("want empty map, got %v", results)
	}
}
