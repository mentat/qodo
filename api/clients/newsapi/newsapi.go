// Package newsapi implements a focused client for newsapi.org.
//
// The client is "smart" in three ways: (1) it enforces per-source content
// truncation so callers can never exceed an LLM's input budget, (2) it
// normalizes NewsAPI's fiddly response shape into a flat Article struct with
// a single text blob per article, and (3) it maps NewsAPI's non-2xx JSON
// error responses into structured Go errors.
package newsapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURL     = "https://newsapi.org/v2"
	defaultTimeout     = 10 * time.Second
	defaultPageSize    = 5
	maxPageSize        = 20
	defaultCharBudget  = 5000
	defaultLanguage    = ""
	defaultSortBy      = "relevancy"
)

// Client is a NewsAPI client.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	charBudget int
	// Optional enrichments. If fetcher is set, Search populates Article.Markdown
	// by downloading each article URL and converting the HTML to markdown. If
	// summarizer is set, Search populates Article.Summary from either the
	// fetched markdown or the NewsAPI snippet. Enrichments run in parallel
	// per-article so a search of 5 articles costs ~one fetcher/summarizer
	// round-trip, not 5.
	fetcher    *ArticleFetcher
	summarizer *ArticleSummarizer
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL (mainly for tests).
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }

// WithHTTPClient overrides the HTTP client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }

// WithCharBudget overrides the per-article character budget (content + description).
// Defaults to 5000; pass <=0 to disable truncation.
func WithCharBudget(n int) Option { return func(c *Client) { c.charBudget = n } }

// WithFetcher enables full-article HTML scraping + markdown conversion.
// When enabled, Article.Markdown is populated for each result (best-effort —
// fetch failures leave Markdown empty but never fail the whole search).
func WithFetcher(f *ArticleFetcher) Option { return func(c *Client) { c.fetcher = f } }

// WithSummarizer enables Gemini-powered per-article summaries.
// When enabled, Article.Summary is populated for each result. Best used in
// combination with WithFetcher so the summarizer sees full article bodies
// instead of NewsAPI's ~260-char truncation.
func WithSummarizer(s *ArticleSummarizer) Option { return func(c *Client) { c.summarizer = s } }

// New constructs a Client. apiKey is required.
func New(apiKey string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("newsapi: api key is required")
	}
	c := &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		charBudget: defaultCharBudget,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// Article is the normalized shape returned to callers.
type Article struct {
	Title       string    `json:"title"`
	Source      string    `json:"source"`
	URL         string    `json:"url"`
	Author      string    `json:"author,omitempty"`
	PublishedAt time.Time `json:"publishedAt"`
	// Text merges NewsAPI's description + content snippet and is truncated
	// to the client's character budget. This is always present and is the
	// fallback when Markdown/Summary aren't available.
	Text string `json:"text"`
	// Markdown is the scraped article body converted to markdown. Populated
	// when the client was built with WithFetcher. Empty on scrape failure.
	Markdown string `json:"markdown,omitempty"`
	// Summary is a 2–4 sentence LLM-generated summary. Populated when the
	// client was built with WithSummarizer. Empty on summarization failure.
	Summary string `json:"summary,omitempty"`
}

// SearchParams controls the /v2/everything query. Only Query is required.
type SearchParams struct {
	Query    string
	Language string // e.g. "en"; empty means any
	SortBy   string // "relevancy" (default), "popularity", "publishedAt"
	PageSize int    // default 5, capped at 20 (keeps context small for LLMs)
	From     time.Time
	To       time.Time
	Domains  []string
}

// Search calls /v2/everything. Returns a bounded list of normalized articles.
func (c *Client) Search(ctx context.Context, p SearchParams) ([]Article, error) {
	if strings.TrimSpace(p.Query) == "" {
		return nil, errors.New("newsapi: query is required")
	}
	page := p.PageSize
	if page <= 0 {
		page = defaultPageSize
	}
	if page > maxPageSize {
		page = maxPageSize
	}
	sortBy := p.SortBy
	if sortBy == "" {
		sortBy = defaultSortBy
	}

	q := url.Values{}
	q.Set("q", p.Query)
	q.Set("sortBy", sortBy)
	q.Set("pageSize", strconv.Itoa(page))
	if p.Language != "" {
		q.Set("language", p.Language)
	}
	if !p.From.IsZero() {
		q.Set("from", p.From.UTC().Format(time.RFC3339))
	}
	if !p.To.IsZero() {
		q.Set("to", p.To.UTC().Format(time.RFC3339))
	}
	if len(p.Domains) > 0 {
		q.Set("domains", strings.Join(p.Domains, ","))
	}

	u := c.baseURL + "/everything?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "qodo-marvin/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("newsapi: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("newsapi: read body: %w", err)
	}

	// NewsAPI uses 200 for "ok" and 4xx/5xx for errors, both with JSON bodies.
	if resp.StatusCode >= 400 {
		var e errorResponse
		_ = json.Unmarshal(body, &e)
		if e.Message != "" {
			return nil, &APIError{Status: resp.StatusCode, Code: e.Code, Message: e.Message}
		}
		return nil, &APIError{Status: resp.StatusCode, Message: string(body)}
	}

	var raw rawResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("newsapi: decode: %w (body=%q)", err, truncateForError(string(body)))
	}
	if raw.Status != "" && raw.Status != "ok" {
		return nil, &APIError{Status: resp.StatusCode, Code: raw.Code, Message: raw.Message}
	}

	out := make([]Article, len(raw.Articles))
	for i, a := range raw.Articles {
		out[i] = c.normalize(a)
	}
	c.enrich(ctx, out)
	return out, nil
}

// enrich runs the optional fetcher + summarizer across all articles in
// parallel. Each article's enrichment is independent so a stuck site can't
// stall the whole result set (the fetcher has its own timeout).
func (c *Client) enrich(ctx context.Context, arts []Article) {
	if c.fetcher == nil && c.summarizer == nil {
		return
	}
	var wg sync.WaitGroup
	wg.Add(len(arts))
	for i := range arts {
		go func(i int) {
			defer wg.Done()
			art := &arts[i]
			// 1) Scrape + convert HTML to markdown (if fetcher enabled).
			if c.fetcher != nil && art.URL != "" {
				if md, err := c.fetcher.Fetch(ctx, art.URL); err == nil && md != "" {
					art.Markdown = md
				}
			}
			// 2) Summarize via Gemini (if summarizer enabled).
			if c.summarizer != nil {
				body := art.Markdown
				if body == "" {
					body = art.Text
				}
				if s, err := c.summarizer.Summarize(ctx, art.Title, body); err == nil && s != "" {
					art.Summary = s
				}
			}
		}(i)
	}
	wg.Wait()
}

func (c *Client) normalize(r rawArticle) Article {
	text := strings.TrimSpace(r.Description)
	if r.Content != "" {
		// NewsAPI content is often truncated with "[+1234 chars]" — strip it.
		content := stripContentTrailer(r.Content)
		if text != "" && !strings.Contains(text, content) {
			text = text + "\n\n" + content
		} else if text == "" {
			text = content
		}
	}
	text = Truncate(text, c.charBudget)
	var pub time.Time
	if r.PublishedAt != "" {
		if t, err := time.Parse(time.RFC3339, r.PublishedAt); err == nil {
			pub = t
		}
	}
	return Article{
		Title:       strings.TrimSpace(r.Title),
		Source:      strings.TrimSpace(r.Source.Name),
		URL:         strings.TrimSpace(r.URL),
		Author:      strings.TrimSpace(r.Author),
		PublishedAt: pub,
		Text:        text,
	}
}

// Truncate returns s clamped to maxChars, appending "…[truncated]" when cut.
// Safe for multibyte strings: trims at rune boundary.
// If maxChars <= 0 the input is returned unchanged.
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

// stripContentTrailer removes NewsAPI's `… [+1234 chars]` markers.
func stripContentTrailer(s string) string {
	idx := strings.LastIndex(s, "[+")
	if idx < 0 {
		return s
	}
	end := strings.Index(s[idx:], "]")
	if end < 0 {
		return s
	}
	return strings.TrimSpace(s[:idx])
}

func truncateForError(s string) string {
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}

// APIError is returned for non-2xx responses or "status: error" bodies.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("newsapi: %s (http %d, code=%s)", e.Message, e.Status, e.Code)
	}
	return fmt.Sprintf("newsapi: %s (http %d)", e.Message, e.Status)
}

type rawResponse struct {
	Status       string       `json:"status"`
	TotalResults int          `json:"totalResults"`
	Articles     []rawArticle `json:"articles"`
	// Error shape
	Code    string `json:"code"`
	Message string `json:"message"`
}

type rawArticle struct {
	Source      struct{ ID, Name string } `json:"source"`
	Author      string                    `json:"author"`
	Title       string                    `json:"title"`
	Description string                    `json:"description"`
	URL         string                    `json:"url"`
	URLToImage  string                    `json:"urlToImage"`
	PublishedAt string                    `json:"publishedAt"`
	Content     string                    `json:"content"`
}

type errorResponse struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
