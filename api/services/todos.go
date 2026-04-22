// Package services contains reusable business logic shared by HTTP handlers
// and the AI agent's tools. Every method enforces the invariant that a
// userID is required and that a user can only ever read or mutate their own
// todos.
package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/mentat/qodo/api/search"
)

// Todo is the canonical Firestore-mapped todo.
// Fields mirror handlers.Todo to keep the JSON contract stable.
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
	// FullText is an inverted-index-friendly repeated field of stemmed
	// tokens derived from Title + Description + Category. It's rebuilt on
	// every write and queried via array-contains-any for full-text search.
	// Capped at search.MaxTokens (5) entries.
	FullText []string `json:"fullText,omitempty" firestore:"fullText,omitempty"`
}

// buildFullText computes the FullText index entries from a todo's text fields.
func buildFullText(t Todo) []string {
	return search.Build(t.Title, t.Description, t.Category)
}

// ListFilter narrows a List query. Zero value means no filter.
type ListFilter struct {
	Completed *bool
	Priority  string
	Search    string
}

// Errors returned by TodoService.
var (
	ErrNotFound       = errors.New("todo not found")
	ErrInvalidInput   = errors.New("invalid input")
	ErrUnauthenticated = errors.New("userID required")
)

// TodoService exposes todo CRUD. Safe for concurrent use.
type TodoService struct {
	fs         *firestore.Client
	collection string
}

// NewTodoService constructs a service backed by the given Firestore client.
// The collection defaults to "todos".
func NewTodoService(fs *firestore.Client) *TodoService {
	return &TodoService{fs: fs, collection: "todos"}
}

// WithCollection returns a copy of the service that reads/writes the given
// collection. Used in tests to isolate against a per-test namespace.
func (s *TodoService) WithCollection(name string) *TodoService {
	cp := *s
	cp.collection = name
	return &cp
}

// Collection returns the active collection name (mostly for tests).
func (s *TodoService) Collection() string { return s.collection }

func (s *TodoService) col() *firestore.CollectionRef {
	return s.fs.Collection(s.collection)
}

// List returns the user's todos, ordered by position ascending.
func (s *TodoService) List(ctx context.Context, userID string, f ListFilter) ([]Todo, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	q := s.col().Where("userId", "==", userID).OrderBy("position", firestore.Asc)
	if f.Completed != nil {
		q = s.col().Where("userId", "==", userID).Where("completed", "==", *f.Completed).OrderBy("position", firestore.Asc)
	}
	if f.Priority != "" {
		q = s.col().Where("userId", "==", userID).Where("priority", "==", f.Priority).OrderBy("position", firestore.Asc)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	todos := make([]Todo, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list todos: %w", err)
		}
		var t Todo
		if err := doc.DataTo(&t); err != nil {
			return nil, fmt.Errorf("decode todo: %w", err)
		}
		t.ID = doc.Ref.ID
		todos = append(todos, t)
	}

	if f.Search != "" {
		search := strings.ToLower(f.Search)
		filtered := todos[:0]
		for _, t := range todos {
			if strings.Contains(strings.ToLower(t.Title), search) ||
				strings.Contains(strings.ToLower(t.Description), search) {
				filtered = append(filtered, t)
			}
		}
		todos = filtered
	}
	return todos, nil
}

// Get returns a single todo owned by userID. ErrNotFound if missing or
// owned by someone else (indistinguishable by design).
func (s *TodoService) Get(ctx context.Context, userID, id string) (Todo, error) {
	if userID == "" {
		return Todo{}, ErrUnauthenticated
	}
	doc, err := s.col().Doc(id).Get(ctx)
	if err != nil {
		return Todo{}, ErrNotFound
	}
	var t Todo
	if err := doc.DataTo(&t); err != nil {
		return Todo{}, fmt.Errorf("decode todo: %w", err)
	}
	t.ID = doc.Ref.ID
	if t.UserID != userID {
		return Todo{}, ErrNotFound
	}
	return t, nil
}

// CreateInput is the writable fields for a new todo.
type CreateInput struct {
	Title       string
	Description string
	Priority    string
	Category    string
	DueDate     *time.Time
}

// Create persists a new todo. Position is appended after the user's existing todos.
func (s *TodoService) Create(ctx context.Context, userID string, in CreateInput) (Todo, error) {
	if userID == "" {
		return Todo{}, ErrUnauthenticated
	}
	if strings.TrimSpace(in.Title) == "" {
		return Todo{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	priority := in.Priority
	if priority == "" {
		priority = "medium"
	}
	if priority != "low" && priority != "medium" && priority != "high" {
		return Todo{}, fmt.Errorf("%w: priority must be low, medium, or high", ErrInvalidInput)
	}

	// Find next position.
	maxPos := -1
	it := s.col().Where("userId", "==", userID).OrderBy("position", firestore.Desc).Limit(1).Documents(ctx)
	if doc, err := it.Next(); err == nil {
		var existing Todo
		doc.DataTo(&existing)
		maxPos = existing.Position
	}
	it.Stop()

	now := time.Now().UTC()
	t := Todo{
		Title:       strings.TrimSpace(in.Title),
		Description: in.Description,
		Priority:    priority,
		Category:    in.Category,
		DueDate:     in.DueDate,
		Completed:   false,
		Position:    maxPos + 1,
		UserID:      userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	t.FullText = buildFullText(t)

	ref, _, err := s.col().Add(ctx, t)
	if err != nil {
		return Todo{}, fmt.Errorf("add todo: %w", err)
	}
	t.ID = ref.ID
	return t, nil
}

// Patch applies a partial update. Fields `id`, `userId`, and `createdAt` are
// rejected. `updatedAt` is always overwritten with now.
func (s *TodoService) Patch(ctx context.Context, userID, id string, patch map[string]any) (Todo, error) {
	if userID == "" {
		return Todo{}, ErrUnauthenticated
	}
	if _, err := s.Get(ctx, userID, id); err != nil {
		return Todo{}, err
	}
	if patch == nil {
		patch = map[string]any{}
	}
	delete(patch, "id")
	delete(patch, "userId")
	delete(patch, "createdAt")
	if p, ok := patch["priority"].(string); ok {
		if p != "low" && p != "medium" && p != "high" {
			return Todo{}, fmt.Errorf("%w: priority must be low, medium, or high", ErrInvalidInput)
		}
	}
	if t, ok := patch["title"].(string); ok && strings.TrimSpace(t) == "" {
		return Todo{}, fmt.Errorf("%w: title cannot be empty", ErrInvalidInput)
	}
	patch["updatedAt"] = time.Now().UTC()

	updates := make([]firestore.Update, 0, len(patch))
	for k, v := range patch {
		updates = append(updates, firestore.Update{Path: k, Value: v})
	}
	if _, err := s.col().Doc(id).Update(ctx, updates); err != nil {
		return Todo{}, fmt.Errorf("patch todo: %w", err)
	}
	// If any of the indexed text fields changed, rebuild FullText in a
	// second write. Cheap and straightforward — avoids hand-merging the
	// patch's partial state with the existing record.
	if touchesText(patch) {
		cur, err := s.Get(ctx, userID, id)
		if err != nil {
			return Todo{}, err
		}
		ft := buildFullText(cur)
		if _, err := s.col().Doc(id).Update(ctx, []firestore.Update{{Path: "fullText", Value: ft}}); err != nil {
			return Todo{}, fmt.Errorf("patch fullText: %w", err)
		}
	}
	return s.Get(ctx, userID, id)
}

// touchesText reports whether a patch map modifies any field that feeds FullText.
func touchesText(patch map[string]any) bool {
	for _, k := range []string{"title", "description", "category"} {
		if _, ok := patch[k]; ok {
			return true
		}
	}
	return false
}

// Replace does a full replacement of the mutable fields of a todo (keeping
// ID, UserID, CreatedAt). Returns the post-state.
func (s *TodoService) Replace(ctx context.Context, userID, id string, in Todo) (Todo, error) {
	if userID == "" {
		return Todo{}, ErrUnauthenticated
	}
	existing, err := s.Get(ctx, userID, id)
	if err != nil {
		return Todo{}, err
	}
	if strings.TrimSpace(in.Title) == "" {
		return Todo{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	in.ID = id
	in.UserID = userID
	in.CreatedAt = existing.CreatedAt
	in.UpdatedAt = time.Now().UTC()
	in.FullText = buildFullText(in)
	if _, err := s.col().Doc(id).Set(ctx, in); err != nil {
		return Todo{}, fmt.Errorf("replace todo: %w", err)
	}
	return in, nil
}

// Delete removes a todo owned by userID.
func (s *TodoService) Delete(ctx context.Context, userID, id string) error {
	if userID == "" {
		return ErrUnauthenticated
	}
	if _, err := s.Get(ctx, userID, id); err != nil {
		return err
	}
	if _, err := s.col().Doc(id).Delete(ctx); err != nil {
		return fmt.Errorf("delete todo: %w", err)
	}
	return nil
}

// Search runs a full-text search against the user's todos using the
// FullText inverted index. It respects the same Completed / Priority
// filters as List so the UI can combine search with the status toggle.
// Behavior:
//
//   - Empty or all-stopword query → delegates to List(filter), mirroring
//     what the UI expects when the search box is cleared.
//   - Otherwise, issues a Firestore query:
//     userId == x [AND completed == ?] [AND priority == ?]
//     AND fullText array-contains-any <tokens>
//     Tokens are stemmed so "running" matches a todo indexed as "run".
//   - Firestore requires composite indexes for each equality combination
//     with an array-contains query; see firestore.indexes.json.
//   - Results are sorted by position ASC in-memory (adding OrderBy would
//     require yet another composite index).
//
// limit caps the returned slice (0 or <0 → no cap).
func (s *TodoService) Search(ctx context.Context, userID, query string, limit int, filter ListFilter) ([]Todo, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	tokens := search.BuildQuery(query)
	if len(tokens) == 0 {
		return s.List(ctx, userID, filter)
	}

	// Firestore caps array-contains-any at 30 tokens; BuildQuery clamps.
	q := s.col().
		Where("userId", "==", userID).
		Where("fullText", "array-contains-any", toAnySlice(tokens))
	if filter.Completed != nil {
		q = q.Where("completed", "==", *filter.Completed)
	}
	if filter.Priority != "" {
		q = q.Where("priority", "==", filter.Priority)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	// Non-nil empty slice so JSON serialization emits [] rather than null
	// when zero results match.
	todos := make([]Todo, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("search: %w", err)
		}
		var t Todo
		if err := doc.DataTo(&t); err != nil {
			return nil, fmt.Errorf("search decode: %w", err)
		}
		t.ID = doc.Ref.ID
		todos = append(todos, t)
	}

	sort.Slice(todos, func(i, j int) bool { return todos[i].Position < todos[j].Position })
	if limit > 0 && len(todos) > limit {
		todos = todos[:limit]
	}
	return todos, nil
}

func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// ReorderItem is a position update targeting a todo ID.
type ReorderItem struct {
	ID       string
	Position int
}

// Reorder applies position updates in a single batch. Items owned by other
// users are silently skipped so a single rogue id does not poison the batch.
func (s *TodoService) Reorder(ctx context.Context, userID string, items []ReorderItem) error {
	if userID == "" {
		return ErrUnauthenticated
	}
	batch := s.fs.Batch()
	now := time.Now().UTC()
	applied := 0
	for _, it := range items {
		ref := s.col().Doc(it.ID)
		doc, err := ref.Get(ctx)
		if err != nil {
			continue
		}
		var t Todo
		if err := doc.DataTo(&t); err != nil {
			continue
		}
		if t.UserID != userID {
			continue
		}
		batch.Update(ref, []firestore.Update{
			{Path: "position", Value: it.Position},
			{Path: "updatedAt", Value: now},
		})
		applied++
	}
	if applied == 0 {
		return nil
	}
	if _, err := batch.Commit(ctx); err != nil {
		return fmt.Errorf("reorder: %w", err)
	}
	return nil
}
