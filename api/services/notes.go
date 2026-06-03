package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/mentat/qodo/api/search"
)

// Note is a markdown note. Body holds raw markdown; the frontend renders it.
type Note struct {
	ID        string    `json:"id" firestore:"-"`
	UserID    string    `json:"userId" firestore:"userId"`
	Title     string    `json:"title" firestore:"title"`
	Body      string    `json:"body" firestore:"body"`
	Tags      []string  `json:"tags,omitempty" firestore:"tags,omitempty"`
	FullText  []string  `json:"fullText,omitempty" firestore:"fullText,omitempty"`
	CreatedAt time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" firestore:"updatedAt"`
}

func buildNoteFullText(n Note) []string {
	return search.Build(n.Title, n.Body, strings.Join(n.Tags, " "))
}

// NoteService exposes note CRUD + search. Safe for concurrent use.
type NoteService struct {
	fs         *firestore.Client
	collection string
}

// NewNoteService constructs a service backed by the given Firestore client.
// The collection defaults to "notes".
func NewNoteService(fs *firestore.Client) *NoteService {
	return &NoteService{fs: fs, collection: "notes"}
}

// WithCollection returns a copy pointed at `name` (tests).
func (s *NoteService) WithCollection(name string) *NoteService {
	cp := *s
	cp.collection = name
	return &cp
}

// Collection returns the active collection name (tests).
func (s *NoteService) Collection() string { return s.collection }

func (s *NoteService) col() *firestore.CollectionRef { return s.fs.Collection(s.collection) }

// List returns the user's notes, most-recently-updated first.
func (s *NoteService) List(ctx context.Context, userID string) ([]Note, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	iter := s.col().Where("userId", "==", userID).OrderBy("updatedAt", firestore.Desc).Documents(ctx)
	return collectNotes(iter)
}

// Get returns a single note owned by userID.
func (s *NoteService) Get(ctx context.Context, userID, id string) (Note, error) {
	if userID == "" {
		return Note{}, ErrUnauthenticated
	}
	doc, err := s.col().Doc(id).Get(ctx)
	if err != nil {
		return Note{}, ErrNotFound
	}
	var n Note
	if err := doc.DataTo(&n); err != nil {
		return Note{}, fmt.Errorf("decode note: %w", err)
	}
	n.ID = doc.Ref.ID
	if n.UserID != userID {
		return Note{}, ErrNotFound
	}
	return n, nil
}

// NoteInput is the writable surface for a note.
type NoteInput struct {
	Title string
	Body  string
	Tags  []string
}

// Create persists a new note.
func (s *NoteService) Create(ctx context.Context, userID string, in NoteInput) (Note, error) {
	if userID == "" {
		return Note{}, ErrUnauthenticated
	}
	if strings.TrimSpace(in.Title) == "" && strings.TrimSpace(in.Body) == "" {
		return Note{}, fmt.Errorf("%w: a note needs a title or body", ErrInvalidInput)
	}
	now := time.Now().UTC()
	n := Note{
		UserID:    userID,
		Title:     strings.TrimSpace(in.Title),
		Body:      in.Body,
		Tags:      in.Tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
	n.FullText = buildNoteFullText(n)
	ref, _, err := s.col().Add(ctx, n)
	if err != nil {
		return Note{}, fmt.Errorf("add note: %w", err)
	}
	n.ID = ref.ID
	return n, nil
}

// Patch applies a partial update, rebuilding FullText when text changes.
func (s *NoteService) Patch(ctx context.Context, userID, id string, patch map[string]any) (Note, error) {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return Note{}, err
	}
	if patch == nil {
		patch = map[string]any{}
	}
	delete(patch, "id")
	delete(patch, "userId")
	delete(patch, "createdAt")
	patch["updatedAt"] = time.Now().UTC()

	updates := make([]firestore.Update, 0, len(patch))
	for k, v := range patch {
		updates = append(updates, firestore.Update{Path: k, Value: v})
	}
	if _, err := s.col().Doc(id).Update(ctx, updates); err != nil {
		return Note{}, fmt.Errorf("patch note: %w", err)
	}
	if touchesNoteText(patch) {
		cur, err := s.Get(ctx, userID, id)
		if err != nil {
			return Note{}, err
		}
		ft := buildNoteFullText(cur)
		if _, err := s.col().Doc(id).Update(ctx, []firestore.Update{{Path: "fullText", Value: ft}}); err != nil {
			return Note{}, fmt.Errorf("patch note fullText: %w", err)
		}
	}
	return s.Get(ctx, userID, id)
}

func touchesNoteText(patch map[string]any) bool {
	for _, k := range []string{"title", "body", "tags"} {
		if _, ok := patch[k]; ok {
			return true
		}
	}
	return false
}

// Replace does a full replacement of the mutable fields (keeping ID, UserID,
// CreatedAt).
func (s *NoteService) Replace(ctx context.Context, userID, id string, in NoteInput) (Note, error) {
	existing, err := s.Get(ctx, userID, id)
	if err != nil {
		return Note{}, err
	}
	if strings.TrimSpace(in.Title) == "" && strings.TrimSpace(in.Body) == "" {
		return Note{}, fmt.Errorf("%w: a note needs a title or body", ErrInvalidInput)
	}
	n := Note{
		ID:        id,
		UserID:    userID,
		Title:     strings.TrimSpace(in.Title),
		Body:      in.Body,
		Tags:      in.Tags,
		CreatedAt: existing.CreatedAt,
		UpdatedAt: time.Now().UTC(),
	}
	n.FullText = buildNoteFullText(n)
	if _, err := s.col().Doc(id).Set(ctx, n); err != nil {
		return Note{}, fmt.Errorf("replace note: %w", err)
	}
	return n, nil
}

// Delete removes a note owned by userID.
func (s *NoteService) Delete(ctx context.Context, userID, id string) error {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return err
	}
	if _, err := s.col().Doc(id).Delete(ctx); err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	return nil
}

// Search runs a stemmed full-text search over the user's notes. An empty or
// all-stopword query falls through to List.
func (s *NoteService) Search(ctx context.Context, userID, query string, limit int) ([]Note, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	tokens := search.BuildQuery(query)
	if len(tokens) == 0 {
		return s.List(ctx, userID)
	}
	iter := s.col().
		Where("userId", "==", userID).
		Where("fullText", "array-contains-any", toAnySlice(tokens)).
		Documents(ctx)
	out, err := collectNotes(iter)
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func collectNotes(iter *firestore.DocumentIterator) ([]Note, error) {
	defer iter.Stop()
	out := make([]Note, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list notes: %w", err)
		}
		var n Note
		if err := doc.DataTo(&n); err != nil {
			return nil, fmt.Errorf("decode note: %w", err)
		}
		n.ID = doc.Ref.ID
		out = append(out, n)
	}
	return out, nil
}
