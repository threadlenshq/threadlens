package services

import "testing"

func TestAdaptQueryURL_Reddit(t *testing.T) {
	got, err := AdaptQueryURL("reddit", "manual invoicing")
	if err != nil {
		t.Fatal(err)
	}
	// Matches the codebase's canonical site-wide default (see query_ai.go query
	// format rules). The .json suffix is on the PATH, not a param value.
	want := "https://www.reddit.com/search.json?q=manual+invoicing&sort=new&t=month&limit=100"
	if got != want {
		t.Fatalf("reddit url\n got: %q\nwant: %q", got, want)
	}
}

func TestAdaptQueryURL_GoogleAndBluesky(t *testing.T) {
	g, err := AdaptQueryURL("google", "  manual invoicing  ")
	if err != nil {
		t.Fatal(err)
	}
	if g != "manual invoicing" {
		t.Fatalf("google: got %q", g)
	}
	b, err := AdaptQueryURL("bluesky", "manual invoicing")
	if err != nil {
		t.Fatal(err)
	}
	if b != "manual invoicing" {
		t.Fatalf("bluesky: got %q", b)
	}
}

func TestAdaptQueryURL_Errors(t *testing.T) {
	if _, err := AdaptQueryURL("reddit", "   "); err == nil {
		t.Fatal("want error on empty text")
	}
	if _, err := AdaptQueryURL("myspace", "x"); err == nil {
		t.Fatal("want error on unknown platform")
	}
}

func TestNormalizeQueryURL(t *testing.T) {
	a := NormalizeQueryURL("google", "  Manual   Invoicing ")
	b := NormalizeQueryURL("google", "manual invoicing")
	if a != b {
		t.Fatalf("normalize mismatch: %q vs %q", a, b)
	}
}
