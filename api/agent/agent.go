// Package agent builds Marvin — the chat agent wired into Qodo's HTTP API.
//
// Marvin uses Google's ADK for Go (google.golang.org/adk) with Gemini on
// Vertex AI as the LLM. Tools give him access to NewsAPI, Wikipedia, and
// CRUD over the user's todos. The [Agent.Invoke] method is the single
// entry point used by HTTP handlers: it takes a userID + message and
// returns the final assistant text after any tool-calling iterations.
//
// TODO(marvin/context-compaction): long-running chats and beefy tool
// responses will blow past Gemini's input budget. Build a context
// compaction layer that (a) summarizes old turns via a cheap model
// before they're replayed, and (b) extracts durable "reveries" —
// high-signal facts the user tells Marvin (preferences, names, project
// codes) — into a persistent reveries store keyed by userID. Those
// short tidbits should be spliced into Marvin's system prompt every
// turn (uncompressed) while the rolling chat log itself is aggressively
// summarized. See agent.Config.History and the chat package.
package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"

	"github.com/mentat/qodo/api/chat"
	"github.com/mentat/qodo/api/clients/newsapi"
	"github.com/mentat/qodo/api/clients/wikipedia"
	"github.com/mentat/qodo/api/services"
)

const (
	// DefaultModelName is Marvin's main LLM. Gemini 2.5 Flash balances speed,
	// tool-calling reliability, and cost for a chat UX.
	DefaultModelName = "gemini-2.5-flash"
	// AppName identifies this agent inside the ADK session service.
	AppName = "qodo-marvin"
)

// Config configures Marvin.
type Config struct {
	// ProjectID is the GCP project for Vertex AI. Defaults to
	// $GOOGLE_CLOUD_PROJECT, then "qodo-demo".
	ProjectID string
	// Location is the Vertex AI region. Defaults to
	// $GOOGLE_CLOUD_LOCATION, then "us-central1".
	Location string
	// ModelName overrides the Gemini model. Defaults to DefaultModelName.
	ModelName string
	// NewsAPIKey is the NewsAPI.org API key. If empty, the search_news
	// tool is omitted and Marvin will be told he can't fetch news.
	NewsAPIKey string

	// TodoService provides todo CRUD (required).
	TodoService *services.TodoService
	// NewsAPI is an optional pre-built client; if nil, one is built from NewsAPIKey.
	NewsAPI *newsapi.Client
	// Wikipedia is an optional pre-built client; if nil, a default is built.
	Wikipedia *wikipedia.Client
}

// Agent is a constructed Marvin instance plus its ADK runner.
type Agent struct {
	llm       agent.Agent
	run       *runner.Runner
	sessions  session.Service
	modelName string
}

// New builds Marvin. It fails if the LLM or required services can't be initialized.
func New(ctx context.Context, cfg Config) (*Agent, error) {
	if cfg.TodoService == nil {
		return nil, errors.New("agent: TodoService is required")
	}
	projectID := firstNonEmpty(cfg.ProjectID, os.Getenv("GOOGLE_CLOUD_PROJECT"), "qodo-demo")
	location := firstNonEmpty(cfg.Location, os.Getenv("GOOGLE_CLOUD_LOCATION"), "us-central1")
	modelName := firstNonEmpty(cfg.ModelName, DefaultModelName)

	model, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{
		Backend:  genai.BackendVertexAI,
		Project:  projectID,
		Location: location,
	})
	if err != nil {
		return nil, fmt.Errorf("agent: init gemini on vertex ai: %w", err)
	}

	// Tools.
	tools := []tool.Tool{}

	// News (optional — only if an API key is available).
	newsClient := cfg.NewsAPI
	if newsClient == nil && cfg.NewsAPIKey != "" {
		c, err := newsapi.New(cfg.NewsAPIKey)
		if err != nil {
			return nil, fmt.Errorf("agent: newsapi client: %w", err)
		}
		newsClient = c
	}
	if newsClient != nil {
		t, err := NewSearchNewsTool(newsClient)
		if err != nil {
			return nil, err
		}
		tools = append(tools, t)
	}

	// Wikipedia.
	wikiClient := cfg.Wikipedia
	if wikiClient == nil {
		wikiClient = wikipedia.New()
	}
	wtool, err := NewSearchWikipediaTool(wikiClient)
	if err != nil {
		return nil, err
	}
	tools = append(tools, wtool)

	// Todo tools.
	listT, err := NewListTodosTool(cfg.TodoService)
	if err != nil {
		return nil, err
	}
	createT, err := NewCreateTodoTool(cfg.TodoService)
	if err != nil {
		return nil, err
	}
	updateT, err := NewUpdateTodoTool(cfg.TodoService)
	if err != nil {
		return nil, err
	}
	deleteT, err := NewDeleteTodoTool(cfg.TodoService)
	if err != nil {
		return nil, err
	}
	tools = append(tools, listT, createT, updateT, deleteT)

	instruction := MarvinInstruction
	if newsClient == nil {
		instruction += "\n\nNOTE: news search is currently UNAVAILABLE. Tell the user so if they ask."
	}

	llm, err := llmagent.New(llmagent.Config{
		Name:        "marvin",
		Model:       model,
		Description: "A slightly malfunctioning 90s productivity robot that manages the user's todos and can fetch news + Wikipedia.",
		Instruction: instruction,
		Tools:       tools,
	})
	if err != nil {
		return nil, fmt.Errorf("agent: llmagent: %w", err)
	}

	sessSvc := session.InMemoryService()
	run, err := runner.New(runner.Config{
		AppName:           AppName,
		Agent:             llm,
		SessionService:    sessSvc,
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, fmt.Errorf("agent: runner: %w", err)
	}

	return &Agent{llm: llm, run: run, sessions: sessSvc, modelName: modelName}, nil
}

// InvokeResult is the summary of one /chat turn.
type InvokeResult struct {
	Reply     string         // Marvin's final text.
	ToolCalls []ToolCallInfo // Every tool call + brief result snippet, in order.
}

// ToolCallInfo is a succinct record of a tool invocation.
type ToolCallInfo struct {
	Name   string
	Args   map[string]any
	Result string // JSON-ish string, truncated for logging.
}

// Invoke runs Marvin for one user message. It uses a per-invocation ADK
// session keyed on sessionID so callers can bring their own history-loading
// strategy. The returned Reply is the concatenation of any text emitted
// after the last tool call finished.
func (a *Agent) Invoke(ctx context.Context, userID, sessionID, message string, history []chat.Message) (InvokeResult, error) {
	if strings.TrimSpace(userID) == "" {
		return InvokeResult{}, errors.New("agent: userID required")
	}
	if strings.TrimSpace(message) == "" {
		return InvokeResult{}, errors.New("agent: message required")
	}
	ctx = WithUserID(ctx, userID)

	// Pre-populate the ADK session with prior turns so Marvin has context.
	// ADK's session service keys sessions by (appName, userID, sessionID);
	// we use the Firestore message list as source of truth and replay it.
	if err := a.replayHistory(ctx, userID, sessionID, history); err != nil {
		return InvokeResult{}, fmt.Errorf("agent: replay history: %w", err)
	}

	content := &genai.Content{
		Role:  "user",
		Parts: []*genai.Part{{Text: message}},
	}
	var reply strings.Builder
	var calls []ToolCallInfo

	for ev, err := range a.run.Run(ctx, userID, sessionID, content, agent.RunConfig{}) {
		if err != nil {
			return InvokeResult{}, fmt.Errorf("agent: run: %w", err)
		}
		if ev == nil || ev.LLMResponse.Content == nil {
			continue
		}
		// Collect tool calls for logging.
		for _, p := range ev.LLMResponse.Content.Parts {
			if p == nil {
				continue
			}
			if p.FunctionCall != nil {
				calls = append(calls, ToolCallInfo{
					Name: p.FunctionCall.Name,
					Args: p.FunctionCall.Args,
				})
			}
			if p.FunctionResponse != nil && len(calls) > 0 {
				// Attach response to the most recent matching call.
				for i := len(calls) - 1; i >= 0; i-- {
					if calls[i].Name == p.FunctionResponse.Name && calls[i].Result == "" {
						calls[i].Result = summarizeResponse(p.FunctionResponse.Response)
						break
					}
				}
			}
		}
		// Only capture text from assistant turns, and only when it's a
		// non-partial final-ish piece. Partial pieces get appended across
		// the stream, so we just always accumulate author=="marvin" text.
		if ev.Author != "user" && !ev.LLMResponse.Partial {
			for _, p := range ev.LLMResponse.Content.Parts {
				if p != nil && p.Text != "" {
					if reply.Len() > 0 {
						reply.WriteString("\n")
					}
					reply.WriteString(p.Text)
				}
			}
		}
	}

	return InvokeResult{Reply: strings.TrimSpace(reply.String()), ToolCalls: calls}, nil
}

// replayHistory seeds the in-memory ADK session with prior chat messages.
// If the session already exists with events, it's left alone.
func (a *Agent) replayHistory(ctx context.Context, userID, sessionID string, history []chat.Message) error {
	if len(history) == 0 {
		return nil
	}
	// Try to fetch — if it exists, assume it's already seeded.
	if resp, err := a.sessions.Get(ctx, &session.GetRequest{
		AppName: AppName, UserID: userID, SessionID: sessionID,
	}); err == nil && resp != nil && resp.Session != nil {
		if resp.Session.Events().Len() > 0 {
			return nil
		}
	}
	// Create (or get) then append a synthetic event per history message.
	created, err := a.sessions.Create(ctx, &session.CreateRequest{
		AppName: AppName, UserID: userID, SessionID: sessionID,
	})
	if err != nil || created == nil || created.Session == nil {
		// Might already exist — tolerate.
		get, err2 := a.sessions.Get(ctx, &session.GetRequest{
			AppName: AppName, UserID: userID, SessionID: sessionID,
		})
		if err2 != nil || get == nil {
			if err != nil {
				return err
			}
			return err2
		}
		created = &session.CreateResponse{Session: get.Session}
	}
	for _, m := range history {
		role := "user"
		if m.Role == chat.RoleAssistant {
			role = "model"
		}
		ev := session.NewEvent("seed")
		ev.Author = role
		if role == "model" {
			ev.Author = "marvin"
		}
		ev.LLMResponse.Content = &genai.Content{
			Role:  role,
			Parts: []*genai.Part{{Text: m.Content}},
		}
		if err := a.sessions.AppendEvent(ctx, created.Session, ev); err != nil {
			return err
		}
	}
	return nil
}

// ModelName returns the Gemini model name in use.
func (a *Agent) ModelName() string { return a.modelName }

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func summarizeResponse(resp map[string]any) string {
	const max = 200
	s := fmt.Sprintf("%v", resp)
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
