// Package wikipedia implements a focused Wikipedia client.
//
// The client is "smart" in three ways: (1) it uses the MediaWiki opensearch
// endpoint to resolve a free-text query to the best-matching page title,
// then fetches the page summary via the REST API — recovering gracefully
// when a title doesn't exist as-is; (2) it normalizes the summary into a
// flat Result with a single Extract blob truncated to a configurable
// character budget (default 5000); (3) it treats disambiguation pages as a
// soft error by returning the candidate titles so callers (the LLM) can
// choose.
package wikipedia

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultActionBase  = "https://en.wikipedia.org/w/api.php"
	defaultRESTBase    = "https://en.wikipedia.org/api/rest_v1"
	defaultUserAgent   = "qodo-marvin/1.0 (contact: jesse.l@qodo.ai)"
	defaultTimeout     = 10 * time.Second
	defaultCharBudget  = 5000
)

// Client fetches Wikipedia content.
type Client struct {
	actionBase string
	restBase   string
	userAgent  string
	httpClient *http.Client
	charBudget int
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURLs overrides the action+REST base URLs (mainly for tests).
// Pass the httptest server URL as both arguments when stubbing.
func WithBaseURLs(action, rest string) Option {
	return func(c *Client) { c.actionBase = action; c.restBase = rest }
}

// WithHTTPClient overrides the HTTP client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }

// WithCharBudget sets the Extract truncation cap. Defaults to 5000; <=0 disables.
func WithCharBudget(n int) Option { return func(c *Client) { c.charBudget = n } }

// WithUserAgent overrides the HTTP User-Agent (Wikipedia requires one).
func WithUserAgent(ua string) Option { return func(c *Client) { c.userAgent = ua } }

// New constructs a default Client.
func New(opts ...Option) *Client {
	c := &Client{
		actionBase: defaultActionBase,
		restBase:   defaultRESTBase,
		userAgent:  defaultUserAgent,
		httpClient: &http.Client{Timeout: defaultTimeout},
		charBudget: defaultCharBudget,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Result is the normalized shape returned from Search.
type Result struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Extract     string `json:"extract"` // truncated plain-text summary
	URL         string `json:"url"`
	// If the matched page is a disambiguation page, Candidates will be
	// populated with alternative titles and Extract will describe the
	// situation rather than being substantive content.
	Candidates []string `json:"candidates,omitempty"`
}

// Search resolves `query` to a Wikipedia page and returns a truncated summary.
// Errors: ErrNotFound if no page matches; APIError for non-2xx HTTP.
func (c *Client) Search(ctx context.Context, query string) (*Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("wikipedia: query is required")
	}

	title, err := c.resolveTitle(ctx, query)
	if err != nil {
		return nil, err
	}
	return c.Summary(ctx, title)
}

// Summary fetches the REST /page/summary/{title} for an exact title.
// Disambiguation results are surfaced via Result.Candidates.
func (c *Client) Summary(ctx context.Context, title string) (*Result, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("wikipedia: title is required")
	}
	u := c.restBase + "/page/summary/" + url.PathEscape(title)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wikipedia: summary request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode >= 400 {
		return nil, &APIError{Status: resp.StatusCode, Message: truncateForError(string(body))}
	}
	var s summaryResponse
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, fmt.Errorf("wikipedia: decode summary: %w", err)
	}

	r := &Result{
		Title:       s.Title,
		Description: strings.TrimSpace(s.Description),
		Extract:     Truncate(strings.TrimSpace(s.Extract), c.charBudget),
		URL:         s.ContentURLs.Desktop.Page,
	}
	// REST summary returns type="disambiguation" when the page is one.
	if strings.EqualFold(s.Type, "disambiguation") {
		cands, _ := c.disambiguationLinks(ctx, title)
		r.Candidates = cands
		if r.Extract == "" {
			r.Extract = fmt.Sprintf("%q is a disambiguation page. Candidates: %s", title, strings.Join(cands, ", "))
		}
	}
	return r, nil
}

// resolveTitle uses opensearch to pick the most relevant title for a query.
func (c *Client) resolveTitle(ctx context.Context, query string) (string, error) {
	q := url.Values{}
	q.Set("action", "opensearch")
	q.Set("search", query)
	q.Set("limit", "1")
	q.Set("namespace", "0")
	q.Set("format", "json")
	u := c.actionBase + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("wikipedia: opensearch request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", &APIError{Status: resp.StatusCode, Message: truncateForError(string(body))}
	}

	// opensearch returns [query, [titles...], [descriptions...], [urls...]]
	var arr []json.RawMessage
	if err := json.Unmarshal(body, &arr); err != nil {
		return "", fmt.Errorf("wikipedia: decode opensearch: %w (body=%q)", err, truncateForError(string(body)))
	}
	if len(arr) < 2 {
		return "", ErrNotFound
	}
	var titles []string
	if err := json.Unmarshal(arr[1], &titles); err != nil {
		return "", fmt.Errorf("wikipedia: decode opensearch titles: %w", err)
	}
	if len(titles) == 0 {
		return "", ErrNotFound
	}
	return titles[0], nil
}

// disambiguationLinks best-effort pulls ~10 alternative titles off a disambig page.
func (c *Client) disambiguationLinks(ctx context.Context, title string) ([]string, error) {
	q := url.Values{}
	q.Set("action", "query")
	q.Set("titles", title)
	q.Set("prop", "links")
	q.Set("pllimit", "10")
	q.Set("plnamespace", "0")
	q.Set("format", "json")
	u := c.actionBase + "?" + q.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var r struct {
		Query struct {
			Pages map[string]struct {
				Links []struct{ Title string } `json:"links"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	out := make([]string, 0, 10)
	for _, page := range r.Query.Pages {
		for _, l := range page.Links {
			if l.Title != "" {
				out = append(out, l.Title)
			}
		}
	}
	return out, nil
}

// ErrNotFound is returned when a search yields no page.
var ErrNotFound = errors.New("wikipedia: no matching page")

// APIError represents a non-2xx response.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("wikipedia: %s (http %d)", e.Message, e.Status) }

// Truncate clamps s to maxChars (runes), appending "…[truncated]" when cut.
func Truncate(s string, maxChars int) string {
	if maxChars <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	const suffix = "…[truncated]"
	if maxChars <= len([]rune(suffix)) {
		return string(runes[:maxChars])
	}
	return string(runes[:maxChars-len([]rune(suffix))]) + suffix
}

func truncateForError(s string) string {
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}

type summaryResponse struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Extract     string `json:"extract"`
	ContentURLs struct {
		Desktop struct {
			Page string `json:"page"`
		} `json:"desktop"`
	} `json:"content_urls"`
}
