package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/go-chi/chi/v5"
	"github.com/mentat/qodo/api/middleware"
	"google.golang.org/api/iterator"
)

type Todo struct {
	ID          string     `json:"id" firestore:"-"`
	Title       string     `json:"title" firestore:"title"`
	Description string     `json:"description" firestore:"description"`
	Completed   bool       `json:"completed" firestore:"completed"`
	Priority    string     `json:"priority" firestore:"priority"`
	Category    string     `json:"category" firestore:"category"`
	DueDate     *time.Time `json:"dueDate" firestore:"dueDate"`
	Position    int        `json:"position" firestore:"position"`
	UserID      string     `json:"userId" firestore:"userId"`
	CreatedAt   time.Time  `json:"createdAt" firestore:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt" firestore:"updatedAt"`
}

type TodoHandler struct {
	fs *firestore.Client
}

func NewTodoHandler(fs *firestore.Client) *TodoHandler {
	return &TodoHandler{fs: fs}
}

func (h *TodoHandler) collection() *firestore.CollectionRef {
	return h.fs.Collection("todos")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// List returns all todos for the authenticated user.
func (h *TodoHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())

	query := h.collection().Where("userId", "==", uid).OrderBy("position", firestore.Asc)

	if v := r.URL.Query().Get("completed"); v != "" {
		completed := v == "true"
		query = h.collection().Where("userId", "==", uid).Where("completed", "==", completed).OrderBy("position", firestore.Asc)
	}
	if v := r.URL.Query().Get("priority"); v != "" {
		query = h.collection().Where("userId", "==", uid).Where("priority", "==", v).OrderBy("position", firestore.Asc)
	}

	iter := query.Documents(r.Context())
	defer iter.Stop()

	todos := make([]Todo, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list todos")
			return
		}
		var t Todo
		doc.DataTo(&t)
		t.ID = doc.Ref.ID
		todos = append(todos, t)
	}

	// Client-side search filter
	if search := r.URL.Query().Get("search"); search != "" {
		search = strings.ToLower(search)
		filtered := make([]Todo, 0)
		for _, t := range todos {
			if strings.Contains(strings.ToLower(t.Title), search) ||
				strings.Contains(strings.ToLower(t.Description), search) {
				filtered = append(filtered, t)
			}
		}
		todos = filtered
	}

	writeJSON(w, http.StatusOK, todos)
}

// Create adds a new todo.
func (h *TodoHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())

	var t Todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(t.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	if t.Priority == "" {
		t.Priority = "medium"
	}
	if t.Priority != "low" && t.Priority != "medium" && t.Priority != "high" {
		writeError(w, http.StatusBadRequest, "priority must be low, medium, or high")
		return
	}

	// Get max position
	maxPos := -1
	iter := h.collection().Where("userId", "==", uid).OrderBy("position", firestore.Desc).Limit(1).Documents(r.Context())
	doc, err := iter.Next()
	if err == nil {
		var existing Todo
		doc.DataTo(&existing)
		maxPos = existing.Position
	}
	iter.Stop()

	now := time.Now().UTC()
	t.UserID = uid
	t.Completed = false
	t.Position = maxPos + 1
	t.CreatedAt = now
	t.UpdatedAt = now

	ref, _, err := h.collection().Add(r.Context(), t)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create todo")
		return
	}

	t.ID = ref.ID
	writeJSON(w, http.StatusCreated, t)
}

// Get returns a single todo.
func (h *TodoHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")

	doc, err := h.collection().Doc(id).Get(r.Context())
	if err != nil {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}

	var t Todo
	doc.DataTo(&t)
	t.ID = doc.Ref.ID

	if t.UserID != uid {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// Update performs a full replacement of a todo.
func (h *TodoHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")

	doc, err := h.collection().Doc(id).Get(r.Context())
	if err != nil {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}

	var existing Todo
	doc.DataTo(&existing)
	if existing.UserID != uid {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}

	var t Todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(t.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	t.ID = id
	t.UserID = uid
	t.CreatedAt = existing.CreatedAt
	t.UpdatedAt = time.Now().UTC()

	if _, err := h.collection().Doc(id).Set(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update todo")
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// Patch performs a partial update.
func (h *TodoHandler) Patch(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")

	doc, err := h.collection().Doc(id).Get(r.Context())
	if err != nil {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}

	var existing Todo
	doc.DataTo(&existing)
	if existing.UserID != uid {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}

	var patch map[string]any
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Prevent overwriting protected fields
	delete(patch, "id")
	delete(patch, "userId")
	delete(patch, "createdAt")
	patch["updatedAt"] = time.Now().UTC()

	updates := make([]firestore.Update, 0, len(patch))
	for k, v := range patch {
		updates = append(updates, firestore.Update{Path: k, Value: v})
	}

	if _, err := h.collection().Doc(id).Update(r.Context(), updates); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to patch todo")
		return
	}

	// Re-read to return full object
	doc, _ = h.collection().Doc(id).Get(r.Context())
	var t Todo
	doc.DataTo(&t)
	t.ID = doc.Ref.ID
	writeJSON(w, http.StatusOK, t)
}

// Delete removes a todo.
func (h *TodoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")

	doc, err := h.collection().Doc(id).Get(r.Context())
	if err != nil {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}

	var existing Todo
	doc.DataTo(&existing)
	if existing.UserID != uid {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}

	if _, err := h.collection().Doc(id).Delete(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete todo")
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

	batch := h.fs.Batch()
	now := time.Now().UTC()

	for _, item := range req.Items {
		ref := h.collection().Doc(item.ID)
		// Verify ownership
		doc, err := ref.Get(r.Context())
		if err != nil {
			continue
		}
		var t Todo
		doc.DataTo(&t)
		if t.UserID != uid {
			continue
		}
		batch.Update(ref, []firestore.Update{
			{Path: "position", Value: item.Position},
			{Path: "updatedAt", Value: now},
		})
	}

	if _, err := batch.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reorder todos")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
