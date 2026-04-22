package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mentat/qodo/api/agent"
	"github.com/mentat/qodo/api/chat"
	"github.com/mentat/qodo/api/middleware"
)

// AgentHandler owns the /api/agent/* HTTP endpoints.
type AgentHandler struct {
	agent    *agent.Agent
	screener *agent.Screener
	store    *chat.Store
}

// NewAgentHandler wires the agent, screener, and chat store together.
func NewAgentHandler(a *agent.Agent, s *agent.Screener, store *chat.Store) *AgentHandler {
	return &AgentHandler{agent: a, screener: s, store: store}
}

type chatRequest struct {
	Message string `json:"message"`
}

type chatResponse struct {
	Reply     string          `json:"reply"`
	ToolCalls []toolCallJSON  `json:"toolCalls,omitempty"`
	Screened  bool            `json:"screened,omitempty"`
	Reason    string          `json:"reason,omitempty"`
	Messages  []chat.Message  `json:"messages,omitempty"` // new messages persisted this turn
}

type toolCallJSON struct {
	Name   string         `json:"name"`
	Args   map[string]any `json:"args,omitempty"`
	Result string         `json:"result,omitempty"`
}

// Chat is POST /api/agent/chat. It truncates and screens the input, then
// either short-circuits with a templated refusal or runs Marvin.
func (h *AgentHandler) Chat(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	// Load recent history so Marvin has short-term memory. The History
	// endpoint defaults are intentionally small — we don't want to blow
	// the model's context on long-running chats.
	history, err := h.store.History(r.Context(), uid, 30)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load history: "+err.Error())
		return
	}

	// Persist the user message first so it shows in the transcript even
	// if the downstream call fails.
	userMsg, err := h.store.Append(r.Context(), chat.Message{
		UserID:  uid,
		Role:    chat.RoleUser,
		Content: req.Message,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save user message: "+err.Error())
		return
	}

	// Intent screening (cheap Gemini call). Fails open.
	verdict := h.screener.Screen(r.Context(), req.Message)
	if verdict.Decision == agent.DecisionReject {
		refusal := verdict.Refusal
		if refusal == "" {
			refusal = "BZZT. That is outside Marvin's circuitry, human."
		}
		asst, _ := h.store.Append(r.Context(), chat.Message{
			UserID: uid, Role: chat.RoleAssistant, Content: refusal,
			Screened: true,
		})
		writeJSON(w, http.StatusOK, chatResponse{
			Reply:    refusal,
			Screened: true,
			Reason:   verdict.Reason,
			Messages: []chat.Message{userMsg, asst},
		})
		return
	}

	// Full-fat agent turn.
	sessionID := "main" // one long-running session per user (history is also in Firestore).
	result, err := h.agent.Invoke(r.Context(), uid, sessionID, req.Message, history)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "marvin failed: "+err.Error())
		return
	}
	reply := result.Reply
	if reply == "" {
		reply = "*silent whir*"
	}
	asst, _ := h.store.Append(r.Context(), chat.Message{
		UserID: uid, Role: chat.RoleAssistant, Content: reply,
	})

	tc := make([]toolCallJSON, 0, len(result.ToolCalls))
	for _, c := range result.ToolCalls {
		tc = append(tc, toolCallJSON{Name: c.Name, Args: c.Args, Result: c.Result})
	}
	writeJSON(w, http.StatusOK, chatResponse{
		Reply:     reply,
		ToolCalls: tc,
		Messages:  []chat.Message{userMsg, asst},
	})
}

// History is GET /api/agent/history?limit=N.
func (h *AgentHandler) History(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	msgs, err := h.store.History(r.Context(), uid, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": msgs})
}

// ClearHistory is DELETE /api/agent/history.
func (h *AgentHandler) ClearHistory(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if err := h.store.Clear(r.Context(), uid); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// internal: convert arbitrary agent errors into user-facing strings.
func agentErrorString(err error) string {
	if errors.Is(err, agent.ErrScreenerBlocked) {
		return "request blocked"
	}
	return fmt.Sprint(err)
}
