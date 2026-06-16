// Package pipeline contains Scout data-fetching helpers that mirror the Express pipeline.
package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// Mimic a real Chrome browser to avoid Reddit's bot detection.
	redditUserAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	redditQueryDelay = 2500 * time.Millisecond
	redditMaxRetries = 2
	// Re-harvest cookies after this duration (token_v2 lasts ~24h; refresh earlier).
	redditCookieTTL = 20 * time.Hour
)

// RedditComment is a single top-level comment returned by FetchRedditContext.
type RedditComment struct {
	Author string `json:"author"`
	Body   string `json:"body"`
	Score  int    `json:"score"`
}

// RedditContext holds the full post body and top comments retrieved from Reddit.
type RedditContext struct {
	FullBody    string
	TopComments []RedditComment
}

// redditSession holds a cookie jar seeded by an anonymous visit to reddit.com.
type redditSession struct {
	mu          sync.Mutex
	jar         http.CookieJar
	harvestedAt time.Time
}

const redditFetchMinInterval = 1500 * time.Millisecond

var (
	globalRedditSession = &redditSession{}
	// Separate client without a jar used only for the harvest request.
	redditHarvestClient  = &http.Client{Timeout: 15 * time.Second}
	lastRedditFetch      time.Time
	lastRedditFetchMu    sync.Mutex
)

// redditClient returns an http.Client with a fresh-enough cookie jar.
// If the jar is stale (or missing), it harvests fresh anonymous cookies first.
func redditClient(ctx context.Context) (*http.Client, error) {
	globalRedditSession.mu.Lock()
	defer globalRedditSession.mu.Unlock()

	if globalRedditSession.jar == nil || time.Since(globalRedditSession.harvestedAt) > redditCookieTTL {
		if err := harvestRedditCookies(ctx); err != nil {
			return nil, err
		}
	}
	return &http.Client{
		Timeout: 15 * time.Second,
		Jar:     globalRedditSession.jar,
	}, nil
}

// harvestRedditCookies solves Reddit's JS challenge to obtain a valid session.
//
// Reddit serves a trivial JS proof-of-work on first visit from unknown IPs:
//   1. GET reddit.com → sets edgebucket cookie, returns HTML with a hidden form
//      containing a `token` and an inline script that computes `solution` = nonce+nonce.
//   2. GET /?solution=<nonce+nonce>&js_challenge=1&token=<token> → 302, sets rdt cookie.
//   3. Follow redirect → session accepted; subsequent .json API calls succeed.
//
// Must be called with globalRedditSession.mu held.
func harvestRedditCookies(ctx context.Context) error {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("reddit cookie jar: %w", err)
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
		Jar:     jar,
		// Don't auto-follow redirects; we handle them manually.
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Step 1: GET reddit.com — collect edgebucket + parse challenge fields.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.reddit.com/", nil)
	if err != nil {
		return fmt.Errorf("reddit harvest step1 request: %w", err)
	}
	redditSetBrowserHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("reddit harvest step1 fetch: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	html := string(body)
	log.Printf("[reddit] harvest step1 status %d, cookies %d", resp.StatusCode, len(jar.Cookies(mustParseURL("https://www.reddit.com"))))

	// If no challenge page (no form with js_challenge), we're already good.
	if !strings.Contains(html, "js_challenge") {
		globalRedditSession.jar = jar
		globalRedditSession.harvestedAt = time.Now()
		log.Printf("[reddit] no challenge — session ready")
		return nil
	}

	// Extract token from: <input type="hidden" name="token" value="..."/>
	token := extractAttr(html, `name="token"`, "value")
	// Extract nonce from: await(async e=>e+e)("<nonce>")
	nonce := extractNonce(html)
	if token == "" || nonce == "" {
		return fmt.Errorf("reddit harvest: could not parse challenge (token=%q nonce=%q)", token, nonce)
	}
	solution := nonce + nonce

	log.Printf("[reddit] challenge: nonce=%s token=%s...", nonce, token[:min(8, len(token))])

	// Step 2: Submit challenge solution.
	challengeURL := fmt.Sprintf("https://www.reddit.com/?solution=%s&js_challenge=1&token=%s&jsc_orig_r=", solution, token)
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, challengeURL, nil)
	if err != nil {
		return fmt.Errorf("reddit harvest step2 request: %w", err)
	}
	redditSetBrowserHeaders(req2)
	req2.Header.Set("Referer", "https://www.reddit.com/")
	req2.Header.Set("Sec-Fetch-Site", "same-origin")

	resp2, err := client.Do(req2)
	if err != nil {
		return fmt.Errorf("reddit harvest step2 fetch: %w", err)
	}
	resp2.Body.Close()
	log.Printf("[reddit] challenge step2 status %d, cookies %d", resp2.StatusCode, len(jar.Cookies(mustParseURL("https://www.reddit.com"))))

	// Step 3: Follow redirect to finalise session.
	loc := resp2.Header.Get("Location")
	if loc == "" {
		loc = "https://www.reddit.com/"
	}
	if !strings.HasPrefix(loc, "http") {
		loc = "https://www.reddit.com" + loc
	}
	req3, err := http.NewRequestWithContext(ctx, http.MethodGet, loc, nil)
	if err != nil {
		return fmt.Errorf("reddit harvest step3 request: %w", err)
	}
	redditSetBrowserHeaders(req3)
	req3.Header.Set("Referer", "https://www.reddit.com/")
	req3.Header.Set("Sec-Fetch-Site", "same-origin")

	resp3, err := client.Do(req3)
	if err != nil {
		return fmt.Errorf("reddit harvest step3 fetch: %w", err)
	}
	resp3.Body.Close()

	u := mustParseURL("https://www.reddit.com")
	cookies := jar.Cookies(u)
	log.Printf("[reddit] harvest complete: %d cookies (step3 status %d)", len(cookies), resp3.StatusCode)

	globalRedditSession.jar = jar
	globalRedditSession.harvestedAt = time.Now()
	return nil
}

// extractAttr finds the value of an HTML attribute near a marker string.
// e.g. extractAttr(html, `name="token"`, "value") returns the value of value="..." on that element.
func extractAttr(html, marker, attr string) string {
	i := strings.Index(html, marker)
	if i < 0 {
		return ""
	}
	// Search backwards for the opening < of the element.
	start := strings.LastIndex(html[:i], "<")
	if start < 0 {
		return ""
	}
	// Find the end of the element.
	end := strings.Index(html[start:], ">")
	if end < 0 {
		return ""
	}
	elem := html[start : start+end+1]
	key := attr + `="`
	j := strings.Index(elem, key)
	if j < 0 {
		return ""
	}
	rest := elem[j+len(key):]
	k := strings.Index(rest, `"`)
	if k < 0 {
		return ""
	}
	return rest[:k]
}

// extractNonce pulls the nonce string from Reddit's inline challenge script.
// The pattern is: await(async e=>e+e)("<nonce>")
func extractNonce(html string) string {
	marker := `await(async e=>e+e)(`
	i := strings.Index(html, marker)
	if i < 0 {
		return ""
	}
	rest := html[i+len(marker):]
	// Strip leading quote char (' or ")
	if len(rest) == 0 {
		return ""
	}
	quote := string(rest[0])
	rest = rest[1:]
	j := strings.Index(rest, quote)
	if j < 0 {
		return ""
	}
	return rest[:j]
}

func mustParseURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}



// redditSetBrowserHeaders sets the headers Reddit expects from a real browser.
func redditSetBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", redditUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}

// redditFetchWithRetry fetches a URL with retry logic.
// On a 403 it re-harvests cookies once and retries before giving up.
func redditFetchWithRetry(ctx context.Context, fetchURL string) ([]byte, error) {
	cookieRefreshed := false
	for attempt := 0; attempt <= redditMaxRetries; attempt++ {
		client, err := redditClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("reddit session: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		redditSetBrowserHeaders(req)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch: %w", err)
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read body: %w", readErr)
		}

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		// On 403: force a cookie re-harvest once, then retry immediately.
		if resp.StatusCode == http.StatusForbidden && !cookieRefreshed {
			log.Printf("[reddit] 403 on attempt %d — re-harvesting cookies", attempt)
			globalRedditSession.mu.Lock()
			globalRedditSession.jar = nil // force refresh
			globalRedditSession.mu.Unlock()
			cookieRefreshed = true
			continue
		}

		if (resp.StatusCode == 429 || resp.StatusCode == 503) && attempt < redditMaxRetries {
			delay := time.Duration(1<<uint(attempt)) * 5 * time.Second
			if resp.StatusCode == 429 {
				if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
					if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 && seconds <= 120 {
						delay = time.Duration(seconds) * time.Second
					}
				}
				log.Printf("[reddit] 429 on attempt %d, waiting %s before retry", attempt, delay)
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			continue
		}

		return nil, fmt.Errorf("Reddit fetch failed: %d for %s", resp.StatusCode, fetchURL)
	}
	return nil, fmt.Errorf("Reddit fetch failed after retries: %s", fetchURL)
}

// redditHTTPClient is kept for use in tests.
var redditHTTPClient = &http.Client{Timeout: 15 * time.Second}

// FetchRedditPosts fetches posts from an array of search query URLs.
// It deduplicates across queries by post name.
// Mirrors fetchRedditPosts() from apps/api/server/pipeline/reddit.js.
func FetchRedditPosts(ctx context.Context, queryURLs []string, onProgress func(done int, total int)) ([]FetchedPost, error) {
	seen := make(map[string]FetchedPost)
	order := make([]string, 0)

	for i, url := range queryURLs {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(redditQueryDelay):
			}
		}

		body, err := redditFetchWithRetry(ctx, url)
		if err != nil {
			log.Printf("[reddit] Failed to fetch %s: %v", url, err)
		} else {
			var listing struct {
				Data struct {
					Children []struct {
						Data struct {
							Name        string  `json:"name"`
							Title       string  `json:"title"`
							Selftext    string  `json:"selftext"`
							Author      string  `json:"author"`
							Permalink   string  `json:"permalink"`
							Subreddit   string  `json:"subreddit"`
							Score       int     `json:"score"`
							NumComments int     `json:"num_comments"`
							CreatedUTC  float64 `json:"created_utc"`
							URL         string  `json:"url"`
						} `json:"data"`
					} `json:"children"`
				} `json:"data"`
			}
			if jsonErr := json.Unmarshal(body, &listing); jsonErr == nil {
				for _, child := range listing.Data.Children {
					d := child.Data
					if d.Name == "" {
						continue
					}
					if _, exists := seen[d.Name]; !exists {
						seen[d.Name] = FetchedPost{
							ID:          d.Name,
							Title:       d.Title,
							Selftext:    d.Selftext,
							Author:      d.Author,
							Permalink:   d.Permalink,
							Subreddit:   d.Subreddit,
							Score:       d.Score,
							NumComments: d.NumComments,
							CreatedUTC:  d.CreatedUTC,
							URL:         d.URL,
						}
						order = append(order, d.Name)
					}
				}
			}
		}

		if onProgress != nil {
			onProgress(i+1, len(queryURLs))
		}
	}

	posts := make([]FetchedPost, 0, len(order))
	for _, name := range order {
		posts = append(posts, seen[name])
	}
	return posts, nil
}

// FetchRedditContext fetches the full post body and top 5 comments for a Reddit
// post URL (permalink path, e.g. "/r/foo/comments/…").  It mirrors the Express
// fetchRedditContext() from apps/api/server/pipeline/reddit.js.
//
// On network failure the function returns an empty RedditContext and the error
// so that callers can decide whether to abort (500) or continue with partial data.
func FetchRedditContext(ctx context.Context, postURL string) (RedditContext, error) {
	if err := redditThrottle(ctx); err != nil {
		return RedditContext{}, err
	}

	// Build the JSON endpoint URL.
	// postURL may be a full https URL or a /r/… permalink path.
	var jsonURL string
	if strings.HasPrefix(postURL, "http") {
		// Strip trailing slash, append .json
		trimmed := strings.TrimRight(postURL, "/")
		jsonURL = trimmed + ".json?limit=5&depth=1&sort=top"
	} else {
		// Treat as a Reddit permalink path.
		trimmed := strings.TrimRight(postURL, "/")
		jsonURL = "https://www.reddit.com" + trimmed + ".json?limit=5&depth=1&sort=top"
	}

	body, err := redditFetchWithRetry(ctx, jsonURL)
	if err != nil {
		return RedditContext{}, fmt.Errorf("reddit context: %w", err)
	}

	// Reddit returns a 2-element JSON array: [postListing, commentListing].
	var listings []json.RawMessage
	if err := json.Unmarshal(body, &listings); err != nil || len(listings) < 2 {
		return RedditContext{}, fmt.Errorf("reddit context: unexpected response format")
	}

	// Extract full body from post listing.
	var postListing struct {
		Data struct {
			Children []struct {
				Data struct {
					Selftext string `json:"selftext"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	_ = json.Unmarshal(listings[0], &postListing)
	fullBody := ""
	if len(postListing.Data.Children) > 0 {
		fullBody = postListing.Data.Children[0].Data.Selftext
	}

	// Extract top comments from comment listing.
	var commentListing struct {
		Data struct {
			Children []struct {
				Kind string `json:"kind"`
				Data struct {
					Author string `json:"author"`
					Body   string `json:"body"`
					Score  int    `json:"score"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	_ = json.Unmarshal(listings[1], &commentListing)

	var topComments []RedditComment
	for _, child := range commentListing.Data.Children {
		if child.Kind != "t1" {
			continue
		}
		body := child.Data.Body
		if len(body) > 500 {
			body = body[:500]
		}
		topComments = append(topComments, RedditComment{
			Author: child.Data.Author,
			Body:   body,
			Score:  child.Data.Score,
		})
		if len(topComments) >= 5 {
			break
		}
	}
	if topComments == nil {
		topComments = []RedditComment{}
	}

	return RedditContext{FullBody: fullBody, TopComments: topComments}, nil
}
