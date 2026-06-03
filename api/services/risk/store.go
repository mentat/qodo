package risk

import (
	"context"
	"fmt"
	"strings"
)

// Publisher is the same single-topic interface as services.Publisher, repeated
// here to avoid an import cycle. Used to enqueue an AI-turn job whenever it
// becomes an AI player's turn.
type Publisher interface {
	Publish(ctx context.Context, attrs map[string]string, data []byte) (string, error)
}

// Store is the application-layer Risk service. It owns Persistence and an
// optional Publisher; the HTTP handlers call into it. Every mutating method
// loads the current state, applies the engine transition, saves the new state,
// and (if the next turn belongs to an AI) publishes an ai-turn job.
type Store struct {
	p          *Persistence
	publisher  Publisher
	topic      string
}

// NewStore builds a Store backed by the given Persistence.
func NewStore(p *Persistence) *Store {
	return &Store{p: p}
}

// WithPublisher returns a copy that publishes AI turn jobs via pub.
func (s *Store) WithPublisher(pub Publisher) *Store {
	cp := *s
	cp.publisher = pub
	return &cp
}

// StartGame creates a fresh game for the user, deleting any existing one.
// Lifetime "totalGamesStarted" is incremented.
func (s *Store) StartGame(ctx context.Context, userID string, settings Settings) (State, error) {
	if userID == "" {
		return State{}, fmt.Errorf("user id required")
	}
	if settings.PlayerCount < 2 || settings.PlayerCount > 6 {
		return State{}, fmt.Errorf("%w: playerCount must be 2-6", ErrInvalidSetup)
	}
	switch settings.Difficulty {
	case DifficultyEasy, DifficultyNormal, DifficultyHard:
	default:
		settings.Difficulty = DifficultyNormal
	}

	slots := buildSlots(settings)
	r := NewRand()
	state, err := NewGame(slots, settings, r)
	if err != nil {
		return State{}, err
	}
	// In the 2-player variant the engine auto-places the neutral's armies as
	// soon as a human placement triggers it. To start, set status to Playing
	// only after everyone's setup armies have been placed. NewGame leaves us
	// in setup with armies still to place, so save as-is.

	if err := s.p.SaveGame(ctx, userID, state); err != nil {
		return State{}, err
	}
	_ = s.p.UpdateStats(ctx, userID, func(st *Stats) {
		st.TotalGamesStarted++
	})
	return state, nil
}

// buildSlots constructs the PlayerSlot list for a settings choice. The human
// player always goes first; AI generals are rotated through Generals so two
// successive games don't show identical opponents. In the 2-player variant a
// neutral slot is appended (the engine auto-places its armies).
func buildSlots(settings Settings) []PlayerSlot {
	human := PlayerSlot{
		ID: "human", Name: "You", Kind: KindHuman, Color: PlayerColors[0],
	}
	slots := []PlayerSlot{human}
	if settings.PlayerCount == 2 {
		// 1 AI + 1 neutral.
		gen := Generals[0]
		slots = append(slots, PlayerSlot{
			ID: "ai-1", Name: gen.Name, Kind: KindAI,
			Color: PlayerColors[1], GeneralID: gen.ID,
		})
		slots = append(slots, PlayerSlot{
			ID: "neutral", Name: "Neutral", Kind: KindNeutral,
			Color: PlayerColors[2],
		})
		return slots
	}
	// 3..6 players: human + N AIs.
	aiCount := settings.PlayerCount - 1
	generals := PickGenerals(aiCount, 0)
	for i, g := range generals {
		slots = append(slots, PlayerSlot{
			ID:        PlayerID(fmt.Sprintf("ai-%d", i+1)),
			Name:      g.Name,
			Kind:      KindAI,
			Color:     PlayerColors[(i+1)%len(PlayerColors)],
			GeneralID: g.ID,
		})
	}
	return slots
}

// Get loads the user's current game; ErrNoGame if none.
func (s *Store) Get(ctx context.Context, userID string) (State, error) {
	return s.p.LoadGame(ctx, userID)
}

// Stats loads the user's lifetime aggregates.
func (s *Store) Stats(ctx context.Context, userID string) (Stats, error) {
	return s.p.LoadStats(ctx, userID)
}

// PlaceInitialAction routes a setup-phase placement and saves the result.
func (s *Store) PlaceInitialAction(ctx context.Context, userID string, terr TerritoryID) (State, error) {
	st, err := s.p.LoadGame(ctx, userID)
	if err != nil {
		return State{}, err
	}
	r := NewRand()
	st, err = PlaceInitial(st, "human", terr, r)
	if err != nil {
		return State{}, err
	}
	if err := s.p.SaveGame(ctx, userID, st); err != nil {
		return State{}, err
	}
	s.maybePublishAITurn(ctx, userID, st)
	return st, nil
}

// PlaceAction places armies for the current player.
func (s *Store) PlaceAction(ctx context.Context, userID string, terr TerritoryID, n int) (State, error) {
	st, err := s.p.LoadGame(ctx, userID)
	if err != nil {
		return State{}, err
	}
	st, err = PlaceArmies(st, "human", terr, n)
	if err != nil {
		return State{}, err
	}
	if err := s.p.SaveGame(ctx, userID, st); err != nil {
		return State{}, err
	}
	return st, nil
}

// TradeAction trades cards.
func (s *Store) TradeAction(ctx context.Context, userID string, cardIDs []string) (State, error) {
	st, err := s.p.LoadGame(ctx, userID)
	if err != nil {
		return State{}, err
	}
	st, err = TradeCards(st, "human", cardIDs)
	if err != nil {
		return State{}, err
	}
	if err := s.p.SaveGame(ctx, userID, st); err != nil {
		return State{}, err
	}
	return st, nil
}

// AttackAction runs one or more dice rounds.
func (s *Store) AttackAction(ctx context.Context, userID string, from, to TerritoryID, mode AttackMode) (State, []AttackResult, error) {
	st, err := s.p.LoadGame(ctx, userID)
	if err != nil {
		return State{}, nil, err
	}
	r := NewRand()
	st, rounds, err := Attack(st, "human", from, to, mode, r)
	if err != nil {
		return State{}, nil, err
	}
	if err := s.p.SaveGame(ctx, userID, st); err != nil {
		return State{}, nil, err
	}
	// Update lifetime stats on win.
	if st.Status == StatusWon {
		s.recordEnd(ctx, userID, st)
	}
	return st, rounds, nil
}

// PostConquestAction finalizes the army-move after a conquest.
func (s *Store) PostConquestAction(ctx context.Context, userID string, n int) (State, error) {
	st, err := s.p.LoadGame(ctx, userID)
	if err != nil {
		return State{}, err
	}
	st, err = ResolvePostConquest(st, "human", n)
	if err != nil {
		return State{}, err
	}
	if err := s.p.SaveGame(ctx, userID, st); err != nil {
		return State{}, err
	}
	return st, nil
}

// FortifyAction performs one fortify and advances the turn.
func (s *Store) FortifyAction(ctx context.Context, userID string, from, to TerritoryID, n int) (State, error) {
	st, err := s.p.LoadGame(ctx, userID)
	if err != nil {
		return State{}, err
	}
	r := NewRand()
	st, err = Fortify(st, "human", from, to, n, r)
	if err != nil {
		return State{}, err
	}
	if err := s.p.SaveGame(ctx, userID, st); err != nil {
		return State{}, err
	}
	s.maybePublishAITurn(ctx, userID, st)
	return st, nil
}

// EndPhaseAction advances Place → Attack → Fortify.
func (s *Store) EndPhaseAction(ctx context.Context, userID string) (State, error) {
	st, err := s.p.LoadGame(ctx, userID)
	if err != nil {
		return State{}, err
	}
	r := NewRand()
	st, err = EndPhase(st, "human", r)
	if err != nil {
		return State{}, err
	}
	if err := s.p.SaveGame(ctx, userID, st); err != nil {
		return State{}, err
	}
	s.maybePublishAITurn(ctx, userID, st)
	return st, nil
}

// SkipFortifyAction ends the human turn from the fortify phase.
func (s *Store) SkipFortifyAction(ctx context.Context, userID string) (State, error) {
	st, err := s.p.LoadGame(ctx, userID)
	if err != nil {
		return State{}, err
	}
	r := NewRand()
	st, err = SkipFortify(st, "human", r)
	if err != nil {
		return State{}, err
	}
	if err := s.p.SaveGame(ctx, userID, st); err != nil {
		return State{}, err
	}
	s.maybePublishAITurn(ctx, userID, st)
	return st, nil
}

// SurrenderAction concedes the game.
func (s *Store) SurrenderAction(ctx context.Context, userID string) (State, error) {
	st, err := s.p.LoadGame(ctx, userID)
	if err != nil {
		return State{}, err
	}
	st, err = Surrender(st, "human")
	if err != nil {
		return State{}, err
	}
	if err := s.p.SaveGame(ctx, userID, st); err != nil {
		return State{}, err
	}
	s.recordEnd(ctx, userID, st)
	return st, nil
}

// recordEnd updates lifetime stats for a finished game.
func (s *Store) recordEnd(ctx context.Context, userID string, st State) {
	diff := string(st.Settings.Difficulty)
	_ = s.p.UpdateStats(ctx, userID, func(stats *Stats) {
		switch st.Status {
		case StatusWon:
			stats.WinsByDifficulty[diff]++
			stats.CurrentWinStreak++
			if stats.CurrentWinStreak > stats.LongestWinStreak {
				stats.LongestWinStreak = stats.CurrentWinStreak
			}
		case StatusSurrendered:
			stats.SurrendersByDifficulty[diff]++
			stats.CurrentWinStreak = 0
		case StatusLost:
			stats.LossesByDifficulty[diff]++
			stats.CurrentWinStreak = 0
		}
		if st.Turn.TurnNumber > stats.LongestGameTurns {
			stats.LongestGameTurns = st.Turn.TurnNumber
		}
	})
}

// maybePublishAITurn enqueues an ai-turn job if the current player is AI and
// the game is still active. The push handler (pubsub.go) picks it up and runs
// the AI worker.
func (s *Store) maybePublishAITurn(ctx context.Context, userID string, st State) {
	if s.publisher == nil {
		return
	}
	if st.Status != StatusPlaying {
		return
	}
	cur := playerByID(st, st.Turn.CurrentPlayerID)
	if cur == nil || cur.Kind != KindAI {
		return
	}
	attrs := map[string]string{
		"kind":       "ai-turn",
		"userId":     userID,
		"gameId":     st.GameID,
		"aiPlayerId": string(cur.ID),
	}
	// Embed a tiny JSON body so subscribers that prefer Data over Attributes still work.
	data := []byte(strings.Join([]string{
		`{"kind":"ai-turn"`,
		fmt.Sprintf(`"userId":%q`, userID),
		fmt.Sprintf(`"gameId":%q`, st.GameID),
		fmt.Sprintf(`"aiPlayerId":%q}`, cur.ID),
	}, ","))
	_, _ = s.publisher.Publish(ctx, attrs, data)
}

// PublishAITurn is exposed so the pubsub push handler can re-enqueue the next
// AI's turn after one AI finishes (in a multi-AI game).
func (s *Store) PublishAITurn(ctx context.Context, userID string, st State) {
	s.maybePublishAITurn(ctx, userID, st)
}

// PersistenceRef exposes the underlying Persistence (used by the AI worker to
// write its sub-step state updates).
func (s *Store) PersistenceRef() *Persistence { return s.p }
