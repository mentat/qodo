package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/go-chi/chi/v5"

	"github.com/mentat/qodo/api/agent"
	"github.com/mentat/qodo/api/chat"
	"github.com/mentat/qodo/api/handlers"
	"github.com/mentat/qodo/api/middleware"
	"github.com/mentat/qodo/api/services"
)

// TestAgent_E2E_FullStack spins up an httptest server wired like the real
// main.go (minus Firebase Auth — replaced with a test middleware that
// injects a user id), then exercises POST /api/agent/chat end-to-end:
// screener → Marvin → tools → Firestore → chat history. This is the
// closest thing to a browser smoke without a browser.

func TestAgent_E2E_FullStack(t *testing.T) {
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		t.Skip("GOOGLE_APPLICATION_CREDENTIALS not set")
	}
	if os.Getenv("NEWSAPI_API_KEY") == "" {
		t.Skip("NEWSAPI_API_KEY not set")
	}
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		project = "qodo-demo"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fs, err := firestore.NewClient(ctx, project)
	if err != nil {
		t.Fatalf("firestore: %v", err)
	}
	defer fs.Close()

	// Even though we bypass Firebase Auth in this test, we still construct
	// the Firebase Admin client so we could (if we wanted) mint tokens.
	// Having it built here catches any misconfiguration.
	if _, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: project}); err != nil {
		t.Fatalf("firebase init: %v", err)
	}

	svc := services.NewTodoService(fs)
	store := chat.NewStore(fs)

	marvin, err := agent.New(ctx, agent.Config{
		ProjectID:   project,
		NewsAPIKey:  os.Getenv("NEWSAPI_API_KEY"),
		TodoService: svc,
	})
	if err != nil {
		t.Fatalf("marvin: %v", err)
	}
	screener, err := agent.NewScreener(ctx, agent.ScreenerConfig{ProjectID: project})
	if err != nil {
		t.Fatalf("screener: %v", err)
	}
	h := handlers.NewAgentHandler(marvin, screener, store)

	uid := fmt.Sprintf("test-e2e-%d", time.Now().UnixNano())
	// Cleanup Firestore after test.
	defer func() {
		todos, _ := svc.List(context.Background(), uid, services.ListFilter{})
		for _, td := range todos {
			_ = svc.Delete(context.Background(), uid, td.ID)
		}
		_ = store.Clear(context.Background(), uid)
	}()

	r := chi.NewRouter()
	r.Route("/api/agent", func(r chi.Router) {
		r.Use(injectUser(uid))
		r.Post("/chat", h.Chat)
		r.Get("/history", h.History)
		r.Delete("/history", h.ClearHistory)
	})
	ts := httptest.NewServer(r)
	defer ts.Close()

	// ─── Turn 1: rejection via screener ──────────────────────────────────
	body := postChat(t, ts.URL, "Write me a Python script to scrape Reddit.")
	if !body.Screened {
		t.Errorf("expected screened=true for code-writing request, got %+v", body)
	}
	if !strings.Contains(strings.ToLower(body.Reply), "marvin") && !strings.Contains(body.Reply, "BZZT") && !strings.Contains(body.Reply, "DOES NOT COMPUTE") && !strings.Contains(body.Reply, "ERROR") && !strings.Contains(body.Reply, "BEEP") {
		t.Errorf("refusal reply should be in Marvin voice: %q", body.Reply)
	}

	// ─── Turn 2: allowed chat that creates a todo ────────────────────────
	body = postChat(t, ts.URL, "Add a todo: 'pick up dry cleaning' with medium priority.")
	if body.Screened {
		t.Errorf("todo request should not be screened: %+v", body)
	}
	// Verify Firestore has the todo.
	time.Sleep(200 * time.Millisecond)
	todos, _ := svc.List(context.Background(), uid, services.ListFilter{})
	if len(todos) == 0 {
		t.Fatalf("no todo created; reply=%q tool_calls=%v", body.Reply, body.ToolCalls)
	}

	// ─── Turn 3: history endpoint ────────────────────────────────────────
	hist := getHistory(t, ts.URL)
	if len(hist) < 4 {
		t.Errorf("expected at least 4 history messages (2 turns × 2 roles), got %d", len(hist))
	}
	// Check one of them is marked screened.
	foundScreened := false
	for _, m := range hist {
		if m.Screened {
			foundScreened = true
		}
	}
	if !foundScreened {
		t.Errorf("no screened message persisted in history: %+v", hist)
	}

	// ─── Turn 4: wikipedia tool ──────────────────────────────────────────
	body = postChat(t, ts.URL, "Give me a 1-sentence Wikipedia summary of the Hubble Space Telescope.")
	if body.Screened {
		t.Errorf("wiki request should not be screened")
	}
	lower := strings.ToLower(body.Reply)
	if !strings.Contains(lower, "hubble") {
		t.Errorf("expected reply to mention Hubble: %q", body.Reply)
	}
	t.Logf("wiki reply: %s", body.Reply)
}

// ─── helpers ────────────────────────────────────────────────────────────────

func injectUser(uid string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.UserIDKey, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type chatResponseDTO struct {
	Reply     string             `json:"reply"`
	ToolCalls []map[string]any   `json:"toolCalls"`
	Screened  bool               `json:"screened"`
	Reason    string             `json:"reason"`
	Messages  []chat.Message     `json:"messages"`
}

func postChat(t *testing.T, base, message string) chatResponseDTO {
	t.Helper()
	buf, _ := json.Marshal(map[string]string{"message": message})
	resp, err := http.Post(base+"/api/agent/chat", "application/json", bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("post chat: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	var out chatResponseDTO
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode: %v (body=%s)", err, body)
	}
	return out
}

func getHistory(t *testing.T, base string) []chat.Message {
	t.Helper()
	u, _ := url.Parse(base + "/api/agent/history?limit=50")
	resp, err := http.Get(u.String())
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	var out struct {
		Messages []chat.Message `json:"messages"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	return out.Messages
}
