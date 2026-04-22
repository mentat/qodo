package agent

import (
	"context"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/mentat/qodo/api/clients/wikipedia"
)

// SearchWikipediaInput is the Gemini-facing schema for the Wikipedia tool.
type SearchWikipediaInput struct {
	Query string `json:"query" jsonschema:"free-text search term; the tool resolves it to the best matching page"`
}

// SearchWikipediaOutput flattens a Wikipedia summary.
type SearchWikipediaOutput struct {
	Found       bool     `json:"found"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Extract     string   `json:"extract,omitempty"`
	URL         string   `json:"url,omitempty"`
	// Candidates is populated when the matched page is a disambiguation
	// page; Marvin should pick one and call the tool again with the
	// exact title.
	Candidates []string `json:"candidates,omitempty"`
	Notice     string   `json:"notice,omitempty"`
}

// NewSearchWikipediaTool wires the Wikipedia client into an ADK function tool.
func NewSearchWikipediaTool(client *wikipedia.Client) (tool.Tool, error) {
	handler := func(ctx tool.Context, in SearchWikipediaInput) (SearchWikipediaOutput, error) {
		r, err := client.Search(context.Background(), in.Query)
		if err != nil {
			if err == wikipedia.ErrNotFound {
				return SearchWikipediaOutput{Found: false, Notice: "no Wikipedia page matched the query"}, nil
			}
			return SearchWikipediaOutput{Found: false, Notice: "wikipedia search failed: " + err.Error()}, nil
		}
		return SearchWikipediaOutput{
			Found:       true,
			Title:       r.Title,
			Description: r.Description,
			Extract:     r.Extract,
			URL:         r.URL,
			Candidates:  r.Candidates,
		}, nil
	}
	return functiontool.New(functiontool.Config{
		Name: "search_wikipedia",
		Description: "Look up a Wikipedia article by free-text query. Returns the page title, " +
			"a short description, and an extract truncated to 5000 characters. When the query matches " +
			"a disambiguation page, the 'candidates' field lists alternative titles; call again with one of those.",
	}, handler)
}
