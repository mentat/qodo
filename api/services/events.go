package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// Event is a calendar event. The frontend listens to the user's full event
// set in real time and slices it into month/week/day views client-side; the
// range query here backs the agent's calendar tool and the agenda endpoint.
type Event struct {
	ID          string    `json:"id" firestore:"-"`
	UserID      string    `json:"userId" firestore:"userId"`
	Title       string    `json:"title" firestore:"title"`
	Description string    `json:"description" firestore:"description"`
	Location    string    `json:"location" firestore:"location"`
	Start       time.Time `json:"start" firestore:"start"`
	End         time.Time `json:"end" firestore:"end"`
	AllDay      bool      `json:"allDay" firestore:"allDay"`
	Color       string    `json:"color" firestore:"color"`
	CharacterID string    `json:"characterId,omitempty" firestore:"characterId,omitempty"`
	CreatedAt   time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" firestore:"updatedAt"`
}

// EventService exposes calendar CRUD. Safe for concurrent use.
type EventService struct {
	fs         *firestore.Client
	collection string
}

// NewEventService constructs a service backed by the given Firestore client.
// The collection defaults to "events".
func NewEventService(fs *firestore.Client) *EventService {
	return &EventService{fs: fs, collection: "events"}
}

// WithCollection returns a copy pointed at `name` (tests).
func (s *EventService) WithCollection(name string) *EventService {
	cp := *s
	cp.collection = name
	return &cp
}

// Collection returns the active collection name (tests).
func (s *EventService) Collection() string { return s.collection }

func (s *EventService) col() *firestore.CollectionRef { return s.fs.Collection(s.collection) }

// List returns all of the user's events, ordered by start ascending.
func (s *EventService) List(ctx context.Context, userID string) ([]Event, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	iter := s.col().Where("userId", "==", userID).OrderBy("start", firestore.Asc).Documents(ctx)
	return collectEvents(iter)
}

// ListRange returns the user's events whose start falls in [from, to),
// ordered by start ascending.
func (s *EventService) ListRange(ctx context.Context, userID string, from, to time.Time) ([]Event, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	iter := s.col().
		Where("userId", "==", userID).
		Where("start", ">=", from).
		Where("start", "<", to).
		OrderBy("start", firestore.Asc).
		Documents(ctx)
	return collectEvents(iter)
}

// Get returns a single event owned by userID.
func (s *EventService) Get(ctx context.Context, userID, id string) (Event, error) {
	if userID == "" {
		return Event{}, ErrUnauthenticated
	}
	doc, err := s.col().Doc(id).Get(ctx)
	if err != nil {
		return Event{}, ErrNotFound
	}
	var e Event
	if err := doc.DataTo(&e); err != nil {
		return Event{}, fmt.Errorf("decode event: %w", err)
	}
	e.ID = doc.Ref.ID
	if e.UserID != userID {
		return Event{}, ErrNotFound
	}
	return e, nil
}

// EventInput is the writable surface for an event.
type EventInput struct {
	Title       string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	AllDay      bool
	Color       string
	CharacterID string
}

// Create persists a new event.
func (s *EventService) Create(ctx context.Context, userID string, in EventInput) (Event, error) {
	if userID == "" {
		return Event{}, ErrUnauthenticated
	}
	if strings.TrimSpace(in.Title) == "" {
		return Event{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if in.Start.IsZero() {
		return Event{}, fmt.Errorf("%w: start is required", ErrInvalidInput)
	}
	if in.End.IsZero() {
		in.End = in.Start.Add(time.Hour)
	}
	if in.End.Before(in.Start) {
		return Event{}, fmt.Errorf("%w: end must be after start", ErrInvalidInput)
	}
	now := time.Now().UTC()
	e := Event{
		UserID:      userID,
		Title:       strings.TrimSpace(in.Title),
		Description: in.Description,
		Location:    in.Location,
		Start:       in.Start.UTC(),
		End:         in.End.UTC(),
		AllDay:      in.AllDay,
		Color:       in.Color,
		CharacterID: in.CharacterID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	ref, _, err := s.col().Add(ctx, e)
	if err != nil {
		return Event{}, fmt.Errorf("add event: %w", err)
	}
	e.ID = ref.ID
	return e, nil
}

// Patch applies a partial update. Protected fields are stripped.
func (s *EventService) Patch(ctx context.Context, userID, id string, patch map[string]any) (Event, error) {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return Event{}, err
	}
	if patch == nil {
		patch = map[string]any{}
	}
	delete(patch, "id")
	delete(patch, "userId")
	delete(patch, "createdAt")
	if t, ok := patch["title"].(string); ok && strings.TrimSpace(t) == "" {
		return Event{}, fmt.Errorf("%w: title cannot be empty", ErrInvalidInput)
	}
	patch["updatedAt"] = time.Now().UTC()

	updates := make([]firestore.Update, 0, len(patch))
	for k, v := range patch {
		updates = append(updates, firestore.Update{Path: k, Value: v})
	}
	if _, err := s.col().Doc(id).Update(ctx, updates); err != nil {
		return Event{}, fmt.Errorf("patch event: %w", err)
	}
	return s.Get(ctx, userID, id)
}

// Replace does a full replacement of the mutable fields (keeping ID, UserID,
// CreatedAt).
func (s *EventService) Replace(ctx context.Context, userID, id string, in EventInput) (Event, error) {
	existing, err := s.Get(ctx, userID, id)
	if err != nil {
		return Event{}, err
	}
	if strings.TrimSpace(in.Title) == "" {
		return Event{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if in.Start.IsZero() {
		return Event{}, fmt.Errorf("%w: start is required", ErrInvalidInput)
	}
	if in.End.IsZero() {
		in.End = in.Start.Add(time.Hour)
	}
	e := Event{
		ID:          id,
		UserID:      userID,
		Title:       strings.TrimSpace(in.Title),
		Description: in.Description,
		Location:    in.Location,
		Start:       in.Start.UTC(),
		End:         in.End.UTC(),
		AllDay:      in.AllDay,
		Color:       in.Color,
		CharacterID: in.CharacterID,
		CreatedAt:   existing.CreatedAt,
		UpdatedAt:   time.Now().UTC(),
	}
	if _, err := s.col().Doc(id).Set(ctx, e); err != nil {
		return Event{}, fmt.Errorf("replace event: %w", err)
	}
	return e, nil
}

// Move reschedules an event to a new start/end (a thin Patch for the agent).
// A zero newEnd preserves the original duration.
func (s *EventService) Move(ctx context.Context, userID, id string, newStart, newEnd time.Time) (Event, error) {
	cur, err := s.Get(ctx, userID, id)
	if err != nil {
		return Event{}, err
	}
	if newStart.IsZero() {
		return Event{}, fmt.Errorf("%w: new start is required", ErrInvalidInput)
	}
	if newEnd.IsZero() {
		newEnd = newStart.Add(cur.End.Sub(cur.Start))
	}
	return s.Patch(ctx, userID, id, map[string]any{
		"start": newStart.UTC(),
		"end":   newEnd.UTC(),
	})
}

// Delete removes an event owned by userID.
func (s *EventService) Delete(ctx context.Context, userID, id string) error {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return err
	}
	if _, err := s.col().Doc(id).Delete(ctx); err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	return nil
}

func collectEvents(iter *firestore.DocumentIterator) ([]Event, error) {
	defer iter.Stop()
	out := make([]Event, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list events: %w", err)
		}
		var e Event
		if err := doc.DataTo(&e); err != nil {
			return nil, fmt.Errorf("decode event: %w", err)
		}
		e.ID = doc.Ref.ID
		out = append(out, e)
	}
	return out, nil
}
