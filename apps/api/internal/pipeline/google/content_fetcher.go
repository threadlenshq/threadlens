package google

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	defaultTimeoutMs    = 15000
	defaultMaxRedirects = 5
	defaultMaxRetries   = 1
	retryBackoffMs      = 500
	maxPageBytes        = 2 * 1024 * 1024
	defaultUserAgent    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

// FetchResult mirrors the return shape of fetchPageContent() from content-fetcher.js.
type FetchResult struct {
	OK          bool   `json:"ok"`
	Status      int    `json:"status"`
	PageText    string `json:"pageText"`
	FinalURL    string `json:"finalUrl"`
	ContentType string `json:"contentType"`
}

var nonContentTags = []string{"script", "style", "noscript", "svg", "iframe", "canvas", "template"}

var transientStatuses = map[int]bool{408: true, 425: true, 429: true, 500: true, 502: true, 503: true, 504: true}

var htmlEntityRe = regexp.MustCompile(`&#x?[0-9a-fA-F]+;|&[a-z]+;`)
var htmlTagRe = regexp.MustCompile(`<[^>]+>`)
var multiSpaceRe = regexp.MustCompile(`\s+`)
var aposEntityRe = regexp.MustCompile(`(?i)&#39;|&apos;`)
var linkLocalIPv6Re = regexp.MustCompile(`^fe[89ab]`)

// nonContentTagBlockRes holds one precompiled block-stripping regex per
// nonContentTags entry, built once at init rather than per fetched page.
var nonContentTagBlockRes = func() []*regexp.Regexp {
	res := make([]*regexp.Regexp, len(nonContentTags))
	for i, tag := range nonContentTags {
		res[i] = regexp.MustCompile(`(?i)<` + regexp.QuoteMeta(tag) + `\b[^>]*>[\s\S]*?</` + regexp.QuoteMeta(tag) + `>`)
	}
	return res
}()

func stripHtmlTags(html string) string {
	cleaned := htmlTagRe.ReplaceAllString(html, " ")
	cleaned = strings.ReplaceAll(cleaned, "&nbsp;", " ")
	cleaned = strings.ReplaceAll(cleaned, "&amp;", "&")
	cleaned = strings.ReplaceAll(cleaned, "&lt;", "<")
	cleaned = strings.ReplaceAll(cleaned, "&gt;", ">")
	cleaned = aposEntityRe.ReplaceAllString(cleaned, "'")
	cleaned = strings.ReplaceAll(cleaned, "&quot;", "\"")
	cleaned = multiSpaceRe.ReplaceAllString(cleaned, " ")
	return strings.TrimSpace(cleaned)
}

func extractPageText(html string) string {
	cleaned := html
	for _, re := range nonContentTagBlockRes {
		cleaned = re.ReplaceAllString(cleaned, " ")
	}
	return stripHtmlTags(cleaned)
}

func failureResponse(status int, finalURL, contentType string) FetchResult {
	return FetchResult{OK: false, Status: status, PageText: "", FinalURL: finalURL, ContentType: contentType}
}

func parseIPv4(address string) []int {
	parts := strings.Split(address, ".")
	if len(parts) != 4 {
		return nil
	}
	octets := make([]int, 4)
	for i, p := range parts {
		var n int
		for _, c := range p {
			if c < '0' || c > '9' {
				return nil
			}
			n = n*10 + int(c-'0')
		}
		if n < 0 || n > 255 {
			return nil
		}
		octets[i] = n
	}
	return octets
}

func isDisallowedIPv4(address string) bool {
	octets := parseIPv4(address)
	if octets == nil {
		return false
	}
	first, second := octets[0], octets[1]
	if first == 0 || first == 10 || first == 127 {
		return true
	}
	if first == 100 && second >= 64 && second <= 127 {
		return true
	}
	if first == 169 && second == 254 {
		return true
	}
	if first == 172 && second >= 16 && second <= 31 {
		return true
	}
	if first == 198 && (second == 18 || second == 19) {
		return true
	}
	if first == 192 && second == 168 {
		return true
	}
	return false
}

func isDisallowedIPv6(address string) bool {
	normalized := strings.ToLower(address)
	// strip zone id
	if idx := strings.Index(normalized, "%"); idx >= 0 {
		normalized = normalized[:idx]
	}
	if normalized == "" {
		return false
	}
	if normalized == "::1" {
		return true
	}
	if strings.HasPrefix(normalized, "fc") || strings.HasPrefix(normalized, "fd") {
		return true
	}
	if linkLocalIPv6Re.MatchString(normalized) {
		return true
	}
	if strings.HasPrefix(normalized, "::ffff:") {
		mapped := normalized[len("::ffff:"):]
		return isDisallowedIPv4(mapped)
	}
	return false
}

func isDisallowedIP(address string) bool {
	if net.ParseIP(address) == nil {
		return false
	}
	if strings.Contains(address, ":") {
		return isDisallowedIPv6(address)
	}
	return isDisallowedIPv4(address)
}

func isLocalhostHostname(hostname string) bool {
	h := strings.ToLower(hostname)
	return h == "localhost" || strings.HasSuffix(h, ".localhost")
}

// LookupIPFunc is a function that resolves a hostname to IP addresses.
type LookupIPFunc func(hostname string) ([]string, error)

func defaultLookupIP(hostname string) ([]string, error) {
	addrs, err := net.LookupHost(hostname)
	return addrs, err
}

func safeTransport() *http.Transport {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}
			if tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok && isDisallowedIP(tcpAddr.IP.String()) {
				_ = conn.Close()
				return nil, &net.OpError{Op: "dial", Net: network, Addr: tcpAddr, Err: errDisallowedIP{ip: tcpAddr.IP.String()}}
			}
			return conn, nil
		},
	}
}

type errDisallowedIP struct{ ip string }

func (e errDisallowedIP) Error() string { return "disallowed target IP: " + e.ip }

func validateTarget(rawURL string, lookupIP LookupIPFunc) (bool, *url.URL, int) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false, nil, 400
	}

	hostname := strings.ToLower(parsed.Hostname())
	if isLocalhostHostname(hostname) || isDisallowedIP(hostname) {
		return false, parsed, 403
	}

	if net.ParseIP(hostname) == nil {
		// hostname – need DNS check
		ips, err := lookupIP(hostname)
		if err != nil {
			return false, parsed, 403
		}
		if len(ips) == 0 {
			return false, parsed, 403
		}
		for _, ip := range ips {
			if isDisallowedIP(ip) {
				return false, parsed, 403
			}
		}
	}

	return true, parsed, 200
}

func isRedirectStatus(status int) bool {
	for _, s := range []int{301, 302, 303, 307, 308} {
		if status == s {
			return true
		}
	}
	return false
}

// FetchPageContent fetches a URL and returns extracted page text.
// Mirrors fetchPageContent() from content-fetcher.js.
// Pass nil for client to use the default http.Client.
func FetchPageContent(ctx context.Context, rawURL string) (FetchResult, error) {
	return fetchPageContentWithDeps(ctx, rawURL, nil, defaultLookupIP)
}

// FetchPageContentWithDeps is the testable variant that accepts injectable deps.
func fetchPageContentWithDeps(ctx context.Context, rawURL string, client *http.Client, lookupIP LookupIPFunc) (FetchResult, error) {
	inputURL := strings.TrimSpace(rawURL)
	if inputURL == "" {
		return failureResponse(400, "", ""), nil
	}

	if client == nil {
		client = &http.Client{
			Timeout:   time.Duration(defaultTimeoutMs) * time.Millisecond,
			Transport: safeTransport(),
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	ok, parsed, code := validateTarget(inputURL, lookupIP)
	if !ok {
		return failureResponse(code, inputURL, ""), nil
	}
	currentURL := parsed.String()

	for attempt := 0; attempt <= defaultMaxRedirects; attempt++ {
		var resp *http.Response
		var lastErr error

		for retry := 0; retry <= defaultMaxRetries; retry++ {
			okCurrent, parsedCurrent, codeCurrent := validateTarget(currentURL, lookupIP)
			if !okCurrent {
				return failureResponse(codeCurrent, currentURL, ""), nil
			}
			currentURL = parsedCurrent.String()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, currentURL, nil)
			if err != nil {
				return failureResponse(0, currentURL, ""), nil
			}
			req.Header.Set("User-Agent", defaultUserAgent)
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
			req.Header.Set("Accept-Language", "en-US,en;q=0.9")

			resp, lastErr = client.Do(req)
			if lastErr != nil {
				if retry < defaultMaxRetries {
					time.Sleep(time.Duration(retryBackoffMs*(retry+1)) * time.Millisecond)
					continue
				}
				return failureResponse(0, currentURL, ""), nil
			}

			if retry < defaultMaxRetries && transientStatuses[resp.StatusCode] {
				resp.Body.Close()
				time.Sleep(time.Duration(retryBackoffMs*(retry+1)) * time.Millisecond)
				resp = nil
				continue
			}
			break
		}

		if resp == nil && lastErr != nil {
			return failureResponse(0, currentURL, ""), nil
		}
		if resp == nil {
			return failureResponse(0, currentURL, ""), nil
		}

		contentType := strings.ToLower(resp.Header.Get("Content-Type"))
		var responseURL string
		if resp.Request != nil && resp.Request.URL != nil {
			responseURL = resp.Request.URL.String()
		}
		if responseURL == "" {
			responseURL = currentURL
		}

		// validate final URL
		okFinal, _, codeFinal := validateTarget(responseURL, lookupIP)
		if !okFinal {
			resp.Body.Close()
			return failureResponse(codeFinal, responseURL, contentType), nil
		}

		if !isRedirectStatus(resp.StatusCode) {
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				resp.Body.Close()
				return failureResponse(resp.StatusCode, responseURL, contentType), nil
			}

			var pageText string
			if strings.Contains(contentType, "text/html") {
				body, err := io.ReadAll(io.LimitReader(resp.Body, maxPageBytes+1))
				resp.Body.Close()
				if err == nil && int64(len(body)) <= maxPageBytes {
					pageText = extractPageText(string(body))
				}
			} else {
				resp.Body.Close()
			}

			return FetchResult{
				OK:          true,
				Status:      resp.StatusCode,
				PageText:    pageText,
				FinalURL:    responseURL,
				ContentType: contentType,
			}, nil
		}

		// redirect
		location := strings.TrimSpace(resp.Header.Get("Location"))
		resp.Body.Close()
		if location == "" {
			return failureResponse(resp.StatusCode, responseURL, contentType), nil
		}

		if attempt >= defaultMaxRedirects {
			return failureResponse(508, responseURL, contentType), nil
		}

		redirectURL, err := url.Parse(location)
		if err != nil {
			return failureResponse(400, responseURL, contentType), nil
		}
		base, _ := url.Parse(responseURL)
		resolved := base.ResolveReference(redirectURL).String()

		okRedir, parsedRedir, codeRedir := validateTarget(resolved, lookupIP)
		if !okRedir {
			return failureResponse(codeRedir, resolved, ""), nil
		}
		currentURL = parsedRedir.String()
	}

	return failureResponse(508, currentURL, ""), nil
}
