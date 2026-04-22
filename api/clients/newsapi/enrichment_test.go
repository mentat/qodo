package newsapi_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mentat/qodo/api/clients/newsapi"
)

// Live integration: fetches a real article URL and converts it to markdown.
// Gated on network access; any specific URL may become unreachable.
func TestArticleFetcher_Live(t *testing.T) {
	f := newsapi.NewArticleFetcher()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Wikipedia is a stable, permissive target.
	md, err := f.Fetch(ctx, "https://en.wikipedia.org/wiki/Voyager_1")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(md) == 0 {
		t.Fatal("empty markdown")
	}
	if !strings.Contains(strings.ToLower(md), "voyager") {
		t.Errorf("expected markdown to mention voyager; got: %s", md[:min(500, len(md))])
	}
	if n := len([]rune(md)); n > 8000 {
		t.Errorf("markdown not truncated to 8000 runes: %d", n)
	}
}

// End-to-end: NewsAPI → scrape → markdown → Gemini summary.
// Required env: NEWSAPI_API_KEY, GOOGLE_APPLICATION_CREDENTIALS,
// GOOGLE_CLOUD_PROJECT (defaults to qodo-demo), GOOGLE_CLOUD_LOCATION.
func TestEnrichedSearch_Live(t *testing.T) {
	key := os.Getenv("NEWSAPI_API_KEY")
	if key == "" {
		t.Skip("NEWSAPI_API_KEY not set")
	}
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		t.Skip("GOOGLE_APPLICATION_CREDENTIALS not set")
	}
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		project = "qodo-demo"
	}
	location := os.Getenv("GOOGLE_CLOUD_LOCATION")
	if location == "" {
		location = "us-central1"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	summ, err := newsapi.NewVertexSummarizer(ctx, project, location)
	if err != nil {
		t.Fatalf("summarizer: %v", err)
	}
	c, err := newsapi.New(key,
		newsapi.WithFetcher(newsapi.NewArticleFetcher()),
		newsapi.WithSummarizer(summ),
	)
	if err != nil {
		t.Fatal(err)
	}
	arts, err := c.Search(ctx, newsapi.SearchParams{
		Query: "Kansas", Language: "en", PageSize: 3,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(arts) == 0 {
		t.Fatal("no articles")
	}
	// At least one of the 3 should have a non-empty Summary (scraping can
	// legitimately fail on some sites; requiring all N to succeed would be
	// flaky).
	summaries := 0
	markdowns := 0
	for _, a := range arts {
		if a.Summary != "" {
			summaries++
		}
		if a.Markdown != "" {
			markdowns++
		}
		t.Logf("- %s [%s] md=%d summary=%d", a.Title, a.Source, len([]rune(a.Markdown)), len([]rune(a.Summary)))
		if a.Summary != "" {
			t.Logf("  summary: %s", a.Summary)
		}
	}
	if summaries == 0 {
		t.Errorf("no article got a summary; expected at least 1 of %d", len(arts))
	}
	if markdowns == 0 {
		t.Errorf("no article got markdown; expected at least 1 of %d", len(arts))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
