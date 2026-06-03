package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// Email is one message in the in-app mail client. There is no real SMTP —
// `From`/`To` are opaque handles ("me" for the user, a character's address
// otherwise). Threads are derived client-side by grouping on ThreadID, so
// there is no separate threads collection to keep in sync.
type Email struct {
	ID          string    `json:"id" firestore:"-"`
	UserID      string    `json:"userId" firestore:"userId"`
	ThreadID    string    `json:"threadId" firestore:"threadId"`
	From        string    `json:"from" firestore:"from"`
	FromName    string    `json:"fromName" firestore:"fromName"`
	To          string    `json:"to" firestore:"to"`
	Subject     string    `json:"subject" firestore:"subject"`
	Body        string    `json:"body" firestore:"body"`
	Direction   string    `json:"direction" firestore:"direction"` // "inbound" | "outbound"
	Read        bool      `json:"read" firestore:"read"`
	CharacterID string    `json:"characterId,omitempty" firestore:"characterId,omitempty"`
	CreatedAt   time.Time `json:"createdAt" firestore:"createdAt"`
}

const (
	// DirectionInbound marks mail the user received (from a character).
	DirectionInbound = "inbound"
	// DirectionOutbound marks mail the user sent.
	DirectionOutbound = "outbound"
	// UserAddr is the opaque handle for the signed-in user.
	UserAddr = "me"
)

// Publisher publishes an async job (e.g. to Cloud Pub/Sub). EmailService
// uses it to kick off an in-character reply after the user sends mail to a
// character. A nil Publisher (or one that returns an error) is non-fatal:
// the email is still stored; only the auto-reply is skipped.
type Publisher interface {
	Publish(ctx context.Context, attrs map[string]string, data []byte) (string, error)
}

// EmailService exposes mail CRUD. Safe for concurrent use.
type EmailService struct {
	fs         *firestore.Client
	collection string
	publisher  Publisher
}

// NewEmailService constructs a service backed by the given Firestore client.
// The collection defaults to "emails".
func NewEmailService(fs *firestore.Client) *EmailService {
	return &EmailService{fs: fs, collection: "emails"}
}

// WithCollection returns a copy pointed at `name` (tests).
func (s *EmailService) WithCollection(name string) *EmailService {
	cp := *s
	cp.collection = name
	return &cp
}

// WithPublisher returns a copy that publishes reply jobs via p.
func (s *EmailService) WithPublisher(p Publisher) *EmailService {
	cp := *s
	cp.publisher = p
	return &cp
}

// Collection returns the active collection name (tests).
func (s *EmailService) Collection() string { return s.collection }

func (s *EmailService) col() *firestore.CollectionRef { return s.fs.Collection(s.collection) }

// ListInbox returns the user's most recent emails, newest first. The client
// derives threads + unread counts from this set. limit defaults to 200.
func (s *EmailService) ListInbox(ctx context.Context, userID string, limit int) ([]Email, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	iter := s.col().
		Where("userId", "==", userID).
		OrderBy("createdAt", firestore.Desc).
		Limit(limit).
		Documents(ctx)
	return collectEmails(iter)
}

// ListThread returns every message in a thread owned by userID, oldest first.
func (s *EmailService) ListThread(ctx context.Context, userID, threadID string) ([]Email, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	iter := s.col().
		Where("userId", "==", userID).
		Where("threadId", "==", threadID).
		OrderBy("createdAt", firestore.Asc).
		Documents(ctx)
	return collectEmails(iter)
}

// Get returns a single email owned by userID.
func (s *EmailService) Get(ctx context.Context, userID, id string) (Email, error) {
	if userID == "" {
		return Email{}, ErrUnauthenticated
	}
	doc, err := s.col().Doc(id).Get(ctx)
	if err != nil {
		return Email{}, ErrNotFound
	}
	var e Email
	if err := doc.DataTo(&e); err != nil {
		return Email{}, fmt.Errorf("decode email: %w", err)
	}
	e.ID = doc.Ref.ID
	if e.UserID != userID {
		return Email{}, ErrNotFound
	}
	return e, nil
}

// SendInput is the writable surface for an outbound email.
type SendInput struct {
	To          string // recipient handle (a character's address)
	ToName      string
	Subject     string
	Body        string
	ThreadID    string // empty starts a new thread
	CharacterID string // recipient persona; when set, an auto-reply is published
}

// Send persists an outbound email. When CharacterID is set and a Publisher is
// configured, it publishes a reply job so the character can answer
// asynchronously. Publish failures are swallowed (the email is still sent).
func (s *EmailService) Send(ctx context.Context, userID string, in SendInput) (Email, error) {
	if userID == "" {
		return Email{}, ErrUnauthenticated
	}
	if strings.TrimSpace(in.Body) == "" && strings.TrimSpace(in.Subject) == "" {
		return Email{}, fmt.Errorf("%w: an email needs a subject or body", ErrInvalidInput)
	}
	threadID := in.ThreadID
	if threadID == "" {
		threadID = uuid.NewString()
	}
	e := Email{
		UserID:      userID,
		ThreadID:    threadID,
		From:        UserAddr,
		FromName:    "You",
		To:          in.To,
		Subject:     strings.TrimSpace(in.Subject),
		Body:        in.Body,
		Direction:   DirectionOutbound,
		Read:        true,
		CharacterID: in.CharacterID,
		CreatedAt:   time.Now().UTC(),
	}
	ref, _, err := s.col().Add(ctx, e)
	if err != nil {
		return Email{}, fmt.Errorf("send email: %w", err)
	}
	e.ID = ref.ID

	if shouldPublishReply(in.CharacterID) && s.publisher != nil {
		_ = s.publishReply(ctx, e)
	}
	return e, nil
}

// shouldPublishReply reports whether an outbound email warrants an auto-reply:
// only when it targets a known character.
func shouldPublishReply(characterID string) bool {
	return strings.TrimSpace(characterID) != ""
}

func (s *EmailService) publishReply(ctx context.Context, e Email) error {
	attrs := map[string]string{
		"kind":        "reply",
		"userId":      e.UserID,
		"threadId":    e.ThreadID,
		"emailId":     e.ID,
		"characterId": e.CharacterID,
	}
	data := []byte(fmt.Sprintf(
		`{"kind":"reply","userId":%q,"threadId":%q,"emailId":%q,"characterId":%q}`,
		e.UserID, e.ThreadID, e.ID, e.CharacterID))
	_, err := s.publisher.Publish(ctx, attrs, data)
	return err
}

// InboundInput is the writable surface for an inbound email (from a
// character). Used by the seeder, the Pub/Sub reply worker, and the drip job.
type InboundInput struct {
	From        string
	FromName    string
	Subject     string
	Body        string
	ThreadID    string // empty starts a new thread
	CharacterID string
	CreatedAt   time.Time // zero → now
}

// CreateInbound persists an inbound email. No reply is published.
func (s *EmailService) CreateInbound(ctx context.Context, userID string, in InboundInput) (Email, error) {
	if userID == "" {
		return Email{}, ErrUnauthenticated
	}
	threadID := in.ThreadID
	if threadID == "" {
		threadID = uuid.NewString()
	}
	created := in.CreatedAt
	if created.IsZero() {
		created = time.Now().UTC()
	}
	e := Email{
		UserID:      userID,
		ThreadID:    threadID,
		From:        in.From,
		FromName:    in.FromName,
		To:          UserAddr,
		Subject:     strings.TrimSpace(in.Subject),
		Body:        in.Body,
		Direction:   DirectionInbound,
		Read:        false,
		CharacterID: in.CharacterID,
		CreatedAt:   created.UTC(),
	}
	ref, _, err := s.col().Add(ctx, e)
	if err != nil {
		return Email{}, fmt.Errorf("create inbound: %w", err)
	}
	e.ID = ref.ID
	return e, nil
}

// MarkRead flips read=true on an email owned by userID.
func (s *EmailService) MarkRead(ctx context.Context, userID, id string) (Email, error) {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return Email{}, err
	}
	if _, err := s.col().Doc(id).Update(ctx, []firestore.Update{{Path: "read", Value: true}}); err != nil {
		return Email{}, fmt.Errorf("mark read: %w", err)
	}
	return s.Get(ctx, userID, id)
}

// Delete removes an email owned by userID.
func (s *EmailService) Delete(ctx context.Context, userID, id string) error {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return err
	}
	if _, err := s.col().Doc(id).Delete(ctx); err != nil {
		return fmt.Errorf("delete email: %w", err)
	}
	return nil
}

func collectEmails(iter *firestore.DocumentIterator) ([]Email, error) {
	defer iter.Stop()
	out := make([]Email, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list emails: %w", err)
		}
		var e Email
		if err := doc.DataTo(&e); err != nil {
			return nil, fmt.Errorf("decode email: %w", err)
		}
		e.ID = doc.Ref.ID
		out = append(out, e)
	}
	return out, nil
}
