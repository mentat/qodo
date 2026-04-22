package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/mentat/qodo/api/services"
)

// The userID for a tool invocation must flow from the HTTP request into the
// tool handler so Marvin can only ever act on the signed-in user's data.
// ADK's tool.Context doesn't expose the calling user; we pass it in via
// context.WithValue before invoking the runner.

type userIDKey struct{}

// WithUserID attaches the caller's userID to the context used by the runner.
func WithUserID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, userIDKey{}, uid)
}

// UserIDFromContext extracts the userID set by WithUserID. Returns "" if unset.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey{}).(string)
	return v
}

// ─── list_todos ──────────────────────────────────────────────────────────────

type ListTodosInput struct {
	Completed *bool  `json:"completed,omitempty" jsonschema:"filter by completion status"`
	Priority  string `json:"priority,omitempty" jsonschema:"filter by priority: low | medium | high"`
	Search    string `json:"search,omitempty" jsonschema:"case-insensitive substring match on title/description"`
}

type TodoOut struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Completed   bool   `json:"completed"`
	Priority    string `json:"priority"`
	Category    string `json:"category,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
	Position    int    `json:"position"`
}

type ListTodosOutput struct {
	Todos  []TodoOut `json:"todos"`
	Count  int       `json:"count"`
	Notice string    `json:"notice,omitempty"`
}

func NewListTodosTool(svc *services.TodoService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in ListTodosInput) (ListTodosOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return ListTodosOutput{Notice: "internal: missing user context"}, nil
		}
		filter := services.ListFilter{Priority: in.Priority, Search: in.Search}
		if in.Completed != nil {
			filter.Completed = in.Completed
		}
		items, err := svc.List(context.Background(), uid, filter)
		if err != nil {
			return ListTodosOutput{Notice: err.Error()}, nil
		}
		out := ListTodosOutput{Todos: make([]TodoOut, 0, len(items)), Count: len(items)}
		for _, it := range items {
			out.Todos = append(out.Todos, toTodoOut(it))
		}
		return out, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "list_todos",
		Description: "List the current user's todos. Returns id, title, description, completed, priority, category, due_date, and position. All fields optional filters.",
	}, handler)
}

// ─── create_todo ─────────────────────────────────────────────────────────────

type CreateTodoInput struct {
	Title       string `json:"title" jsonschema:"the todo title, required"`
	Description string `json:"description,omitempty"`
	Priority    string `json:"priority,omitempty" jsonschema:"low | medium | high (default medium)"`
	Category    string `json:"category,omitempty"`
	DueDate     string `json:"due_date,omitempty" jsonschema:"ISO 8601 date (YYYY-MM-DD) or datetime"`
}

type CreateTodoOutput struct {
	Todo   *TodoOut `json:"todo,omitempty"`
	Error  string   `json:"error,omitempty"`
}

func NewCreateTodoTool(svc *services.TodoService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in CreateTodoInput) (CreateTodoOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return CreateTodoOutput{Error: "internal: missing user context"}, nil
		}
		due, err := parseDueDate(in.DueDate)
		if err != nil {
			return CreateTodoOutput{Error: "due_date: " + err.Error()}, nil
		}
		t, err := svc.Create(context.Background(), uid, services.CreateInput{
			Title:       in.Title,
			Description: in.Description,
			Priority:    in.Priority,
			Category:    in.Category,
			DueDate:     due,
		})
		if err != nil {
			return CreateTodoOutput{Error: err.Error()}, nil
		}
		out := toTodoOut(t)
		return CreateTodoOutput{Todo: &out}, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "create_todo",
		Description: "Create a new todo for the current user. Only 'title' is required. Priority defaults to 'medium'. Returns the created todo.",
	}, handler)
}

// ─── update_todo ─────────────────────────────────────────────────────────────

type UpdateTodoInput struct {
	ID          string `json:"id" jsonschema:"the id of the todo to update, from list_todos"`
	Title       string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Completed   *bool   `json:"completed,omitempty"`
	Priority    string  `json:"priority,omitempty" jsonschema:"low | medium | high"`
	Category    *string `json:"category,omitempty"`
	DueDate     string  `json:"due_date,omitempty" jsonschema:"ISO 8601 date/datetime, or empty to leave unchanged"`
	ClearDueDate bool   `json:"clear_due_date,omitempty" jsonschema:"set to true to explicitly clear the due date"`
}

type UpdateTodoOutput struct {
	Todo  *TodoOut `json:"todo,omitempty"`
	Error string   `json:"error,omitempty"`
}

func NewUpdateTodoTool(svc *services.TodoService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in UpdateTodoInput) (UpdateTodoOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return UpdateTodoOutput{Error: "internal: missing user context"}, nil
		}
		if strings.TrimSpace(in.ID) == "" {
			return UpdateTodoOutput{Error: "id is required"}, nil
		}
		patch := map[string]any{}
		if in.Title != "" {
			patch["title"] = in.Title
		}
		if in.Description != nil {
			patch["description"] = *in.Description
		}
		if in.Completed != nil {
			patch["completed"] = *in.Completed
		}
		if in.Priority != "" {
			patch["priority"] = in.Priority
		}
		if in.Category != nil {
			patch["category"] = *in.Category
		}
		if in.ClearDueDate {
			patch["dueDate"] = nil
		} else if in.DueDate != "" {
			due, err := parseDueDate(in.DueDate)
			if err != nil {
				return UpdateTodoOutput{Error: "due_date: " + err.Error()}, nil
			}
			patch["dueDate"] = due
		}
		if len(patch) == 0 {
			return UpdateTodoOutput{Error: "no fields to update"}, nil
		}
		t, err := svc.Patch(context.Background(), uid, in.ID, patch)
		if err != nil {
			return UpdateTodoOutput{Error: errString(err)}, nil
		}
		out := toTodoOut(t)
		return UpdateTodoOutput{Todo: &out}, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "update_todo",
		Description: "Update fields on an existing todo. Only provide the fields you want to change. Use completed=true to mark done, completed=false to re-open. Use clear_due_date=true to remove the due date.",
	}, handler)
}

// ─── delete_todo ─────────────────────────────────────────────────────────────

type DeleteTodoInput struct {
	ID string `json:"id" jsonschema:"the id of the todo to delete"`
}

type DeleteTodoOutput struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func NewDeleteTodoTool(svc *services.TodoService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in DeleteTodoInput) (DeleteTodoOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return DeleteTodoOutput{Error: "internal: missing user context"}, nil
		}
		if strings.TrimSpace(in.ID) == "" {
			return DeleteTodoOutput{Error: "id is required"}, nil
		}
		if err := svc.Delete(context.Background(), uid, in.ID); err != nil {
			return DeleteTodoOutput{Error: errString(err)}, nil
		}
		return DeleteTodoOutput{OK: true}, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "delete_todo",
		Description: "Delete one of the current user's todos by id.",
	}, handler)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func toTodoOut(t services.Todo) TodoOut {
	out := TodoOut{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Completed:   t.Completed,
		Priority:    t.Priority,
		Category:    t.Category,
		Position:    t.Position,
	}
	if t.DueDate != nil {
		out.DueDate = t.DueDate.UTC().Format(time.RFC3339)
	}
	return out
}

// parseDueDate accepts RFC3339, YYYY-MM-DD, or YYYY-MM-DDTHH:MM:SS.
// Returns nil with nil error for empty input.
func parseDueDate(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			t = t.UTC()
			return &t, nil
		}
	}
	return nil, fmt.Errorf("could not parse %q as an ISO 8601 date", s)
}

func errString(err error) string {
	if errors.Is(err, services.ErrNotFound) {
		return "todo not found"
	}
	return err.Error()
}
