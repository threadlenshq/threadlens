package google

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func makeMockLookup(ips []string, err error) LookupIPFunc {
	return func(hostname string) ([]string, error) {
		return ips, err
	}
}

func makeHTTPClientFromResponses(responses []*http.Response, errs []error) func(req *http.Request) (*http.Response, error) {
	idx := 0
	return func(req *http.Request) (*http.Response, error) {
		i := idx
		idx++
		if i < len(errs) && errs[i] != nil {
			return nil, errs[i]
		}
		if i < len(responses) {
			return responses[i], nil
		}
		return nil, nil
	}
}

// mockClient wraps a function as http.RoundTripper
type mockRoundTripper struct {
	fn func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.fn(req)
}

func newMockClient(fn func(req *http.Request) (*http.Response, error)) *http.Client {
	return &http.Client{
		Transport: &mockRoundTripper{fn: fn},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func makeHTTPResponse(status int, body, contentType, location string) *http.Response {
	headers := http.Header{}
	if contentType != "" {
		headers.Set("Content-Type", contentType)
	}
	if location != "" {
		headers.Set("Location", location)
	}
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	resp := &http.Response{
		StatusCode: status,
		Header:     headers,
		Body:       http.NoBody,
	}
	resp.Body = io_nopCloser(bodyReader)
	return resp
}

type nopCloser struct{ *strings.Reader }

func (n nopCloser) Close() error { return nil }

func io_nopCloser(r *strings.Reader) nopCloser { return nopCloser{r} }

func TestFetchPageContentRejectsNonHTTP(t *testing.T) {
	calls := 0
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		calls++
		return makeHTTPResponse(200, "<html><body>ok</body></html>", "text/html", ""), nil
	})

	result, err := fetchPageContentWithDeps(context.Background(), "ftp://example.com/file.txt", client, makeMockLookup([]string{"93.184.216.34"}, nil))
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Errorf("expected 0 fetch calls, got %d", calls)
	}
	if result.OK {
		t.Error("expected ok=false")
	}
	if result.Status != 400 {
		t.Errorf("expected status=400, got %d", result.Status)
	}
	if result.FinalURL != "ftp://example.com/file.txt" {
		t.Errorf("unexpected finalUrl: %q", result.FinalURL)
	}
}

func TestFetchPageContentBlocksLocalhost(t *testing.T) {
	calls := 0
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		calls++
		return makeHTTPResponse(200, "", "text/html", ""), nil
	})

	r1, _ := fetchPageContentWithDeps(context.Background(), "http://localhost:8080", client, makeMockLookup([]string{}, nil))
	r2, _ := fetchPageContentWithDeps(context.Background(), "http://192.168.1.12/private", client, makeMockLookup([]string{}, nil))

	if calls != 0 {
		t.Errorf("expected 0 fetch calls, got %d", calls)
	}
	if r1.Status != 403 {
		t.Errorf("localhost: expected 403, got %d", r1.Status)
	}
	if r2.Status != 403 {
		t.Errorf("private IP: expected 403, got %d", r2.Status)
	}
}

func TestFetchPageContentBlocksPrivateDNS(t *testing.T) {
	calls := 0
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		calls++
		return makeHTTPResponse(200, "", "text/html", ""), nil
	})

	result, _ := fetchPageContentWithDeps(context.Background(), "https://example.com/resource", client, makeMockLookup([]string{"10.20.30.40"}, nil))

	if calls != 0 {
		t.Errorf("expected 0 fetch calls, got %d", calls)
	}
	if result.Status != 403 {
		t.Errorf("expected 403, got %d", result.Status)
	}
}

func TestFetchPageContentBlocksDNSFailure(t *testing.T) {
	calls := 0
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		calls++
		return makeHTTPResponse(200, "", "text/html", ""), nil
	})

	result, _ := fetchPageContentWithDeps(context.Background(), "https://example.com/resource", client, makeMockLookup(nil, http.ErrAbortHandler))

	if calls != 0 {
		t.Errorf("expected 0 fetch calls, got %d", calls)
	}
	if result.Status != 403 {
		t.Errorf("expected 403, got %d", result.Status)
	}
}

func TestFetchPageContentBlocksEmptyDNS(t *testing.T) {
	calls := 0
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		calls++
		return makeHTTPResponse(200, "", "text/html", ""), nil
	})

	result, _ := fetchPageContentWithDeps(context.Background(), "https://example.com/resource", client, makeMockLookup([]string{}, nil))

	if calls != 0 {
		t.Errorf("expected 0 fetch calls, got %d", calls)
	}
	if result.Status != 403 {
		t.Errorf("expected 403, got %d", result.Status)
	}
}

func TestFetchPageContentFollowsRedirect(t *testing.T) {
	calls := 0
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return makeHTTPResponse(302, "", "text/html", "/article"), nil
		}
		return makeHTTPResponse(200, "<html><body><h1>Hello</h1><script>x</script>World</body></html>", "text/html; charset=utf-8", ""), nil
	})

	result, _ := fetchPageContentWithDeps(context.Background(), "https://docs.example.com/start", client, makeMockLookup([]string{"93.184.216.34"}, nil))

	if calls != 2 {
		t.Errorf("expected 2 fetch calls, got %d", calls)
	}
	if !result.OK {
		t.Errorf("expected ok=true")
	}
	if result.Status != 200 {
		t.Errorf("expected 200, got %d", result.Status)
	}
	if result.PageText != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", result.PageText)
	}
}

func TestFetchPageContentSendsHeaders(t *testing.T) {
	var capturedReq *http.Request
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		capturedReq = req
		return makeHTTPResponse(200, "<html><body>ok</body></html>", "text/html", ""), nil
	})

	fetchPageContentWithDeps(context.Background(), "https://example.com/page", client, makeMockLookup([]string{"93.184.216.34"}, nil))

	if capturedReq == nil {
		t.Fatal("no request captured")
	}
	ua := capturedReq.Header.Get("User-Agent")
	if !strings.Contains(ua, "Mozilla") || !strings.Contains(ua, "Chrome") {
		t.Errorf("unexpected User-Agent: %q", ua)
	}
	accept := capturedReq.Header.Get("Accept")
	if !strings.Contains(accept, "text/html") {
		t.Errorf("unexpected Accept: %q", accept)
	}
	al := capturedReq.Header.Get("Accept-Language")
	if !strings.Contains(al, "en") {
		t.Errorf("unexpected Accept-Language: %q", al)
	}
}
