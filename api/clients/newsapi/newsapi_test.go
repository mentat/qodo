package newsapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSearch_ParsesAndTruncates(t *testing.T) {
	bigContent := strings.Repeat("x", 9000) + " [+123 chars]"
	body := `{
	  "status":"ok","totalResults":2,
	  "articles":[
	    {"source":{"name":"Test Source"},"author":"A","title":"T1","description":"D1","url":"https://example.com/1","publishedAt":"2026-04-22T08:00:00Z","content":"` + bigContent + `"},
	    {"source":{"name":"Test 2"},"title":"T2","description":"D2","url":"https://example.com/2","publishedAt":"2026-04-22T09:00:00Z","content":"short"}
	  ]
	}`
	var gotHeader string
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Api-Key")
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	defer srv.Close()

	c, err := New("test-key", WithBaseURL(srv.URL), WithCharBudget(200))
	if err != nil {
		t.Fatal(err)
	}
	arts, err := c.Search(context.Background(), SearchParams{
		Query: "test query", Language: "en", SortBy: "popularity", PageSize: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotHeader != "test-key" {
		t.Errorf("want X-Api-Key=test-key, got %q", gotHeader)
	}
	for _, want := range []string{"q=test+query", "language=en", "sortBy=popularity", "pageSize=3"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
	if len(arts) != 2 {
		t.Fatalf("want 2 articles, got %d", len(arts))
	}
	// truncation enforced
	if n := len([]rune(arts[0].Text)); n > 200 {
		t.Errorf("article 1 text not truncated to 200 runes: got %d", n)
	}
	if !strings.HasSuffix(arts[0].Text, "…[truncated]") {
		t.Errorf("article 1 should have truncated suffix, got %q", arts[0].Text)
	}
	// [+nnn chars] trailer stripped
	if strings.Contains(arts[0].Text, "[+123 chars]") {
		t.Errorf("content trailer not stripped: %q", arts[0].Text)
	}
	if arts[0].Source != "Test Source" || arts[0].URL != "https://example.com/1" {
		t.Errorf("unexpected normalization: %+v", arts[0])
	}
	if arts[0].PublishedAt.IsZero() || arts[0].PublishedAt.Year() != 2026 {
		t.Errorf("publishedAt not parsed: %v", arts[0].PublishedAt)
	}
}

func TestSearch_DefaultPageSizeAndSortBy(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"status":"ok","articles":[]}`))
	}))
	defer srv.Close()
	c, _ := New("k", WithBaseURL(srv.URL))
	_, err := c.Search(context.Background(), SearchParams{Query: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "pageSize=5") || !strings.Contains(gotQuery, "sortBy=relevancy") {
		t.Errorf("defaults missing: %s", gotQuery)
	}
}

func TestSearch_PageSizeCappedAt20(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"status":"ok","articles":[]}`))
	}))
	defer srv.Close()
	c, _ := New("k", WithBaseURL(srv.URL))
	c.Search(context.Background(), SearchParams{Query: "x", PageSize: 500})
	if !strings.Contains(gotQuery, "pageSize=20") {
		t.Errorf("pageSize should be capped: %s", gotQuery)
	}
}

func TestSearch_APIErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"status":"error","code":"apiKeyInvalid","message":"Your API key is invalid."}`))
	}))
	defer srv.Close()
	c, _ := New("k", WithBaseURL(srv.URL))
	_, err := c.Search(context.Background(), SearchParams{Query: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("want *APIError, got %T: %v", err, err)
	}
	if apiErr.Code != "apiKeyInvalid" || apiErr.Status != 401 {
		t.Errorf("unexpected api error: %+v", apiErr)
	}
}

func TestSearch_StatusErrorOn200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"error","code":"parameterInvalid","message":"bad param"}`))
	}))
	defer srv.Close()
	c, _ := New("k", WithBaseURL(srv.URL))
	_, err := c.Search(context.Background(), SearchParams{Query: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if apiErr, ok := err.(*APIError); !ok || apiErr.Code != "parameterInvalid" {
		t.Errorf("want parameterInvalid, got %v", err)
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	c, _ := New("k")
	if _, err := c.Search(context.Background(), SearchParams{}); err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestNew_EmptyKey(t *testing.T) {
	if _, err := New(""); err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("abcdefghij", 5); got != "abcde" {
		// len(suffix)=13 > 5 so we just cut
		t.Errorf("got %q", got)
	}
	if got := Truncate("abcdefghij", 50); got != "abcdefghij" {
		t.Errorf("no-op case failed: %q", got)
	}
	long := strings.Repeat("a", 100)
	got := Truncate(long, 30)
	if !strings.HasSuffix(got, "…[truncated]") || len([]rune(got)) > 30 {
		t.Errorf("bad truncation: %q len=%d", got, len([]rune(got)))
	}
	// multibyte safety: 10 runes, each 3 bytes
	mb := strings.Repeat("日", 10)
	if got := Truncate(mb, 15); len([]rune(got)) != 10 {
		t.Errorf("multibyte: got %d runes", len([]rune(got)))
	}
}

func TestStripContentTrailer(t *testing.T) {
	if got := stripContentTrailer("hello world [+1234 chars]"); got != "hello world" {
		t.Errorf("got %q", got)
	}
	if got := stripContentTrailer("no trailer"); got != "no trailer" {
		t.Errorf("got %q", got)
	}
}

// ─── Integration (real NewsAPI) ──────────────────────────────────────────────

func TestSearch_Integration(t *testing.T) {
	key := os.Getenv("NEWSAPI_API_KEY")
	if key == "" {
		t.Skip("NEWSAPI_API_KEY not set")
	}
	c, err := New(key)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	arts, err := c.Search(ctx, SearchParams{Query: "technology", Language: "en", PageSize: 3})
	if err != nil {
		t.Fatalf("live search failed: %v", err)
	}
	if len(arts) == 0 {
		t.Fatal("expected at least one article")
	}
	for i, a := range arts {
		if a.Title == "" || a.URL == "" {
			t.Errorf("article %d missing title/url: %+v", i, a)
		}
		if n := len([]rune(a.Text)); n > 5000 {
			t.Errorf("article %d text %d runes exceeds 5000", i, n)
		}
	}
}
