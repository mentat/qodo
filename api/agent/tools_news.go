package agent

import (
	"context"
	"strings"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/mentat/qodo/api/clients/newsapi"
)

// SearchNewsInput is the Gemini-facing schema for the news tool.
type SearchNewsInput struct {
	Query    string `json:"query" jsonschema:"the search query, supports + - and quoted phrases"`
	Language string `json:"language,omitempty" jsonschema:"2-letter ISO-639-1 code e.g. en; defaults to en"`
	SortBy   string `json:"sort_by,omitempty" jsonschema:"relevancy | popularity | publishedAt; defaults to relevancy"`
	PageSize int    `json:"page_size,omitempty" jsonschema:"number of articles to return 1-10 (default 5)"`
}

// SearchNewsOutput is what Marvin sees back.
type SearchNewsOutput struct {
	Articles []NewsArticle `json:"articles"`
	Notice   string        `json:"notice,omitempty"`
}

// NewsArticle is a single flattened article.
type NewsArticle struct {
	Title       string `json:"title"`
	Source      string `json:"source"`
	URL         string `json:"url"`
	Author      string `json:"author,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
	// Summary is an LLM-generated 2–4 sentence digest of the article — the
	// primary field Marvin should quote from. Empty if summarization failed.
	Summary string `json:"summary,omitempty"`
	// Text is the NewsAPI snippet (description + truncated content). Always
	// present; useful as a fallback when Summary is empty.
	Text string `json:"text"`
}

// NewSearchNewsTool wires the NewsAPI client into an ADK function tool.
func NewSearchNewsTool(client *newsapi.Client) (tool.Tool, error) {
	handler := func(ctx tool.Context, in SearchNewsInput) (SearchNewsOutput, error) {
		lang := in.Language
		if lang == "" {
			lang = "en"
		}
		ps := in.PageSize
		if ps <= 0 {
			ps = 5
		}
		if ps > 10 {
			ps = 10
		}
		arts, err := client.Search(context.Background(), newsapi.SearchParams{
			Query:    strings.TrimSpace(in.Query),
			Language: lang,
			SortBy:   in.SortBy,
			PageSize: ps,
		})
		if err != nil {
			// Surface the error as a plain-text notice so Marvin can still respond.
			return SearchNewsOutput{Notice: "news search failed: " + err.Error()}, nil
		}
		out := SearchNewsOutput{Articles: make([]NewsArticle, 0, len(arts))}
		for _, a := range arts {
			out.Articles = append(out.Articles, NewsArticle{
				Title:       a.Title,
				Source:      a.Source,
				URL:         a.URL,
				Author:      a.Author,
				PublishedAt: formatTime(a.PublishedAt),
				Summary:     a.Summary,
				Text:        a.Text,
			})
		}
		if len(out.Articles) == 0 {
			out.Notice = "no articles matched; try a broader query"
		}
		return out, nil
	}
	return functiontool.New(functiontool.Config{
		Name: "search_news",
		Description: "Search recent news articles via NewsAPI's /v2/everything endpoint. " +
			"Returns up to 10 articles; each article's text is truncated to 5000 characters. " +
			"Use this for current events or trending stories.",
	}, handler)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
