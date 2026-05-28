package google

import (
	"context"
	"net/http"
	"testing"
)

// roundTripFunc is an http.RoundTripper that calls fn for every request.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestParallelSearchProviderSearchBatchMissingAPIKey(t *testing.T) {
	p := &ParallelSearchProvider{
		APIKey: "",
		Client: &http.Client{},
	}
	_, err := p.SearchBatch(context.Background(), []string{"test query"}, SearchOptions{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	want := "Missing Parallel configuration (set PARALLEL_API_KEY)"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestParallelSearchProviderSearchBatchBlankAPIKey(t *testing.T) {
	// A blank key (whitespace only) must be rejected before any HTTP request is made.
	requestMade := false
	p := &ParallelSearchProvider{
		APIKey: "   ",
		Client: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				requestMade = true
				return nil, nil
			}),
		},
	}
	_, err := p.SearchBatch(context.Background(), []string{"test query"}, SearchOptions{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	want := "Missing Parallel configuration (set PARALLEL_API_KEY)"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
	if requestMade {
		t.Fatal("expected no HTTP request to be made for blank API key")
	}
}
