// Package risk implements a faithful clone of Hasbro/Parker Brothers Risk.
//
// The engine (engine.go) is a pure rules implementation with no I/O — it
// transforms a State value into the next State via deterministic operations
// (placement, attack, fortify, card trade). I/O lives in persistence.go and
// store.go; the AI agent (../../agent/risk_ai.go) sits on top of the same
// engine to drive non-human players.
package risk

import "time"

// TerritoryID is the canonical short identifier for one of the 42 board
// territories — e.g. "alaska", "great-britain", "kamchatka".
type TerritoryID string

// ContinentID identifies one of the 6 continents.
type ContinentID string

// PlayerID is "human" for the user, "ai-0..ai-5" for AI opponents, or
// "neutral" for the 2-player variant's defender-only army.
type PlayerID string

// PlayerKind distinguishes how a player is driven each turn.
type PlayerKind string

const (
	KindHuman   PlayerKind = "human"
	KindAI      PlayerKind = "ai"
	KindNeutral PlayerKind = "neutral"
)

// Difficulty selects the AI heuristic profile.
type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyNormal Difficulty = "normal"
	DifficultyHard   Difficulty = "hard"
)

// Phase is the active sub-phase of the current player's turn.
type Phase string

const (
	PhasePlace       Phase = "place"
	PhaseAttack      Phase = "attack"
	PhaseFortify     Phase = "fortify"
	PhaseAwaitingAI  Phase = "awaiting_ai"
)

// Status is the game's lifecycle state.
type Status string

const (
	StatusSetup       Status = "setup"
	StatusPlaying     Status = "playing"
	StatusWon         Status = "won"
	StatusLost        Status = "lost"
	StatusSurrendered Status = "surrendered"
)

// CardType is the infantry/cavalry/artillery/wild marking on a Risk card.
type CardType string

const (
	CardInfantry  CardType = "inf"
	CardCavalry   CardType = "cav"
	CardArtillery CardType = "art"
	CardWild      CardType = "wild"
)

// Card is a single Risk card. The two wild cards have TerritoryID = "".
type Card struct {
	ID          string      `json:"id" firestore:"id"`
	Type        CardType    `json:"type" firestore:"type"`
	TerritoryID TerritoryID `json:"territoryId" firestore:"territoryId"`
}

// Player is one game participant. Cards are server-authoritative — when the
// state is rendered for an opponent the engine zeroes the hand contents (only
// the count remains).
type Player struct {
	ID             PlayerID   `json:"id" firestore:"id"`
	Name           string     `json:"name" firestore:"name"`
	Kind           PlayerKind `json:"kind" firestore:"kind"`
	Color          string     `json:"color" firestore:"color"`
	GeneralID      string     `json:"generalId,omitempty" firestore:"generalId,omitempty"`
	Alive          bool       `json:"alive" firestore:"alive"`
	Cards          []Card     `json:"cards" firestore:"cards"`
	CardSetsTraded int        `json:"cardSetsTraded" firestore:"cardSetsTraded"`
	Eliminated     bool       `json:"eliminated" firestore:"eliminated"`
}

// TerritoryState is the per-territory occupation: which player owns it and
// how many armies they have stationed there.
type TerritoryState struct {
	OwnerID PlayerID `json:"ownerId" firestore:"ownerId"`
	Armies  int      `json:"armies" firestore:"armies"`
}

// AttackResult summarizes one die-comparison. Used inside Event payloads.
type AttackResult struct {
	From          TerritoryID `json:"from"`
	To            TerritoryID `json:"to"`
	AttackerDice  []int       `json:"attackerDice"`
	DefenderDice  []int       `json:"defenderDice"`
	AttackerLost  int         `json:"attackerLost"`
	DefenderLost  int         `json:"defenderLost"`
	Conquered     bool        `json:"conquered"`
}

// Turn is the current-player + phase + transient counters used during a turn.
type Turn struct {
	CurrentPlayerID  PlayerID      `json:"currentPlayerId" firestore:"currentPlayerId"`
	TurnNumber       int           `json:"turnNumber" firestore:"turnNumber"`
	Phase            Phase         `json:"phase" firestore:"phase"`
	ArmiesToPlace    int           `json:"armiesToPlace" firestore:"armiesToPlace"`
	ConqueredThisTurn bool         `json:"conqueredThisTurn" firestore:"conqueredThisTurn"`
	LastAttack       *AttackResult `json:"lastAttack,omitempty" firestore:"lastAttack,omitempty"`
	// PostConquestPending, if set, blocks further actions until the attacker
	// resolves how many armies to move into the freshly-conquered territory.
	// The attacker may move at least the number of dice they rolled, up to
	// (source.Armies - 1).
	PostConquestPending *PostConquest `json:"postConquestPending,omitempty" firestore:"postConquestPending,omitempty"`
}

// PostConquest describes the pending army-movement decision after a conquest.
type PostConquest struct {
	From       TerritoryID `json:"from" firestore:"from"`
	To         TerritoryID `json:"to" firestore:"to"`
	MinArmies  int         `json:"minArmies" firestore:"minArmies"`
	MaxArmies  int         `json:"maxArmies" firestore:"maxArmies"`
}

// Event is one append-only log entry. The client uses these to animate the
// AI's turn as it streams in via Firestore onSnapshot.
type Event struct {
	Seq      int                    `json:"seq" firestore:"seq"`
	TS       time.Time              `json:"ts" firestore:"ts"`
	PlayerID PlayerID               `json:"playerId" firestore:"playerId"`
	Kind     string                 `json:"kind" firestore:"kind"`
	Payload  map[string]interface{} `json:"payload" firestore:"payload"`
}

// Settings are the player-chosen game parameters.
type Settings struct {
	Difficulty  Difficulty `json:"difficulty" firestore:"difficulty"`
	PlayerCount int        `json:"playerCount" firestore:"playerCount"`
}

// State is the entire game-doc shape stored in Firestore at riskGames/{userId}.
// All mutations should go through engine.go's transition functions.
type State struct {
	GameID    string                          `json:"gameId" firestore:"gameId"`
	Status    Status                          `json:"status" firestore:"status"`
	CreatedAt time.Time                       `json:"createdAt" firestore:"createdAt"`
	StartedAt time.Time                       `json:"startedAt" firestore:"startedAt"`
	EndedAt   *time.Time                      `json:"endedAt,omitempty" firestore:"endedAt,omitempty"`
	Settings  Settings                        `json:"settings" firestore:"settings"`
	Players   []Player                        `json:"players" firestore:"players"`
	Board     map[TerritoryID]TerritoryState  `json:"board" firestore:"board"`
	Turn      Turn                            `json:"turn" firestore:"turn"`
	Events    []Event                         `json:"events" firestore:"events"`
	// Deck is the shuffled draw pile of remaining unowned Risk cards.
	Deck []Card `json:"deck" firestore:"deck"`
	// SetupPlacements remaining per player during the alternating-placement
	// phase. Empty once status moves to "playing".
	SetupRemaining map[PlayerID]int `json:"setupRemaining" firestore:"setupRemaining"`
	// LastEventSeq is the monotonic counter used to stamp new Events.
	LastEventSeq int `json:"lastEventSeq" firestore:"lastEventSeq"`
}

// Stats is the per-user lifetime aggregate stored at riskStats/{userId}.
type Stats struct {
	WinsByDifficulty       map[string]int `json:"winsByDifficulty" firestore:"winsByDifficulty"`
	LossesByDifficulty     map[string]int `json:"lossesByDifficulty" firestore:"lossesByDifficulty"`
	SurrendersByDifficulty map[string]int `json:"surrendersByDifficulty" firestore:"surrendersByDifficulty"`
	LongestGameTurns       int            `json:"longestGameTurns" firestore:"longestGameTurns"`
	CurrentWinStreak       int            `json:"currentWinStreak" firestore:"currentWinStreak"`
	LongestWinStreak       int            `json:"longestWinStreak" firestore:"longestWinStreak"`
	TotalGamesStarted      int            `json:"totalGamesStarted" firestore:"totalGamesStarted"`
}
