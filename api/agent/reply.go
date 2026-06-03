package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/genai"

	"github.com/mentat/qodo/api/seed"
	"github.com/mentat/qodo/api/services"
)

// DefaultReplyModel composes in-character email replies. Full Flash (not
// Lite) — replies are short but benefit from stronger persona adherence.
const DefaultReplyModel = "gemini-2.5-flash"

// ReplyAgent generates in-character email replies and unprompted greetings.
// It is deliberately NOT Marvin's ADK runner: each character needs its own
// system instruction, replies are stateless one-shots, and characters must
// not touch the user's todos. It mirrors the Screener's direct genai client.
type ReplyAgent struct {
	client    *genai.Client
	modelName string
	timeout   time.Duration
}

// ReplyConfig configures the reply agent.
type ReplyConfig struct {
	ProjectID string
	Location  string
	ModelName string
	Timeout   time.Duration
}

// NewReplyAgent builds a reply agent wired to Vertex AI.
func NewReplyAgent(ctx context.Context, cfg ReplyConfig) (*ReplyAgent, error) {
	projectID := firstNonEmpty(cfg.ProjectID, os.Getenv("GOOGLE_CLOUD_PROJECT"), "qodo-demo")
	location := firstNonEmpty(cfg.Location, os.Getenv("GOOGLE_CLOUD_LOCATION"), "us-central1")
	modelName := firstNonEmpty(cfg.ModelName, DefaultReplyModel)
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend:  genai.BackendVertexAI,
		Project:  projectID,
		Location: location,
	})
	if err != nil {
		return nil, fmt.Errorf("reply agent: init genai client: %w", err)
	}
	return &ReplyAgent{client: client, modelName: modelName, timeout: timeout}, nil
}

// Reply generates a reply to the latest message in a thread, in the voice of
// the given character.
func (a *ReplyAgent) Reply(ctx context.Context, characterID string, thread []services.Email) (string, string, error) {
	ch, ok := seed.CharacterByID(characterID)
	if !ok {
		return "", "", fmt.Errorf("reply agent: unknown character %q", characterID)
	}
	return a.generate(ctx, ch, buildReplyPrompt(ch, thread))
}

// Greeting generates an unprompted in-character note (used by the drip cron).
func (a *ReplyAgent) Greeting(ctx context.Context, characterID string) (string, string, error) {
	ch, ok := seed.CharacterByID(characterID)
	if !ok {
		return "", "", fmt.Errorf("reply agent: unknown character %q", characterID)
	}
	return a.generate(ctx, ch, buildGreetingPrompt(ch))
}

func (a *ReplyAgent) generate(ctx context.Context, ch seed.Character, prompt string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	resp, err := a.client.Models.GenerateContent(ctx, a.modelName, []*genai.Content{
		{Role: "user", Parts: []*genai.Part{{Text: prompt}}},
	}, &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: ch.Persona + replyStyleSuffix}}},
		Temperature:       genai.Ptr[float32](0.9),
		ResponseMIMEType:  "application/json",
		ResponseSchema:    replySchema(),
	})
	if err != nil {
		return "", "", fmt.Errorf("reply generate: %w", err)
	}
	return parseReplyJSON(collectText(resp))
}

const replyStyleSuffix = `

You are writing an email. Reply with STRICT JSON: {"subject": <string>, "body": <string>}.
Keep the subject short; when replying, reuse the thread subject as "Re: ...".
Keep the body to a few sentences. Stay fully in character. Never break the fourth wall or mention being an AI.`

func buildReplyPrompt(ch seed.Character, thread []services.Email) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are %s. Below is an email thread, oldest first. Write your reply to the MOST RECENT message.\n\n", ch.Name)
	for _, m := range thread {
		who := m.FromName
		if who == "" {
			who = m.From
		}
		fmt.Fprintf(&b, "From: %s\nSubject: %s\n%s\n---\n", who, m.Subject, m.Body)
	}
	return b.String()
}

func buildGreetingPrompt(ch seed.Character) string {
	return fmt.Sprintf("You are %s. Write a short, unprompted email to the user — a spontaneous check-in, a passing thought, or an update — fully in character.", ch.Name)
}

func replySchema() *genai.Schema {
	return &genai.Schema{
		Type: "object",
		Properties: map[string]*genai.Schema{
			"subject": {Type: "string"},
			"body":    {Type: "string"},
		},
		Required: []string{"subject", "body"},
	}
}

// parseReplyJSON extracts {subject, body} from the model's JSON response.
func parseReplyJSON(s string) (string, string, error) {
	var out struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(s)), &out); err != nil {
		return "", "", fmt.Errorf("parse reply json: %w", err)
	}
	if strings.TrimSpace(out.Body) == "" {
		return "", "", fmt.Errorf("parse reply json: empty body")
	}
	return strings.TrimSpace(out.Subject), out.Body, nil
}
