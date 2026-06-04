package risk

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrNoGame indicates the user has no active Risk game yet.
var ErrNoGame = errors.New("no active risk game")

// Persistence is the Firestore-backed read/write layer for a user's game and
// lifetime stats. There is one active game per user (collection riskGames),
// and one stats doc per user (collection riskStats).
type Persistence struct {
	fs       *firestore.Client
	gamesCol string
	statsCol string
}

// NewPersistence builds a Persistence using the default collection names.
func NewPersistence(fs *firestore.Client) *Persistence {
	return &Persistence{fs: fs, gamesCol: "riskGames", statsCol: "riskStats"}
}

// WithCollections returns a copy pointed at the given collection names.
// Used in tests to isolate against a per-test namespace.
func (p *Persistence) WithCollections(games, stats string) *Persistence {
	cp := *p
	cp.gamesCol = games
	cp.statsCol = stats
	return &cp
}

// LoadGame reads the user's active game; returns ErrNoGame if none exists.
func (p *Persistence) LoadGame(ctx context.Context, userID string) (State, error) {
	doc, err := p.fs.Collection(p.gamesCol).Doc(userID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return State{}, ErrNoGame
		}
		return State{}, fmt.Errorf("load risk game: %w", err)
	}
	var fs firestoreState
	if err := doc.DataTo(&fs); err != nil {
		return State{}, fmt.Errorf("decode risk game: %w", err)
	}
	return fs.toState(), nil
}

// SaveGame writes the game state to riskGames/{userID}. The doc is overwritten
// in full — Risk turns mutate broad swaths of the state, so a Set is simpler
// and safer than an Update with field paths.
//
// Important: the Firestore Go client cannot serialize maps whose keys are
// string-aliased types like TerritoryID / PlayerID — it does a literal
// `.(string)` type assertion that panics with "is risk.TerritoryID, not
// string". We sidestep it by converting to a firestoreState mirror whose
// maps have plain string keys before persisting.
func (p *Persistence) SaveGame(ctx context.Context, userID string, s State) error {
	if _, err := p.fs.Collection(p.gamesCol).Doc(userID).Set(ctx, fromState(s)); err != nil {
		return fmt.Errorf("save risk game: %w", err)
	}
	return nil
}

// firestoreState is a serialization-only mirror of State. It exists solely
// because Firestore's Go client rejects string-alias map keys.
type firestoreState struct {
	GameID         string                    `firestore:"gameId"`
	Status         Status                    `firestore:"status"`
	CreatedAt      interface{}               `firestore:"createdAt"`
	StartedAt      interface{}               `firestore:"startedAt"`
	EndedAt        interface{}               `firestore:"endedAt,omitempty"`
	Settings       Settings                  `firestore:"settings"`
	Players        []Player                  `firestore:"players"`
	Board          map[string]TerritoryState `firestore:"board"`
	Turn           Turn                      `firestore:"turn"`
	Events         []firestoreEvent          `firestore:"events"`
	Deck           []Card                    `firestore:"deck"`
	SetupRemaining map[string]int            `firestore:"setupRemaining"`
	LastEventSeq   int                       `firestore:"lastEventSeq"`
}

// firestoreEvent mirrors Event but with the payload's TerritoryID/PlayerID
// alias values converted to plain strings (firestore's map-value path is fine
// with alias-typed values, but doc snapshots round-trip cleaner with strings).
type firestoreEvent struct {
	Seq      int                    `firestore:"seq"`
	TS       interface{}            `firestore:"ts"`
	PlayerID string                 `firestore:"playerId"`
	Kind     string                 `firestore:"kind"`
	Payload  map[string]interface{} `firestore:"payload"`
}

func fromState(s State) firestoreState {
	out := firestoreState{
		GameID:         s.GameID,
		Status:         s.Status,
		CreatedAt:      s.CreatedAt,
		StartedAt:      s.StartedAt,
		Settings:       s.Settings,
		Players:        normalizePlayersForFirestore(s.Players),
		Turn:           s.Turn,
		Deck:           s.Deck,
		LastEventSeq:   s.LastEventSeq,
		Board:          make(map[string]TerritoryState, len(s.Board)),
		SetupRemaining: make(map[string]int, len(s.SetupRemaining)),
		Events:         make([]firestoreEvent, len(s.Events)),
	}
	if s.EndedAt != nil {
		out.EndedAt = *s.EndedAt
	}
	for k, v := range s.Board {
		out.Board[string(k)] = v
	}
	for k, v := range s.SetupRemaining {
		out.SetupRemaining[string(k)] = v
	}
	for i, e := range s.Events {
		out.Events[i] = firestoreEvent{
			Seq:      e.Seq,
			TS:       e.TS,
			PlayerID: string(e.PlayerID),
			Kind:     e.Kind,
			Payload:  sanitizePayload(e.Payload),
		}
	}
	return out
}

func normalizePlayersForFirestore(players []Player) []Player {
	out := make([]Player, len(players))
	copy(out, players)
	for i := range out {
		if out[i].Cards == nil {
			out[i].Cards = []Card{}
		}
	}
	return out
}

// sanitizePayload returns a new map where any TerritoryID/PlayerID alias
// values are converted to plain strings. This isn't strictly required for the
// save path (firestore can store alias-typed values), but it keeps the
// round-trip stable: Firestore reads back map[string]interface{} with values
// that are plain strings, not the original alias types.
func sanitizePayload(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		switch tv := v.(type) {
		case TerritoryID:
			out[k] = string(tv)
		case PlayerID:
			out[k] = string(tv)
		case []TerritoryID:
			strs := make([]string, len(tv))
			for i, x := range tv {
				strs[i] = string(x)
			}
			out[k] = strs
		default:
			out[k] = v
		}
	}
	return out
}

func (fs firestoreState) toState() State {
	out := State{
		GameID:         fs.GameID,
		Status:         fs.Status,
		Settings:       fs.Settings,
		Players:        fs.Players,
		Turn:           fs.Turn,
		Deck:           fs.Deck,
		LastEventSeq:   fs.LastEventSeq,
		Board:          make(map[TerritoryID]TerritoryState, len(fs.Board)),
		SetupRemaining: make(map[PlayerID]int, len(fs.SetupRemaining)),
		Events:         make([]Event, len(fs.Events)),
	}
	out.CreatedAt = asTime(fs.CreatedAt)
	out.StartedAt = asTime(fs.StartedAt)
	if fs.EndedAt != nil {
		t := asTime(fs.EndedAt)
		if !t.IsZero() {
			out.EndedAt = &t
		}
	}
	for k, v := range fs.Board {
		out.Board[TerritoryID(k)] = v
	}
	for k, v := range fs.SetupRemaining {
		out.SetupRemaining[PlayerID(k)] = v
	}
	for i, e := range fs.Events {
		out.Events[i] = Event{
			Seq:      e.Seq,
			TS:       asTime(e.TS),
			PlayerID: PlayerID(e.PlayerID),
			Kind:     e.Kind,
			Payload:  e.Payload,
		}
	}
	return out
}

// asTime coerces a Firestore-decoded interface{} into a time.Time. The
// Firestore Go client decodes Timestamp fields directly into time.Time when
// the destination type is time.Time, but when decoding into interface{} it
// hands back time.Time too. We accept both for robustness.
func asTime(v interface{}) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case *time.Time:
		if t != nil {
			return *t
		}
	}
	return time.Time{}
}

// DeleteGame removes the user's active game (e.g. on Restart / Surrender ack).
func (p *Persistence) DeleteGame(ctx context.Context, userID string) error {
	if _, err := p.fs.Collection(p.gamesCol).Doc(userID).Delete(ctx); err != nil {
		return fmt.Errorf("delete risk game: %w", err)
	}
	return nil
}

// LoadStats reads the user's lifetime aggregates, returning a zero-valued
// Stats if none exists yet.
func (p *Persistence) LoadStats(ctx context.Context, userID string) (Stats, error) {
	doc, err := p.fs.Collection(p.statsCol).Doc(userID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return emptyStats(), nil
		}
		return Stats{}, fmt.Errorf("load stats: %w", err)
	}
	var s Stats
	if err := doc.DataTo(&s); err != nil {
		return Stats{}, fmt.Errorf("decode stats: %w", err)
	}
	if s.WinsByDifficulty == nil {
		s.WinsByDifficulty = map[string]int{}
	}
	if s.LossesByDifficulty == nil {
		s.LossesByDifficulty = map[string]int{}
	}
	if s.SurrendersByDifficulty == nil {
		s.SurrendersByDifficulty = map[string]int{}
	}
	return s, nil
}

// UpdateStats reads, modifies, and writes the user's stats in a single
// transaction so concurrent saves (game-end + new-game-start) don't race.
func (p *Persistence) UpdateStats(ctx context.Context, userID string, mut func(*Stats)) error {
	ref := p.fs.Collection(p.statsCol).Doc(userID)
	return p.fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(ref)
		var s Stats
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return err
			}
			s = emptyStats()
		} else {
			if err := doc.DataTo(&s); err != nil {
				return err
			}
			if s.WinsByDifficulty == nil {
				s.WinsByDifficulty = map[string]int{}
			}
			if s.LossesByDifficulty == nil {
				s.LossesByDifficulty = map[string]int{}
			}
			if s.SurrendersByDifficulty == nil {
				s.SurrendersByDifficulty = map[string]int{}
			}
		}
		mut(&s)
		return tx.Set(ref, s)
	})
}

func emptyStats() Stats {
	return Stats{
		WinsByDifficulty:       map[string]int{},
		LossesByDifficulty:     map[string]int{},
		SurrendersByDifficulty: map[string]int{},
	}
}
