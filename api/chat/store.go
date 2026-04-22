// Package chat persists chat history between users and Marvin.
package chat

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// Role is the author of a chat message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
	RoleSystem    Role = "system" // reserved for internal events
)

// Message is a persisted chat message.
type Message struct {
	ID         string    `json:"id" firestore:"-"`
	UserID     string    `json:"userId" firestore:"userId"`
	Role       Role      `json:"role" firestore:"role"`
	Content    string    `json:"content" firestore:"content"`
	ToolName   string    `json:"toolName,omitempty" firestore:"toolName,omitempty"`
	ToolArgs   string    `json:"toolArgs,omitempty" firestore:"toolArgs,omitempty"`
	ToolResult string    `json:"toolResult,omitempty" firestore:"toolResult,omitempty"`
	// Screened indicates a message was short-circuited by the intent screener.
	Screened   bool      `json:"screened,omitempty" firestore:"screened,omitempty"`
	CreatedAt  time.Time `json:"createdAt" firestore:"createdAt"`
}

// Store persists chat messages.
type Store struct {
	fs         *firestore.Client
	collection string
}

// NewStore returns a Store using the "chatMessages" collection.
func NewStore(fs *firestore.Client) *Store {
	return &Store{fs: fs, collection: "chatMessages"}
}

// WithCollection returns a copy pointed at `name` (tests).
func (s *Store) WithCollection(name string) *Store { cp := *s; cp.collection = name; return &cp }

// Collection returns the active collection name.
func (s *Store) Collection() string { return s.collection }

func (s *Store) col() *firestore.CollectionRef { return s.fs.Collection(s.collection) }

// Append writes a new message, returning it with ID populated.
func (s *Store) Append(ctx context.Context, m Message) (Message, error) {
	if m.UserID == "" {
		return Message{}, fmt.Errorf("chat: userId required")
	}
	if m.Role == "" {
		return Message{}, fmt.Errorf("chat: role required")
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	ref, _, err := s.col().Add(ctx, m)
	if err != nil {
		return Message{}, fmt.Errorf("chat append: %w", err)
	}
	m.ID = ref.ID
	return m, nil
}

// History returns the most recent N messages for the user, oldest first.
// If limit <= 0, it defaults to 50 and caps at 500.
func (s *Store) History(ctx context.Context, userID string, limit int) ([]Message, error) {
	if userID == "" {
		return nil, fmt.Errorf("chat: userId required")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	iter := s.col().
		Where("userId", "==", userID).
		OrderBy("createdAt", firestore.Desc).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	var msgs []Message
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("chat history: %w", err)
		}
		var m Message
		if err := doc.DataTo(&m); err != nil {
			return nil, fmt.Errorf("chat decode: %w", err)
		}
		m.ID = doc.Ref.ID
		msgs = append(msgs, m)
	}
	// reverse to oldest-first
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// Clear deletes all messages for a user.
func (s *Store) Clear(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("chat: userId required")
	}
	iter := s.col().Where("userId", "==", userID).Documents(ctx)
	defer iter.Stop()
	batch := s.fs.Batch()
	count := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("chat clear iter: %w", err)
		}
		batch.Delete(doc.Ref)
		count++
		// commit in chunks of 400 (Firestore batch limit is 500).
		if count%400 == 0 {
			if _, err := batch.Commit(ctx); err != nil {
				return fmt.Errorf("chat clear commit: %w", err)
			}
			batch = s.fs.Batch()
		}
	}
	if count%400 != 0 {
		if _, err := batch.Commit(ctx); err != nil {
			return fmt.Errorf("chat clear commit: %w", err)
		}
	}
	return nil
}
