package risk

import (
	"context"
	"errors"
	"fmt"

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
	fs        *firestore.Client
	gamesCol  string
	statsCol  string
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
	var s State
	if err := doc.DataTo(&s); err != nil {
		return State{}, fmt.Errorf("decode risk game: %w", err)
	}
	return s, nil
}

// SaveGame writes the game state to riskGames/{userID}. The doc is overwritten
// in full — Risk turns mutate broad swaths of the state, so a Set is simpler
// and safer than an Update with field paths.
func (p *Persistence) SaveGame(ctx context.Context, userID string, s State) error {
	if _, err := p.fs.Collection(p.gamesCol).Doc(userID).Set(ctx, s); err != nil {
		return fmt.Errorf("save risk game: %w", err)
	}
	return nil
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
