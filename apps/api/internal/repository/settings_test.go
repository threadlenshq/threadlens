package repository

import (
	"context"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func TestSetSetting_InsertsNewKey(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := New(db)
	ctx := context.Background()

	if err := repo.SetSetting(ctx, "test_key", "test_value"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}

	val, ok, err := repo.GetSetting(ctx, "test_key")
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if !ok {
		t.Fatal("GetSetting returned ok=false after SetSetting")
	}
	if val != "test_value" {
		t.Errorf("GetSetting = %q; want %q", val, "test_value")
	}
}

func TestSetSetting_UpdatesExistingKey(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := New(db)
	ctx := context.Background()

	if err := repo.SetSetting(ctx, "test_key", "first"); err != nil {
		t.Fatalf("SetSetting (first): %v", err)
	}
	if err := repo.SetSetting(ctx, "test_key", "second"); err != nil {
		t.Fatalf("SetSetting (second): %v", err)
	}

	val, ok, err := repo.GetSetting(ctx, "test_key")
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if !ok {
		t.Fatal("GetSetting returned ok=false after update")
	}
	if val != "second" {
		t.Errorf("GetSetting = %q; want %q", val, "second")
	}
}

func TestSetSetting_RoundTripsWithGetSetting(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := New(db)
	ctx := context.Background()

	cases := []struct {
		key   string
		value string
	}{
		{"ai_provider", "opencode"},
		{"ai_provider", "claude-cli"},
		{"empty_value", ""},
		{"special_chars", "hello=world&foo"},
	}

	for _, tc := range cases {
		if err := repo.SetSetting(ctx, tc.key, tc.value); err != nil {
			t.Fatalf("SetSetting(%q, %q): %v", tc.key, tc.value, err)
		}
		val, ok, err := repo.GetSetting(ctx, tc.key)
		if err != nil {
			t.Fatalf("GetSetting(%q): %v", tc.key, err)
		}
		if !ok {
			t.Fatalf("GetSetting(%q) returned ok=false", tc.key)
		}
		if val != tc.value {
			t.Errorf("GetSetting(%q) = %q; want %q", tc.key, val, tc.value)
		}
	}
}
