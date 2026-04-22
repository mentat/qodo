package newsapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/genai"
)

// DefaultSummarizerModel is the cheapest Gemini 2.5 tier.
const DefaultSummarizerModel = "gemini-2.5-flash-lite"

// ArticleSummarizer turns an article blob into 2-4 crisp sentences. It's
// designed around the cheapest production Gemini tier (2.5 Flash-Lite) so it
// can be invoked per-article without blowing the budget.
type ArticleSummarizer struct {
	client    *genai.Client
	modelName string
	timeout   time.Duration
}

// SummarizerOption configures an ArticleSummarizer.
type SummarizerOption func(*ArticleSummarizer)

// WithSummarizerModel overrides the Gemini model used. Default is gemini-2.5-flash-lite.
func WithSummarizerModel(name string) SummarizerOption {
	return func(s *ArticleSummarizer) { s.modelName = name }
}

// WithSummarizerTimeout caps each summarization request. Default 8s.
func WithSummarizerTimeout(d time.Duration) SummarizerOption {
	return func(s *ArticleSummarizer) { s.timeout = d }
}

// NewVertexSummarizer builds a summarizer backed by Vertex AI using ADC.
// project + location must be valid GCP coords (e.g. "qodo-demo", "us-central1").
func NewVertexSummarizer(ctx context.Context, project, location string, opts ...SummarizerOption) (*ArticleSummarizer, error) {
	if project == "" || location == "" {
		return nil, fmt.Errorf("summarizer: project and location are required")
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend:  genai.BackendVertexAI,
		Project:  project,
		Location: location,
	})
	if err != nil {
		return nil, fmt.Errorf("summarizer: genai client: %w", err)
	}
	s := &ArticleSummarizer{client: client, modelName: DefaultSummarizerModel, timeout: 8 * time.Second}
	for _, o := range opts {
		o(s)
	}
	return s, nil
}

// Summarize produces a short summary. Returns "" if the body is effectively
// empty so callers can skip the LLM call entirely. On any error the empty
// string is returned along with the error so the caller can decide whether
// to fall back to the original text.
func (s *ArticleSummarizer) Summarize(ctx context.Context, title, body string) (string, error) {
	body = strings.TrimSpace(body)
	if len(body) < 80 {
		return "", nil // not worth an LLM call
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	prompt := fmt.Sprintf(
		"Summarize the following news article in 2–4 short sentences, as plain prose (no bullet points, no title, no markdown). "+
			"Be factual and neutral. Do not invent facts.\n\nTitle: %s\n\nBody:\n%s",
		strings.TrimSpace(title), body,
	)

	resp, err := s.client.Models.GenerateContent(ctx, s.modelName, []*genai.Content{
		{Role: "user", Parts: []*genai.Part{{Text: prompt}}},
	}, &genai.GenerateContentConfig{
		Temperature:     genai.Ptr[float32](0.2),
		MaxOutputTokens: 256,
	})
	if err != nil {
		return "", fmt.Errorf("summarizer: generate: %w", err)
	}
	return strings.TrimSpace(collectText(resp)), nil
}

func collectText(resp *genai.GenerateContentResponse) string {
	if resp == nil {
		return ""
	}
	var b strings.Builder
	for _, c := range resp.Candidates {
		if c == nil || c.Content == nil {
			continue
		}
		for _, p := range c.Content.Parts {
			if p != nil && p.Text != "" {
				b.WriteString(p.Text)
			}
		}
	}
	return b.String()
}
