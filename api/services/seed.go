package services

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mentat/qodo/api/seed"
)

const seedMarkerCollection = "userSeeds"

// SeedService plants (and resets) the witty demo content for a user: the
// email cast as contacts, a starter inbox, sample events, and notes. Todos
// are intentionally left untouched — they're the user's own domain.
type SeedService struct {
	fs       *firestore.Client
	emails   *EmailService
	events   *EventService
	contacts *ContactService
	notes    *NoteService
}

// NewSeedService wires the seeder to the domain services it writes through.
func NewSeedService(fs *firestore.Client, email *EmailService, event *EventService, contact *ContactService, note *NoteService) *SeedService {
	return &SeedService{fs: fs, emails: email, events: event, contacts: contact, notes: note}
}

// Seed plants demo content for a user, exactly once. It returns true if it
// seeded, false if the user was already seeded. Idempotency + concurrency
// safety come from an atomic create-if-missing on the marker doc.
func (s *SeedService) Seed(ctx context.Context, userID string) (bool, error) {
	if userID == "" {
		return false, ErrUnauthenticated
	}
	_, err := s.fs.Collection(seedMarkerCollection).Doc(userID).
		Create(ctx, map[string]any{"seededAt": time.Now().UTC()})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return false, nil
		}
		return false, fmt.Errorf("seed marker: %w", err)
	}
	if err := s.plant(ctx, userID); err != nil {
		return false, err
	}
	return true, nil
}

func (s *SeedService) plant(ctx context.Context, userID string) error {
	now := time.Now().UTC()

	// Contacts: one per character (the mailroom directory).
	for _, c := range seed.Characters {
		if _, err := s.contacts.Create(ctx, userID, ContactInput{
			Name:        c.Name,
			Email:       c.Email,
			Company:     "Synthwave OS",
			Notes:       "Auto-added from your mailroom.",
			CharacterID: c.ID,
		}); err != nil {
			return fmt.Errorf("seed contact %s: %w", c.ID, err)
		}
	}

	// Inbox: each seed email becomes an inbound message, back-dated so the
	// inbox looks lived-in.
	for _, e := range seed.SeedEmails {
		ch, ok := seed.CharacterByID(e.CharacterID)
		if !ok {
			continue
		}
		created := now.Add(-time.Duration(e.AgeHours * float64(time.Hour)))
		atts := make([]Attachment, 0, len(e.Attachments))
		for _, a := range e.Attachments {
			atts = append(atts, Attachment{Name: a.Name, Size: a.Size, ContentType: a.ContentType})
		}
		if _, err := s.emails.CreateInbound(ctx, userID, InboundInput{
			From:        ch.Email,
			FromName:    ch.Name,
			Cc:          e.Cc,
			Subject:     e.Subject,
			Body:        e.Body,
			Signature:   ch.Signature,
			Attachments: atts,
			CharacterID: ch.ID,
			CreatedAt:   created,
		}); err != nil {
			return fmt.Errorf("seed email: %w", err)
		}
	}

	// Events: character-hosted meetings across the coming week.
	for _, ev := range seed.SeedEvents {
		start := dayAt(now, ev.DayOffset, ev.Hour)
		dur := ev.DurationMins
		if dur <= 0 {
			dur = 60
		}
		if _, err := s.events.Create(ctx, userID, EventInput{
			Title:       ev.Title,
			Description: ev.Description,
			Location:    ev.Location,
			Start:       start,
			End:         start.Add(time.Duration(dur) * time.Minute),
			AllDay:      ev.AllDay,
			Color:       ev.Color,
			CharacterID: ev.CharacterID,
		}); err != nil {
			return fmt.Errorf("seed event: %w", err)
		}
	}

	// Notes.
	for _, n := range seed.SeedNotes {
		if _, err := s.notes.Create(ctx, userID, NoteInput{Title: n.Title, Body: n.Body, Tags: n.Tags}); err != nil {
			return fmt.Errorf("seed note: %w", err)
		}
	}
	return nil
}

// Reset deletes the user's seeded suite content (emails, events, contacts,
// notes), drops the marker, and reseeds. Todos are left alone.
func (s *SeedService) Reset(ctx context.Context, userID string) error {
	if userID == "" {
		return ErrUnauthenticated
	}
	for _, col := range []string{
		s.emails.Collection(),
		s.events.Collection(),
		s.contacts.Collection(),
		s.notes.Collection(),
	} {
		if err := deleteAllForUser(ctx, s.fs, col, userID); err != nil {
			return err
		}
	}
	if _, err := s.fs.Collection(seedMarkerCollection).Doc(userID).Delete(ctx); err != nil {
		return fmt.Errorf("reset marker: %w", err)
	}
	if _, err := s.Seed(ctx, userID); err != nil {
		return err
	}
	return nil
}

// SeededUserIDs returns up to `limit` user IDs that have been seeded — the
// fan-out target for the drip cron. limit defaults to 100.
func (s *SeedService) SeededUserIDs(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	iter := s.fs.Collection(seedMarkerCollection).Limit(limit).Documents(ctx)
	defer iter.Stop()
	out := make([]string, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("seeded users: %w", err)
		}
		out = append(out, doc.Ref.ID)
	}
	return out, nil
}

// dayAt returns base + dayOffset days, at the given hour (UTC).
func dayAt(base time.Time, dayOffset, hour int) time.Time {
	d := base.AddDate(0, 0, dayOffset)
	return time.Date(d.Year(), d.Month(), d.Day(), hour, 0, 0, 0, time.UTC)
}

// deleteAllForUser batch-deletes every doc in `collection` owned by userID,
// committing in chunks of 400 (Firestore's batch limit is 500).
func deleteAllForUser(ctx context.Context, fs *firestore.Client, collection, userID string) error {
	iter := fs.Collection(collection).Where("userId", "==", userID).Documents(ctx)
	defer iter.Stop()
	batch := fs.Batch()
	count := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("reset iter %s: %w", collection, err)
		}
		batch.Delete(doc.Ref)
		count++
		if count%400 == 0 {
			if _, err := batch.Commit(ctx); err != nil {
				return fmt.Errorf("reset commit %s: %w", collection, err)
			}
			batch = fs.Batch()
		}
	}
	if count%400 != 0 {
		if _, err := batch.Commit(ctx); err != nil {
			return fmt.Errorf("reset commit %s: %w", collection, err)
		}
	}
	return nil
}
