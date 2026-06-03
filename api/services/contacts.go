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

// Contact is an address-book entry. The email characters seed the directory,
// so a Contact may carry a CharacterID linking it back to a persona.
type Contact struct {
	ID          string    `json:"id" firestore:"-"`
	UserID      string    `json:"userId" firestore:"userId"`
	Name        string    `json:"name" firestore:"name"`
	Email       string    `json:"email" firestore:"email"`
	Phone       string    `json:"phone" firestore:"phone"`
	Company     string    `json:"company" firestore:"company"`
	Notes       string    `json:"notes" firestore:"notes"`
	CharacterID string    `json:"characterId,omitempty" firestore:"characterId,omitempty"`
	FullText    []string  `json:"fullText,omitempty" firestore:"fullText,omitempty"`
	CreatedAt   time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" firestore:"updatedAt"`
}

func buildContactFullText(c Contact) []string {
	return search.Build(c.Name, c.Company, c.Email, c.Notes)
}

// ContactService exposes contact CRUD + search. Safe for concurrent use.
type ContactService struct {
	fs         *firestore.Client
	collection string
}

// NewContactService constructs a service backed by the given Firestore client.
// The collection defaults to "contacts".
func NewContactService(fs *firestore.Client) *ContactService {
	return &ContactService{fs: fs, collection: "contacts"}
}

// WithCollection returns a copy pointed at `name` (tests).
func (s *ContactService) WithCollection(name string) *ContactService {
	cp := *s
	cp.collection = name
	return &cp
}

// Collection returns the active collection name (tests).
func (s *ContactService) Collection() string { return s.collection }

func (s *ContactService) col() *firestore.CollectionRef { return s.fs.Collection(s.collection) }

// List returns the user's contacts, ordered by name ascending.
func (s *ContactService) List(ctx context.Context, userID string) ([]Contact, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	iter := s.col().Where("userId", "==", userID).OrderBy("name", firestore.Asc).Documents(ctx)
	return collectContacts(iter)
}

// Get returns a single contact owned by userID.
func (s *ContactService) Get(ctx context.Context, userID, id string) (Contact, error) {
	if userID == "" {
		return Contact{}, ErrUnauthenticated
	}
	doc, err := s.col().Doc(id).Get(ctx)
	if err != nil {
		return Contact{}, ErrNotFound
	}
	var c Contact
	if err := doc.DataTo(&c); err != nil {
		return Contact{}, fmt.Errorf("decode contact: %w", err)
	}
	c.ID = doc.Ref.ID
	if c.UserID != userID {
		return Contact{}, ErrNotFound
	}
	return c, nil
}

// ContactInput is the writable surface for a contact.
type ContactInput struct {
	Name        string
	Email       string
	Phone       string
	Company     string
	Notes       string
	CharacterID string
}

// Create persists a new contact.
func (s *ContactService) Create(ctx context.Context, userID string, in ContactInput) (Contact, error) {
	if userID == "" {
		return Contact{}, ErrUnauthenticated
	}
	if strings.TrimSpace(in.Name) == "" {
		return Contact{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	now := time.Now().UTC()
	c := Contact{
		UserID:      userID,
		Name:        strings.TrimSpace(in.Name),
		Email:       in.Email,
		Phone:       in.Phone,
		Company:     in.Company,
		Notes:       in.Notes,
		CharacterID: in.CharacterID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	c.FullText = buildContactFullText(c)
	ref, _, err := s.col().Add(ctx, c)
	if err != nil {
		return Contact{}, fmt.Errorf("add contact: %w", err)
	}
	c.ID = ref.ID
	return c, nil
}

// Patch applies a partial update, rebuilding FullText when text changes.
func (s *ContactService) Patch(ctx context.Context, userID, id string, patch map[string]any) (Contact, error) {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return Contact{}, err
	}
	if patch == nil {
		patch = map[string]any{}
	}
	delete(patch, "id")
	delete(patch, "userId")
	delete(patch, "createdAt")
	if n, ok := patch["name"].(string); ok && strings.TrimSpace(n) == "" {
		return Contact{}, fmt.Errorf("%w: name cannot be empty", ErrInvalidInput)
	}
	patch["updatedAt"] = time.Now().UTC()

	updates := make([]firestore.Update, 0, len(patch))
	for k, v := range patch {
		updates = append(updates, firestore.Update{Path: k, Value: v})
	}
	if _, err := s.col().Doc(id).Update(ctx, updates); err != nil {
		return Contact{}, fmt.Errorf("patch contact: %w", err)
	}
	if touchesContactText(patch) {
		cur, err := s.Get(ctx, userID, id)
		if err != nil {
			return Contact{}, err
		}
		ft := buildContactFullText(cur)
		if _, err := s.col().Doc(id).Update(ctx, []firestore.Update{{Path: "fullText", Value: ft}}); err != nil {
			return Contact{}, fmt.Errorf("patch contact fullText: %w", err)
		}
	}
	return s.Get(ctx, userID, id)
}

func touchesContactText(patch map[string]any) bool {
	for _, k := range []string{"name", "company", "email", "notes"} {
		if _, ok := patch[k]; ok {
			return true
		}
	}
	return false
}

// Delete removes a contact owned by userID.
func (s *ContactService) Delete(ctx context.Context, userID, id string) error {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return err
	}
	if _, err := s.col().Doc(id).Delete(ctx); err != nil {
		return fmt.Errorf("delete contact: %w", err)
	}
	return nil
}

// Search runs a stemmed full-text search over the user's contacts. An empty
// or all-stopword query falls through to List.
func (s *ContactService) Search(ctx context.Context, userID, query string, limit int) ([]Contact, error) {
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
	out, err := collectContacts(iter)
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func collectContacts(iter *firestore.DocumentIterator) ([]Contact, error) {
	defer iter.Stop()
	out := make([]Contact, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list contacts: %w", err)
		}
		var c Contact
		if err := doc.DataTo(&c); err != nil {
			return nil, fmt.Errorf("decode contact: %w", err)
		}
		c.ID = doc.Ref.ID
		out = append(out, c)
	}
	return out, nil
}
