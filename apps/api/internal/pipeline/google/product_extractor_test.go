package google

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
)

// mockProvider is a local test double that satisfies ai.Provider.
type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Name() string    { return "copilot" }
func (m *mockProvider) Available() bool { return true }
func (m *mockProvider) Generate(_ context.Context, _ string, _ string, _ string, _ time.Duration) (string, error) {
	return m.response, m.err
}

func makeTestService(response string, shouldErr bool) *ai.Service {
	var err error
	if shouldErr {
		err = errors.New("mock error")
	}
	return ai.NewServiceWithProviders([]ai.Provider{&mockProvider{response: response, err: err}})
}

func TestExtractMentionedProductsBasic(t *testing.T) {
	svc := makeTestService(`["Notion","Linear","Trello"]`, false)
	result := AnalyzedResult{Title: "Best task managers", Snippet: "Notion vs Linear vs Trello"}
	products, err := ExtractMentionedProducts(context.Background(), result, svc)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"Notion", "Linear", "Trello"}
	if len(products) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, products)
	}
	for i, p := range products {
		if p != expected[i] {
			t.Errorf("[%d] expected %q, got %q", i, expected[i], p)
		}
	}
}

func TestExtractMentionedProductsWrappedJSON(t *testing.T) {
	svc := makeTestService(`Here are the products: ["Notion","Asana"] - those are the named ones.`, false)
	result := AnalyzedResult{Title: "Test", Snippet: "test"}
	products, _ := ExtractMentionedProducts(context.Background(), result, svc)
	if len(products) != 2 || products[0] != "Notion" || products[1] != "Asana" {
		t.Errorf("unexpected: %v", products)
	}
}

func TestExtractMentionedProductsInvalidJSON(t *testing.T) {
	svc := makeTestService("not valid json at all", false)
	result := AnalyzedResult{Title: "Test", Snippet: "test"}
	products, _ := ExtractMentionedProducts(context.Background(), result, svc)
	if len(products) != 0 {
		t.Errorf("expected empty, got %v", products)
	}
}

func TestExtractMentionedProductsEmpty(t *testing.T) {
	svc := makeTestService("[]", false)
	result := AnalyzedResult{Title: "Test", Snippet: "test"}
	products, _ := ExtractMentionedProducts(context.Background(), result, svc)
	if len(products) != 0 {
		t.Errorf("expected empty, got %v", products)
	}
}

func TestExtractMentionedProductsError(t *testing.T) {
	svc := makeTestService("", true)
	result := AnalyzedResult{Title: "Test", Snippet: "test"}
	products, _ := ExtractMentionedProducts(context.Background(), result, svc)
	if len(products) != 0 {
		t.Errorf("expected empty on error, got %v", products)
	}
}

func TestExtractMentionedProductsDeduplicatesCaseInsensitive(t *testing.T) {
	svc := makeTestService(`["Notion","notion","NOTION","Linear"]`, false)
	result := AnalyzedResult{Title: "Test", Snippet: "test"}
	products, _ := ExtractMentionedProducts(context.Background(), result, svc)

	notionCount := 0
	hasLinear := false
	for _, p := range products {
		if p == "Notion" {
			notionCount++
		}
		if p == "Linear" {
			hasLinear = true
		}
	}
	if notionCount != 1 {
		t.Errorf("expected 1 Notion, got %d", notionCount)
	}
	if !hasLinear {
		t.Errorf("expected Linear in products")
	}
}

func TestExtractMentionedProductsCapsAt10(t *testing.T) {
	svc := makeTestService(`["A","B","C","D","E","F","G","H","I","J","K","L"]`, false)
	result := AnalyzedResult{Title: "Test", Snippet: "test"}
	products, _ := ExtractMentionedProducts(context.Background(), result, svc)
	if len(products) > 10 {
		t.Errorf("expected <= 10, got %d", len(products))
	}
}

func TestExtractMentionedProductsNoContent(t *testing.T) {
	svc := makeTestService(`["Notion"]`, false)
	result := AnalyzedResult{Title: "", Snippet: "", Summary: ""}
	products, _ := ExtractMentionedProducts(context.Background(), result, svc)
	if len(products) != 0 {
		t.Errorf("expected empty for no-content, got %v", products)
	}
}

func TestExtractProductsFromResultsBelowThreshold(t *testing.T) {
	results := []AnalyzedResult{
		{URL: "https://a.com", RelevanceScore: 1, Title: "Low", Snippet: "test"},
		{URL: "https://b.com", RelevanceScore: 2, Title: "Low2", Snippet: "test"},
	}
	svc := makeTestService(`["Notion"]`, false)
	out, err := ExtractProductsFromResults(context.Background(), results, svc)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2, got %d", len(out))
	}
	if len(out[0].MentionedProducts) != 0 || len(out[1].MentionedProducts) != 0 {
		t.Errorf("expected empty mentioned_products for all low-relevance results")
	}
}

func TestExtractProductsFromResultsAboveThreshold(t *testing.T) {
	results := []AnalyzedResult{
		{URL: "https://a.com", RelevanceScore: 5, Title: "Relevant", Snippet: "uses Notion and Linear"},
		{URL: "https://b.com", RelevanceScore: 1, Title: "Low", Snippet: "nothing"},
	}
	svc := makeTestService(`["Notion","Linear"]`, false)
	out, _ := ExtractProductsFromResults(context.Background(), results, svc)

	var relevant, low *AnalyzedResult
	for i := range out {
		if out[i].URL == "https://a.com" {
			relevant = &out[i]
		}
		if out[i].URL == "https://b.com" {
			low = &out[i]
		}
	}

	if relevant == nil || !containsString(relevant.MentionedProducts, "Notion") {
		t.Errorf("expected Notion in relevant result: %v", relevant)
	}
	if low == nil || len(low.MentionedProducts) != 0 {
		t.Errorf("expected empty for low result: %v", low)
	}
}

func TestExtractProductsFromResultsEmpty(t *testing.T) {
	svc := makeTestService(`["Notion"]`, false)
	out, _ := ExtractProductsFromResults(context.Background(), []AnalyzedResult{}, svc)
	if len(out) != 0 {
		t.Errorf("expected empty, got %v", out)
	}
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
