package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"cloud.google.com/go/firestore"
	"github.com/go-chi/chi/v5"
	"github.com/mentat/qodo/api/middleware"
	"github.com/mentat/qodo/api/services"
)

// Todo is re-exported for backward compatibility with callers that still
// reference handlers.Todo.
type Todo = services.Todo

// TodoHandler adapts services.TodoService to HTTP.
type TodoHandler struct {
	svc *services.TodoService
}

// NewTodoHandler constructs a handler using a Firestore-backed service.
func NewTodoHandler(fs *firestore.Client) *TodoHandler {
	return &TodoHandler{svc: services.NewTodoService(fs)}
}

// NewTodoHandlerWithService lets tests inject a service instance (e.g.
// pointing at a per-test collection).
func NewTodoHandlerWithService(svc *services.TodoService) *TodoHandler {
	return &TodoHandler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func statusFor(err error) (int, string) {
	switch {
	case errors.Is(err, services.ErrNotFound):
		return http.StatusNotFound, "todo not found"
	case errors.Is(err, services.ErrInvalidInput):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, services.ErrUnauthenticated):
		return http.StatusUnauthorized, "unauthorized"
	default:
		return http.StatusInternalServerError, "internal error"
	}
}

// List returns the current user's todos with optional query filters.
func (h *TodoHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())

	var filter services.ListFilter
	if v := r.URL.Query().Get("completed"); v != "" {
		b := v == "true"
		filter.Completed = &b
	}
	filter.Priority = r.URL.Query().Get("priority")
	filter.Search = r.URL.Query().Get("search")

	todos, err := h.svc.List(r.Context(), uid, filter)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, todos)
}

// Create adds a new todo.
func (h *TodoHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var in services.Todo
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t, err := h.svc.Create(r.Context(), uid, services.CreateInput{
		Title:       in.Title,
		Description: in.Description,
		Priority:    in.Priority,
		Category:    in.Category,
		DueDate:     in.DueDate,
	})
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// Get returns a single todo.
func (h *TodoHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	t, err := h.svc.Get(r.Context(), uid, id)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// Update performs a full replacement of a todo.
func (h *TodoHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	var in services.Todo
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t, err := h.svc.Replace(r.Context(), uid, id, in)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// Patch performs a partial update.
func (h *TodoHandler) Patch(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	var patch map[string]any
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t, err := h.svc.Patch(r.Context(), uid, id, patch)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// Delete removes a todo.
func (h *TodoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Search runs a stemmed full-text search over the user's todos. Query
// params: q (required), limit (optional, default 50), completed
// ("true"|"false", optional), priority (low|medium|high, optional).
func (h *TodoHandler) Search(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	q := r.URL.Query().Get("q")
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	var filter services.ListFilter
	if v := r.URL.Query().Get("completed"); v != "" {
		b := v == "true"
		filter.Completed = &b
	}
	filter.Priority = r.URL.Query().Get("priority")

	todos, err := h.svc.Search(r.Context(), uid, q, limit, filter)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, todos)
}

// Reorder batch-updates positions for todos.
func (h *TodoHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var req struct {
		Items []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	items := make([]services.ReorderItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, services.ReorderItem{ID: it.ID, Position: it.Position})
	}
	if err := h.svc.Reorder(r.Context(), uid, items); err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
