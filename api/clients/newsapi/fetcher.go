package newsapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// ArticleFetcher downloads an article URL and converts the HTML body to
// markdown so the text can be fed cleanly to an LLM. Conversion is best-effort:
// many sites return 403 to bot user-agents or paywall the full article; the
// fetcher degrades gracefully (returns empty markdown + the underlying error).
type ArticleFetcher struct {
	httpClient *http.Client
	maxBytes   int64
	charLimit  int
	userAgent  string
}

// FetcherOption configures an ArticleFetcher.
type FetcherOption func(*ArticleFetcher)

// WithFetcherTimeout sets the per-request timeout. Default 6s.
func WithFetcherTimeout(d time.Duration) FetcherOption {
	return func(f *ArticleFetcher) { f.httpClient = &http.Client{Timeout: d} }
}

// WithFetcherMaxBytes caps how much HTML we'll pull off the wire per article.
// Defaults to 1 MiB — covers most articles while keeping memory bounded.
func WithFetcherMaxBytes(n int64) FetcherOption {
	return func(f *ArticleFetcher) { f.maxBytes = n }
}

// WithFetcherCharLimit caps the markdown length handed back to callers.
// Defaults to 8000 — roomy for a summarizer but cheap to pass around.
func WithFetcherCharLimit(n int) FetcherOption {
	return func(f *ArticleFetcher) { f.charLimit = n }
}

// WithFetcherUserAgent overrides the HTTP User-Agent header. Some sites
// accept a real-looking browser UA and reject bot-like UAs. Default is a
// desktop Chrome UA which gets the highest success rate.
func WithFetcherUserAgent(ua string) FetcherOption {
	return func(f *ArticleFetcher) { f.userAgent = ua }
}

// NewArticleFetcher builds a fetcher with sensible defaults.
func NewArticleFetcher(opts ...FetcherOption) *ArticleFetcher {
	f := &ArticleFetcher{
		httpClient: &http.Client{Timeout: 6 * time.Second},
		maxBytes:   1 << 20,
		charLimit:  8000,
		userAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
	}
	for _, o := range opts {
		o(f)
	}
	return f
}

// Fetch downloads u and returns its body converted to markdown, truncated to
// the configured char limit. A non-2xx response returns ErrFetch*. An error
// in html-to-markdown conversion returns the raw text instead of failing —
// the LLM can still work with plain text.
func (f *ArticleFetcher) Fetch(ctx context.Context, u string) (string, error) {
	if strings.TrimSpace(u) == "" {
		return "", fmt.Errorf("fetcher: url is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetcher: get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", &FetchError{URL: u, Status: resp.StatusCode}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, f.maxBytes))
	if err != nil {
		return "", fmt.Errorf("fetcher: read: %w", err)
	}
	md, err := htmltomarkdown.ConvertString(string(body))
	if err != nil || strings.TrimSpace(md) == "" {
		// Fall back to the raw body — it's still usable for summarization.
		md = string(body)
	}
	md = collapseBlankLines(md)
	return Truncate(md, f.charLimit), nil
}

// FetchError represents a non-2xx HTTP response from an article source.
type FetchError struct {
	URL    string
	Status int
}

func (e *FetchError) Error() string { return fmt.Sprintf("fetcher: http %d for %s", e.Status, e.URL) }

// collapseBlankLines squashes runs of 3+ newlines down to exactly 2.
// html-to-markdown can emit long vertical gaps from nested block elements.
func collapseBlankLines(s string) string {
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(s)
}
