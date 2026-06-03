package agent

import (
	"context"
	"strings"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/mentat/qodo/api/services"
)

// ─── list_notes ──────────────────────────────────────────────────────────────

type ListNotesInput struct {
	Search string `json:"search,omitempty" jsonschema:"optional full-text search over title/body/tags"`
}

type NoteOut struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Snippet string `json:"snippet,omitempty"`
}

type ListNotesOutput struct {
	Notes  []NoteOut `json:"notes"`
	Count  int       `json:"count"`
	Notice string    `json:"notice,omitempty"`
}

func NewListNotesTool(svc *services.NoteService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in ListNotesInput) (ListNotesOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return ListNotesOutput{Notice: "internal: missing user context"}, nil
		}
		var items []services.Note
		var err error
		if strings.TrimSpace(in.Search) != "" {
			items, err = svc.Search(context.Background(), uid, in.Search, 0)
		} else {
			items, err = svc.List(context.Background(), uid)
		}
		if err != nil {
			return ListNotesOutput{Notice: err.Error()}, nil
		}
		out := ListNotesOutput{Notes: make([]NoteOut, 0, len(items)), Count: len(items)}
		for _, n := range items {
			out.Notes = append(out.Notes, NoteOut{ID: n.ID, Title: n.Title, Snippet: snippet(n.Body, 140)})
		}
		return out, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "list_notes",
		Description: "List the user's notes, or search them by title/body/tags. Returns id, title, and a snippet.",
	}, handler)
}

// ─── read_note ───────────────────────────────────────────────────────────────

type ReadNoteInput struct {
	ID string `json:"id" jsonschema:"the note id from list_notes"`
}

type ReadNoteOutput struct {
	Note *struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Body  string `json:"body"`
	} `json:"note,omitempty"`
	Error string `json:"error,omitempty"`
}

func NewReadNoteTool(svc *services.NoteService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in ReadNoteInput) (ReadNoteOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return ReadNoteOutput{Error: "internal: missing user context"}, nil
		}
		n, err := svc.Get(context.Background(), uid, in.ID)
		if err != nil {
			return ReadNoteOutput{Error: errString(err)}, nil
		}
		out := ReadNoteOutput{}
		out.Note = &struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Body  string `json:"body"`
		}{ID: n.ID, Title: n.Title, Body: n.Body}
		return out, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "read_note",
		Description: "Read the full markdown body of one note by id.",
	}, handler)
}

// ─── create_note ─────────────────────────────────────────────────────────────

type CreateNoteInput struct {
	Title string `json:"title" jsonschema:"note title"`
	Body  string `json:"body" jsonschema:"note body (markdown supported)"`
}

type CreateNoteOutput struct {
	Note  *NoteOut `json:"note,omitempty"`
	Error string   `json:"error,omitempty"`
}

func NewCreateNoteTool(svc *services.NoteService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in CreateNoteInput) (CreateNoteOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return CreateNoteOutput{Error: "internal: missing user context"}, nil
		}
		n, err := svc.Create(context.Background(), uid, services.NoteInput{Title: in.Title, Body: in.Body})
		if err != nil {
			return CreateNoteOutput{Error: errString(err)}, nil
		}
		return CreateNoteOutput{Note: &NoteOut{ID: n.ID, Title: n.Title, Snippet: snippet(n.Body, 140)}}, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "create_note",
		Description: "Create a new note (markdown supported in the body).",
	}, handler)
}
