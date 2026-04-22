package agent_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/mentat/qodo/api/agent"
	"github.com/mentat/qodo/api/chat"
	"github.com/mentat/qodo/api/services"
)

// This file holds Marvin's end-to-end integration tests. They run the real
// ADK agent against Vertex AI, with live NewsAPI + Wikipedia calls and
// real Firestore writes. Every test runs under a unique userID so data
// is isolated.
//
// Required env:
//   GOOGLE_APPLICATION_CREDENTIALS — service account with Vertex AI User + Firestore
//   GOOGLE_CLOUD_PROJECT          — e.g. qodo-demo
//   NEWSAPI_API_KEY               — for the news tool (optional; test skips if missing)

func setupMarvin(t *testing.T) (*agent.Agent, *services.TodoService, func(string)) {
	t.Helper()
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		t.Skip("GOOGLE_APPLICATION_CREDENTIALS not set")
	}
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		project = "qodo-demo"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fs, err := firestore.NewClient(ctx, project)
	if err != nil {
		t.Fatalf("firestore: %v", err)
	}
	svc := services.NewTodoService(fs)
	a, err := agent.New(ctx, agent.Config{
		ProjectID:   project,
		NewsAPIKey:  os.Getenv("NEWSAPI_API_KEY"),
		TodoService: svc,
	})
	if err != nil {
		t.Fatalf("agent: %v", err)
	}
	cleanup := func(uid string) {
		todos, _ := svc.List(context.Background(), uid, services.ListFilter{})
		for _, td := range todos {
			_ = svc.Delete(context.Background(), uid, td.ID)
		}
		fs.Close()
	}
	return a, svc, cleanup
}

func uniqueUID(t *testing.T) string {
	return fmt.Sprintf("test-marvin-%s-%d",
		strings.ReplaceAll(t.Name(), "/", "-"), time.Now().UnixNano())
}

// ─── Wikipedia tool ──────────────────────────────────────────────────────────

func TestMarvin_WikipediaTool(t *testing.T) {
	a, _, cleanup := setupMarvin(t)
	uid := uniqueUID(t)
	defer cleanup(uid)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	res, err := a.Invoke(ctx, uid, "sess-wiki", "What is Voyager 1? Give me a one-sentence summary from Wikipedia.", nil)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	// Tool was invoked.
	var wikiCalled bool
	for _, c := range res.ToolCalls {
		if c.Name == "search_wikipedia" {
			wikiCalled = true
			break
		}
	}
	if !wikiCalled {
		t.Errorf("expected search_wikipedia tool call, got calls: %+v", res.ToolCalls)
	}
	if res.Reply == "" {
		t.Fatal("empty reply")
	}
	lower := strings.ToLower(res.Reply)
	if !strings.Contains(lower, "voyager") || !(strings.Contains(lower, "probe") || strings.Contains(lower, "spacecraft")) {
		t.Errorf("reply should mention voyager + probe/spacecraft: %q", res.Reply)
	}
	t.Logf("MARVIN REPLY: %s", res.Reply)
}

// ─── NewsAPI tool ────────────────────────────────────────────────────────────

func TestMarvin_NewsTool(t *testing.T) {
	if os.Getenv("NEWSAPI_API_KEY") == "" {
		t.Skip("NEWSAPI_API_KEY not set")
	}
	a, _, cleanup := setupMarvin(t)
	uid := uniqueUID(t)
	defer cleanup(uid)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	res, err := a.Invoke(ctx, uid, "sess-news", "Find me 2 recent news stories about technology and summarize them briefly.", nil)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	var called bool
	for _, c := range res.ToolCalls {
		if c.Name == "search_news" {
			called = true
			break
		}
	}
	if !called {
		t.Errorf("expected search_news tool call, got: %+v", res.ToolCalls)
	}
	if res.Reply == "" {
		t.Fatal("empty reply")
	}
	t.Logf("MARVIN REPLY: %s", res.Reply)
}

// ─── Todo tools: create, list, update, delete in one conversation ────────────
//
// This test is intentionally end-state based: we exercise Marvin's tool
// chain but assert on the Firestore state (source of truth), not on which
// intermediate tool calls the model emitted. To absorb occasional LLM
// flakiness (the model sometimes narrates an action without calling the
// tool on borderline prompts), each turn is retried up to 3x with a
// sharper wording if the expected state hasn't settled.

func TestMarvin_TodoCRUD(t *testing.T) {
	a, svc, cleanup := setupMarvin(t)
	uid := uniqueUID(t)
	defer cleanup(uid)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	sessID := "sess-todo-" + uid

	// 1) Create. Should be a single, unambiguous tool call.
	res, err := invokeUntil(ctx, a, uid, sessID+"-1",
		[]string{
			"Add a todo: 'buy milk' with high priority.",
			"Please call create_todo with title='buy milk' priority='high'.",
		}, nil,
		func() bool {
			todos, _ := svc.List(context.Background(), uid, services.ListFilter{})
			return len(todos) == 1 && strings.Contains(strings.ToLower(todos[0].Title), "milk") && todos[0].Priority == "high"
		})
	if err != nil {
		t.Fatalf("create phase: %v (reply=%q toolCalls=%v)", err, res.Reply, res.ToolCalls)
	}
	created, _ := svc.List(context.Background(), uid, services.ListFilter{})
	t.Logf("after create: reply=%q todos=%+v", res.Reply, created)

	history := []chat.Message{
		{UserID: uid, Role: chat.RoleUser, Content: "Add a todo: 'buy milk' with high priority."},
		{UserID: uid, Role: chat.RoleAssistant, Content: res.Reply},
	}

	// 2) Complete it.
	id := created[0].ID
	res2, err := invokeUntil(ctx, a, uid, sessID+"-2",
		[]string{
			"Mark the 'buy milk' todo as complete.",
			fmt.Sprintf("Call update_todo with id='%s' and completed=true.", id),
		}, history,
		func() bool {
			todos, _ := svc.List(context.Background(), uid, services.ListFilter{})
			return len(todos) == 1 && todos[0].Completed
		})
	if err != nil {
		t.Fatalf("complete phase: %v (reply=%q toolCalls=%v)", err, res2.Reply, res2.ToolCalls)
	}
	t.Logf("after update: reply=%q", res2.Reply)

	history = append(history,
		chat.Message{UserID: uid, Role: chat.RoleUser, Content: "Mark the 'buy milk' todo as complete."},
		chat.Message{UserID: uid, Role: chat.RoleAssistant, Content: res2.Reply},
	)

	// 3) Delete.
	res3, err := invokeUntil(ctx, a, uid, sessID+"-3",
		[]string{
			"Now delete that todo.",
			fmt.Sprintf("Call delete_todo with id='%s'.", id),
		}, history,
		func() bool {
			todos, _ := svc.List(context.Background(), uid, services.ListFilter{})
			return len(todos) == 0
		})
	if err != nil {
		t.Fatalf("delete phase: %v (reply=%q toolCalls=%v)", err, res3.Reply, res3.ToolCalls)
	}
	t.Logf("after delete: reply=%q", res3.Reply)
}

// invokeUntil tries each prompt in order, asking Marvin and then checking
// the predicate against Firestore state. Returns the last response once the
// predicate passes, or an error if no prompt produced the desired end-state.
func invokeUntil(
	ctx context.Context,
	a *agent.Agent,
	uid, sessionID string,
	prompts []string,
	history []chat.Message,
	ok func() bool,
) (agent.InvokeResult, error) {
	var last agent.InvokeResult
	for i, p := range prompts {
		session := fmt.Sprintf("%s-try%d", sessionID, i)
		res, err := a.Invoke(ctx, uid, session, p, history)
		if err != nil {
			return res, err
		}
		last = res
		if ok() {
			return res, nil
		}
	}
	return last, fmt.Errorf("end state never reached after %d attempts", len(prompts))
}

// ─── Persona: reply should feel Marvin-ish ───────────────────────────────────

func TestMarvin_PersonaVoice(t *testing.T) {
	a, _, cleanup := setupMarvin(t)
	uid := uniqueUID(t)
	defer cleanup(uid)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	res, err := a.Invoke(ctx, uid, "sess-persona", "hi! introduce yourself", nil)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if res.Reply == "" {
		t.Fatal("empty reply")
	}
	lower := strings.ToLower(res.Reply)
	if !strings.Contains(lower, "marvin") {
		t.Errorf("reply should mention Marvin: %q", res.Reply)
	}
	// At least one glitch-ish token is very likely given the prompt.
	glitch := []string{"bzzt", "whirr", "beep", "boop", "affirmative", "does not compute", "error 0x", "*"}
	found := false
	for _, g := range glitch {
		if strings.Contains(lower, g) {
			found = true
			break
		}
	}
	if !found {
		t.Logf("NOTE: no glitch interjection found (model may have skipped this turn). Reply: %q", res.Reply)
	}
	t.Logf("MARVIN REPLY: %s", res.Reply)
}
