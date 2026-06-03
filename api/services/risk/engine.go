package risk

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Engine errors — wrapped with %w so handlers can switch on them.
var (
	ErrNotYourTurn        = errors.New("not your turn")
	ErrWrongPhase         = errors.New("wrong phase")
	ErrInvalidPlacement   = errors.New("invalid army placement")
	ErrInvalidAttack      = errors.New("invalid attack")
	ErrInvalidFortify     = errors.New("invalid fortify")
	ErrInvalidCardSet     = errors.New("invalid card set")
	ErrPostConquestPending = errors.New("must resolve post-conquest army movement first")
	ErrGameOver           = errors.New("game is over")
	ErrInvalidSetup       = errors.New("invalid game settings")
)

// startingArmies returns the per-player starting army count per the classic
// 1993+ rulebook.
func startingArmies(players int) int {
	switch players {
	case 2:
		return 40
	case 3:
		return 35
	case 4:
		return 30
	case 5:
		return 25
	case 6:
		return 20
	}
	return 0
}

// PlayerSlot is one of the player descriptions passed into NewGame.
type PlayerSlot struct {
	ID        PlayerID
	Name      string
	Kind      PlayerKind
	Color     string
	GeneralID string
}

// NewGame builds a fresh State for the given player roster. The caller is
// responsible for assembling slots — human first, then AIs, then (for 2-player
// only) the neutral army. Territories are auto-dealt round-robin from a
// shuffled deck; remaining armies are placed one at a time during the setup
// phase. Cards are shuffled into a 44-card deck (42 territories + 2 wilds).
func NewGame(slots []PlayerSlot, settings Settings, r *rand.Rand) (State, error) {
	if r == nil {
		r = newDefaultRand()
	}
	if settings.PlayerCount < 2 || settings.PlayerCount > 6 {
		return State{}, fmt.Errorf("%w: player count must be 2-6", ErrInvalidSetup)
	}
	if settings.PlayerCount != len(slots) {
		// 2-player adds a neutral slot, so len(slots) = 3 when playerCount = 2.
		if !(settings.PlayerCount == 2 && len(slots) == 3) {
			return State{}, fmt.Errorf("%w: %d slots for %d-player game", ErrInvalidSetup, len(slots), settings.PlayerCount)
		}
	}

	now := time.Now().UTC()
	state := State{
		GameID:    uuid.NewString(),
		Status:    StatusSetup,
		CreatedAt: now,
		StartedAt: now,
		Settings:  settings,
		Players:   make([]Player, 0, len(slots)),
		Board:     map[TerritoryID]TerritoryState{},
		Turn: Turn{
			TurnNumber: 1,
			Phase:      PhasePlace,
		},
		Events:         []Event{},
		SetupRemaining: map[PlayerID]int{},
	}
	// Build the players list.
	armies := startingArmies(settings.PlayerCount)
	for _, s := range slots {
		p := Player{
			ID:        s.ID,
			Name:      s.Name,
			Kind:      s.Kind,
			Color:     s.Color,
			GeneralID: s.GeneralID,
			Alive:     true,
			Cards:     []Card{},
		}
		state.Players = append(state.Players, p)
		state.SetupRemaining[s.ID] = armies
	}

	// Shuffle the 42 territories and deal round-robin. With 6 players each
	// gets exactly 7; with 5 players each gets either 8 or 9 (the standard
	// uneven deal — players holding fewer territories get one extra army on
	// each at the start, per the rulebook). With 2 players, the neutral
	// player gets equal territories — 14 each, dealt 14/14/14.
	terrs := AllTerritoryIDs()
	r.Shuffle(len(terrs), func(i, j int) { terrs[i], terrs[j] = terrs[j], terrs[i] })
	for i, tid := range terrs {
		owner := state.Players[i%len(state.Players)].ID
		state.Board[tid] = TerritoryState{OwnerID: owner, Armies: 1}
		state.SetupRemaining[owner]-- // 1 army per dealt territory
	}

	// Build the card deck: 42 territory cards (rotated infantry/cavalry/artillery)
	// + 2 wilds. The exact distribution varies slightly between editions, but
	// modern editions assign each territory a fixed symbol cycled in order.
	state.Deck = buildDeck(r)

	// Set turn order — first player in slots starts. The setup-phase placement
	// alternates through the slots until all SetupRemaining hit 0.
	state.Turn.CurrentPlayerID = state.Players[0].ID
	state.Turn.Phase = PhasePlace
	state.Turn.ArmiesToPlace = 1 // place one army at a time during setup

	return state, nil
}

// buildDeck constructs the 44-card Risk deck and shuffles it.
func buildDeck(r *rand.Rand) []Card {
	types := []CardType{CardInfantry, CardCavalry, CardArtillery}
	deck := make([]Card, 0, len(Territories)+2)
	for i, t := range Territories {
		deck = append(deck, Card{
			ID:          uuid.NewString(),
			Type:        types[i%len(types)],
			TerritoryID: t.ID,
		})
	}
	deck = append(deck, Card{ID: uuid.NewString(), Type: CardWild})
	deck = append(deck, Card{ID: uuid.NewString(), Type: CardWild})
	r.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return deck
}

// IsSetupComplete reports whether every player has placed their starting armies.
func IsSetupComplete(s State) bool {
	for _, remaining := range s.SetupRemaining {
		if remaining > 0 {
			return false
		}
	}
	return true
}

// PlaceInitial places exactly one army during the alternating setup phase. The
// territory must already be owned by the player (territories are auto-dealt
// at the start; players reinforce, not capture, during setup). The neutral
// player is auto-placed by the engine, not by user action.
func PlaceInitial(s State, pid PlayerID, terr TerritoryID, rng *rand.Rand) (State, error) {
	if s.Status != StatusSetup {
		return s, fmt.Errorf("%w: not in setup", ErrWrongPhase)
	}
	if s.Turn.CurrentPlayerID != pid {
		return s, ErrNotYourTurn
	}
	if s.SetupRemaining[pid] <= 0 {
		return s, fmt.Errorf("%w: no armies to place", ErrInvalidPlacement)
	}
	ts, ok := s.Board[terr]
	if !ok {
		return s, fmt.Errorf("%w: unknown territory %s", ErrInvalidPlacement, terr)
	}
	if ts.OwnerID != pid {
		return s, fmt.Errorf("%w: territory %s is not yours", ErrInvalidPlacement, terr)
	}
	ts.Armies++
	s.Board[terr] = ts
	s.SetupRemaining[pid]--
	s = appendEvent(s, pid, "place", map[string]interface{}{
		"territory": terr, "count": 1, "phase": "setup",
	})

	// Advance to the next player who still has armies to place. If everyone is
	// done, flip to the main game.
	s = advanceSetupTurn(s)
	if IsSetupComplete(s) {
		s.Status = StatusPlaying
		s = beginTurn(s, s.Players[0].ID, rng)
	}
	// Auto-place for the neutral player if it's their turn (2-player variant).
	for s.Status == StatusSetup {
		cur := playerByID(s, s.Turn.CurrentPlayerID)
		if cur == nil || cur.Kind != KindNeutral {
			break
		}
		if s.SetupRemaining[cur.ID] <= 0 {
			s = advanceSetupTurn(s)
			continue
		}
		// Pick a random neutral-owned territory.
		neutralTerrs := []TerritoryID{}
		for t, ts := range s.Board {
			if ts.OwnerID == cur.ID {
				neutralTerrs = append(neutralTerrs, t)
			}
		}
		sort.Slice(neutralTerrs, func(i, j int) bool { return neutralTerrs[i] < neutralTerrs[j] })
		pick := neutralTerrs[rng.IntN(len(neutralTerrs))]
		ts := s.Board[pick]
		ts.Armies++
		s.Board[pick] = ts
		s.SetupRemaining[cur.ID]--
		s = appendEvent(s, cur.ID, "place", map[string]interface{}{
			"territory": pick, "count": 1, "phase": "setup",
		})
		s = advanceSetupTurn(s)
		if IsSetupComplete(s) {
			s.Status = StatusPlaying
			s = beginTurn(s, s.Players[0].ID, rng)
		}
	}
	return s, nil
}

// advanceSetupTurn rotates to the next player with armies still to place.
func advanceSetupTurn(s State) State {
	if IsSetupComplete(s) {
		return s
	}
	n := len(s.Players)
	idx := indexOfPlayer(s, s.Turn.CurrentPlayerID)
	for i := 1; i <= n; i++ {
		j := (idx + i) % n
		cand := s.Players[j]
		if s.SetupRemaining[cand.ID] > 0 {
			s.Turn.CurrentPlayerID = cand.ID
			return s
		}
	}
	return s
}

// beginTurn opens a new turn for player pid: compute reinforcement, set the
// phase to Place, and emit a "turn_start" event. Forced card trade-in if the
// hand is ≥5 is enforced (engine sets the phase to Place; the must-trade
// constraint is enforced at PlaceArmies / TradeCards entry).
func beginTurn(s State, pid PlayerID, _ *rand.Rand) State {
	s.Turn.CurrentPlayerID = pid
	s.Turn.Phase = PhasePlace
	s.Turn.ConqueredThisTurn = false
	s.Turn.LastAttack = nil
	s.Turn.PostConquestPending = nil
	s.Turn.ArmiesToPlace = ReinforcementCount(s, pid)
	s = appendEvent(s, pid, "turn_start", map[string]interface{}{
		"turnNumber":    s.Turn.TurnNumber,
		"armiesToPlace": s.Turn.ArmiesToPlace,
	})
	return s
}

// ReinforcementCount computes how many new armies a player draws at the start
// of their turn: max(territories/3, 3), plus continent bonuses for any
// continent the player fully owns. Card-set trades add on top via TradeCards.
func ReinforcementCount(s State, pid PlayerID) int {
	owned := 0
	for _, ts := range s.Board {
		if ts.OwnerID == pid {
			owned++
		}
	}
	base := owned / 3
	if base < 3 {
		base = 3
	}
	bonus := 0
	for _, c := range Continents {
		if ownsContinent(s, pid, c.ID) {
			bonus += c.Bonus
		}
	}
	return base + bonus
}

// ownsContinent reports whether pid owns every territory in continent c.
func ownsContinent(s State, pid PlayerID, c ContinentID) bool {
	for _, t := range TerritoriesIn(c) {
		if s.Board[t].OwnerID != pid {
			return false
		}
	}
	return true
}

// PlaceArmies places n armies on a territory owned by pid. Must be Place
// phase; n must be ≤ ArmiesToPlace.
func PlaceArmies(s State, pid PlayerID, terr TerritoryID, n int) (State, error) {
	if s.Status != StatusPlaying {
		return s, ErrGameOver
	}
	if s.Turn.CurrentPlayerID != pid {
		return s, ErrNotYourTurn
	}
	if s.Turn.Phase != PhasePlace {
		return s, fmt.Errorf("%w: place", ErrWrongPhase)
	}
	if n <= 0 || n > s.Turn.ArmiesToPlace {
		return s, fmt.Errorf("%w: count %d out of bounds (have %d)", ErrInvalidPlacement, n, s.Turn.ArmiesToPlace)
	}
	if mustTrade(s, pid) {
		return s, fmt.Errorf("%w: must trade cards (hand ≥ 5)", ErrInvalidPlacement)
	}
	ts, ok := s.Board[terr]
	if !ok {
		return s, fmt.Errorf("%w: unknown territory %s", ErrInvalidPlacement, terr)
	}
	if ts.OwnerID != pid {
		return s, fmt.Errorf("%w: territory %s is not yours", ErrInvalidPlacement, terr)
	}
	ts.Armies += n
	s.Board[terr] = ts
	s.Turn.ArmiesToPlace -= n
	s = appendEvent(s, pid, "place", map[string]interface{}{
		"territory": terr, "count": n,
	})
	return s, nil
}

// mustTrade reports whether pid must trade cards before placing — the hand
// has 5+ cards at the start of their turn.
func mustTrade(s State, pid PlayerID) bool {
	p := playerByID(s, pid)
	return p != nil && len(p.Cards) >= 5
}

// TradeCards validates and trades a set of 3 cards for armies. The bonus
// escalates: 4, 6, 8, 10, 12, 15, then +5 per set thereafter. If any of the
// traded cards shows a territory the player owns, an additional 2 armies are
// placed directly on the first such territory.
func TradeCards(s State, pid PlayerID, cardIDs []string) (State, error) {
	if s.Status != StatusPlaying {
		return s, ErrGameOver
	}
	if s.Turn.CurrentPlayerID != pid {
		return s, ErrNotYourTurn
	}
	if s.Turn.Phase != PhasePlace {
		return s, fmt.Errorf("%w: place", ErrWrongPhase)
	}
	if len(cardIDs) != 3 {
		return s, fmt.Errorf("%w: must trade exactly 3 cards", ErrInvalidCardSet)
	}
	pIdx := indexOfPlayer(s, pid)
	if pIdx < 0 {
		return s, ErrNotYourTurn
	}
	hand := s.Players[pIdx].Cards
	picked, remaining, err := pickCards(hand, cardIDs)
	if err != nil {
		return s, err
	}
	if !validSet(picked) {
		return s, fmt.Errorf("%w: cards do not form a valid set", ErrInvalidCardSet)
	}

	// Compute bonus per the escalating schedule.
	setNumber := s.Players[pIdx].CardSetsTraded + 1
	bonus := setBonus(setNumber)

	// Update player state.
	s.Players[pIdx].Cards = remaining
	s.Players[pIdx].CardSetsTraded = setNumber
	s.Turn.ArmiesToPlace += bonus

	// Territory bonus: +2 armies on any territory shown on a traded card the
	// player owns. The rulebook awards this for each matching card up to 2
	// armies max per set (place them directly on that territory).
	territoryBonus := 0
	var bonusTerr TerritoryID
	for _, c := range picked {
		if c.TerritoryID == "" {
			continue
		}
		if s.Board[c.TerritoryID].OwnerID != pid {
			continue
		}
		territoryBonus = 2
		bonusTerr = c.TerritoryID
		break
	}
	if territoryBonus > 0 {
		ts := s.Board[bonusTerr]
		ts.Armies += territoryBonus
		s.Board[bonusTerr] = ts
	}

	// Cards go back to the bottom of the deck.
	s.Deck = append(s.Deck, picked...)

	s = appendEvent(s, pid, "trade_cards", map[string]interface{}{
		"setNumber":      setNumber,
		"bonus":          bonus,
		"cardIds":        cardIDs,
		"territoryBonus": territoryBonus,
		"territory":      bonusTerr,
	})
	return s, nil
}

// pickCards selects three cards by ID from the hand, returning the picked
// cards and the remaining hand. Errors if any ID isn't present.
func pickCards(hand []Card, ids []string) ([]Card, []Card, error) {
	picked := make([]Card, 0, 3)
	remaining := make([]Card, 0, len(hand))
	wanted := map[string]bool{}
	for _, id := range ids {
		wanted[id] = true
	}
	for _, c := range hand {
		if wanted[c.ID] && len(picked) < len(ids) {
			picked = append(picked, c)
			delete(wanted, c.ID)
		} else {
			remaining = append(remaining, c)
		}
	}
	if len(wanted) != 0 {
		return nil, nil, fmt.Errorf("%w: cards not in hand", ErrInvalidCardSet)
	}
	return picked, remaining, nil
}

// validSet reports whether 3 cards form a tradeable set per Risk rules:
// 3 of the same type, OR one of each type, OR any 2 cards + a wild.
func validSet(cards []Card) bool {
	if len(cards) != 3 {
		return false
	}
	wilds := 0
	count := map[CardType]int{}
	for _, c := range cards {
		if c.Type == CardWild {
			wilds++
			continue
		}
		count[c.Type]++
	}
	if wilds >= 1 {
		// At least one wild → always valid (wild can fill any third slot).
		return true
	}
	if len(count) == 1 {
		return true // three of the same type
	}
	if len(count) == 3 {
		return true // one of each
	}
	return false
}

// setBonus returns the armies awarded for the nth card set traded across the
// whole game. Sets 1..6 award 4, 6, 8, 10, 12, 15; each set thereafter is
// previous + 5.
func setBonus(n int) int {
	switch n {
	case 1:
		return 4
	case 2:
		return 6
	case 3:
		return 8
	case 4:
		return 10
	case 5:
		return 12
	case 6:
		return 15
	}
	if n <= 0 {
		return 0
	}
	return 15 + 5*(n-6)
}

// EndPhase advances Place → Attack → Fortify → next-player's Place.
// Cards are drawn at end-of-turn if the player conquered at least one
// territory.
func EndPhase(s State, pid PlayerID, rng *rand.Rand) (State, error) {
	if s.Status != StatusPlaying {
		return s, ErrGameOver
	}
	if s.Turn.CurrentPlayerID != pid {
		return s, ErrNotYourTurn
	}
	if s.Turn.PostConquestPending != nil {
		return s, ErrPostConquestPending
	}
	switch s.Turn.Phase {
	case PhasePlace:
		if s.Turn.ArmiesToPlace > 0 {
			return s, fmt.Errorf("%w: %d armies remain to place", ErrInvalidPlacement, s.Turn.ArmiesToPlace)
		}
		if mustTrade(s, pid) {
			return s, fmt.Errorf("%w: must trade cards (hand ≥ 5)", ErrInvalidPlacement)
		}
		s.Turn.Phase = PhaseAttack
		s = appendEvent(s, pid, "phase", map[string]interface{}{"to": "attack"})
	case PhaseAttack:
		s.Turn.Phase = PhaseFortify
		s = appendEvent(s, pid, "phase", map[string]interface{}{"to": "fortify"})
	case PhaseFortify:
		// Draw a card if the player conquered at least one territory this turn.
		if s.Turn.ConqueredThisTurn {
			s = drawCard(s, pid)
		}
		// Advance to next living player.
		next := nextLivingPlayer(s, pid)
		if next == "" {
			return s, ErrGameOver
		}
		// If we've cycled back to the start of the player list, bump turn number.
		if indexOfPlayer(s, next) <= indexOfPlayer(s, pid) {
			s.Turn.TurnNumber++
		}
		s = beginTurn(s, next, rng)
	default:
		return s, fmt.Errorf("%w: %s", ErrWrongPhase, s.Turn.Phase)
	}
	return s, nil
}

// drawCard pulls the top card off the deck and adds it to pid's hand. If the
// deck is empty the call is a no-op (extremely rare — only with extreme card
// hoarding).
func drawCard(s State, pid PlayerID) State {
	if len(s.Deck) == 0 {
		return s
	}
	card := s.Deck[0]
	s.Deck = s.Deck[1:]
	idx := indexOfPlayer(s, pid)
	s.Players[idx].Cards = append(s.Players[idx].Cards, card)
	return appendEvent(s, pid, "draw_card", map[string]interface{}{
		"cardType": card.Type, "territory": card.TerritoryID,
	})
}

// nextLivingPlayer returns the next player ID after pid that is still alive.
// Skips eliminated players. Returns "" if everyone but pid is gone (which
// should have triggered a win already).
func nextLivingPlayer(s State, pid PlayerID) PlayerID {
	n := len(s.Players)
	idx := indexOfPlayer(s, pid)
	for i := 1; i <= n; i++ {
		j := (idx + i) % n
		if !s.Players[j].Eliminated {
			return s.Players[j].ID
		}
	}
	return ""
}

// AttackMode is single (one dice round) or blitz (auto-resolve until one
// side stops or breaks).
type AttackMode string

const (
	AttackSingle AttackMode = "single"
	AttackBlitz  AttackMode = "blitz"
)

// Attack runs one or more combat rounds between two adjacent territories. The
// engine returns the new state plus the per-round results.
//
// Dice rules: attacker rolls min(3, attackerArmies-1) dice; defender rolls
// min(2, defenderArmies). Sort each set descending, pair highest-vs-highest;
// ties go to defender. Each pair removes one army from the loser.
//
// Blitz mode keeps rolling until either:
//   - the defender has 0 armies (conquest), or
//   - the attacker has only 1 army left (can't roll any more dice).
func Attack(s State, pid PlayerID, from, to TerritoryID, mode AttackMode, rng *rand.Rand) (State, []AttackResult, error) {
	if s.Status != StatusPlaying {
		return s, nil, ErrGameOver
	}
	if s.Turn.CurrentPlayerID != pid {
		return s, nil, ErrNotYourTurn
	}
	if s.Turn.Phase != PhaseAttack {
		return s, nil, fmt.Errorf("%w: attack", ErrWrongPhase)
	}
	if s.Turn.PostConquestPending != nil {
		return s, nil, ErrPostConquestPending
	}
	src, ok := s.Board[from]
	if !ok || src.OwnerID != pid {
		return s, nil, fmt.Errorf("%w: must attack from your territory", ErrInvalidAttack)
	}
	dst, ok := s.Board[to]
	if !ok {
		return s, nil, fmt.Errorf("%w: unknown territory %s", ErrInvalidAttack, to)
	}
	if dst.OwnerID == pid {
		return s, nil, fmt.Errorf("%w: cannot attack your own territory", ErrInvalidAttack)
	}
	if !Adjacent(from, to) {
		return s, nil, fmt.Errorf("%w: territories not adjacent", ErrInvalidAttack)
	}
	if src.Armies < 2 {
		return s, nil, fmt.Errorf("%w: need at least 2 armies to attack", ErrInvalidAttack)
	}

	rolls := []AttackResult{}
	for {
		ar := rollBattle(rng, src.Armies, dst.Armies)
		ar.From = from
		ar.To = to
		src.Armies -= ar.AttackerLost
		dst.Armies -= ar.DefenderLost
		conquered := dst.Armies <= 0
		ar.Conquered = conquered
		rolls = append(rolls, ar)
		if conquered {
			// Transfer ownership; pending post-conquest army move.
			oldOwner := dst.OwnerID
			dst.OwnerID = pid
			// Attacker must move at least the dice-count used in the final roll,
			// up to (src.Armies - 1). Defender's armies become 0 in dst until the
			// move resolves. Edge case: if the attacker lost enough armies in the
			// conquering round that (source-1) < dice-count, the rule says they
			// move all-but-one (you can never leave a territory empty), so the
			// minimum is clamped down to the maximum.
			maxMove := src.Armies - 1
			if maxMove < 1 {
				maxMove = 1
			}
			minMove := len(ar.AttackerDice)
			if minMove < 1 {
				minMove = 1
			}
			if minMove > maxMove {
				minMove = maxMove
			}
			dst.Armies = 0
			s.Board[from] = src
			s.Board[to] = dst
			s.Turn.ConqueredThisTurn = true
			s.Turn.PostConquestPending = &PostConquest{
				From: from, To: to, MinArmies: minMove, MaxArmies: maxMove,
			}
			s.Turn.LastAttack = &ar
			s = appendEvent(s, pid, "attack", attackPayload(ar))
			s = appendEvent(s, pid, "conquer", map[string]interface{}{
				"from": from, "to": to, "fromOwner": oldOwner,
			})
			// Eliminate the defender if they're out of territories.
			s = checkElimination(s, oldOwner, pid)
			// Win check.
			if isWinner(s, pid) {
				s.Status = StatusWon
				now := time.Now().UTC()
				s.EndedAt = &now
				s = appendEvent(s, pid, "win", map[string]interface{}{})
			}
			break
		}
		s.Board[from] = src
		s.Board[to] = dst
		s.Turn.LastAttack = &ar
		s = appendEvent(s, pid, "attack", attackPayload(ar))

		if mode != AttackBlitz {
			break
		}
		if src.Armies < 2 {
			break // attacker can't continue
		}
	}
	return s, rolls, nil
}

func attackPayload(ar AttackResult) map[string]interface{} {
	return map[string]interface{}{
		"from": ar.From, "to": ar.To,
		"attackerDice": ar.AttackerDice, "defenderDice": ar.DefenderDice,
		"attackerLost": ar.AttackerLost, "defenderLost": ar.DefenderLost,
		"conquered": ar.Conquered,
	}
}

// rollBattle runs one die-roll round. Dice counts: attacker = min(3, armies-1),
// defender = min(2, armies). Highest-vs-highest, ties to defender.
func rollBattle(r *rand.Rand, attackerArmies, defenderArmies int) AttackResult {
	aDice := 3
	if attackerArmies-1 < aDice {
		aDice = attackerArmies - 1
	}
	if aDice < 1 {
		aDice = 1
	}
	dDice := 2
	if defenderArmies < dDice {
		dDice = defenderArmies
	}
	if dDice < 1 {
		dDice = 1
	}
	a := rollDice(r, aDice)
	d := rollDice(r, dDice)
	res := AttackResult{AttackerDice: a, DefenderDice: d}
	pairs := dDice
	if aDice < pairs {
		pairs = aDice
	}
	for i := 0; i < pairs; i++ {
		if a[i] > d[i] {
			res.DefenderLost++
		} else {
			res.AttackerLost++
		}
	}
	return res
}

// rollDice rolls n six-sided dice and returns them sorted descending.
func rollDice(r *rand.Rand, n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = r.IntN(6) + 1
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

// ResolvePostConquest finalizes the army movement into a freshly-conquered
// territory. n must be in [MinArmies, MaxArmies].
func ResolvePostConquest(s State, pid PlayerID, n int) (State, error) {
	if s.Status != StatusPlaying && s.Status != StatusWon {
		return s, ErrGameOver
	}
	if s.Turn.CurrentPlayerID != pid {
		return s, ErrNotYourTurn
	}
	pc := s.Turn.PostConquestPending
	if pc == nil {
		return s, fmt.Errorf("%w: no pending post-conquest move", ErrInvalidAttack)
	}
	if n < pc.MinArmies || n > pc.MaxArmies {
		return s, fmt.Errorf("%w: must move between %d and %d armies", ErrInvalidAttack, pc.MinArmies, pc.MaxArmies)
	}
	src := s.Board[pc.From]
	dst := s.Board[pc.To]
	src.Armies -= n
	dst.Armies += n
	s.Board[pc.From] = src
	s.Board[pc.To] = dst
	s.Turn.PostConquestPending = nil
	s = appendEvent(s, pid, "post_conquest", map[string]interface{}{
		"from": pc.From, "to": pc.To, "moved": n,
	})
	return s, nil
}

// Fortify performs one fortification move at end-of-turn. Classic rule: source
// and destination must be directly adjacent friendly territories; move
// between 1 and source.Armies-1 armies; one move per turn (call EndPhase
// after).
func Fortify(s State, pid PlayerID, from, to TerritoryID, n int, rng *rand.Rand) (State, error) {
	if s.Status != StatusPlaying {
		return s, ErrGameOver
	}
	if s.Turn.CurrentPlayerID != pid {
		return s, ErrNotYourTurn
	}
	if s.Turn.Phase != PhaseFortify {
		return s, fmt.Errorf("%w: fortify", ErrWrongPhase)
	}
	src, ok := s.Board[from]
	if !ok || src.OwnerID != pid {
		return s, fmt.Errorf("%w: from territory not yours", ErrInvalidFortify)
	}
	dst, ok := s.Board[to]
	if !ok || dst.OwnerID != pid {
		return s, fmt.Errorf("%w: destination territory not yours", ErrInvalidFortify)
	}
	if !Adjacent(from, to) {
		return s, fmt.Errorf("%w: territories not adjacent (classic rule)", ErrInvalidFortify)
	}
	if n < 1 || n > src.Armies-1 {
		return s, fmt.Errorf("%w: must move 1 to %d armies", ErrInvalidFortify, src.Armies-1)
	}
	src.Armies -= n
	dst.Armies += n
	s.Board[from] = src
	s.Board[to] = dst
	s = appendEvent(s, pid, "fortify", map[string]interface{}{
		"from": from, "to": to, "count": n,
	})
	// One fortify per turn → immediately end the turn.
	next := nextLivingPlayer(s, pid)
	if next == "" {
		return s, nil
	}
	if s.Turn.ConqueredThisTurn {
		s = drawCard(s, pid)
	}
	if indexOfPlayer(s, next) <= indexOfPlayer(s, pid) {
		s.Turn.TurnNumber++
	}
	s = beginTurn(s, next, rng)
	return s, nil
}

// SkipFortify ends the turn without fortifying.
func SkipFortify(s State, pid PlayerID, rng *rand.Rand) (State, error) {
	if s.Status != StatusPlaying {
		return s, ErrGameOver
	}
	if s.Turn.CurrentPlayerID != pid {
		return s, ErrNotYourTurn
	}
	if s.Turn.Phase != PhaseFortify {
		return s, fmt.Errorf("%w: fortify", ErrWrongPhase)
	}
	if s.Turn.ConqueredThisTurn {
		s = drawCard(s, pid)
	}
	next := nextLivingPlayer(s, pid)
	if next == "" {
		return s, nil
	}
	if indexOfPlayer(s, next) <= indexOfPlayer(s, pid) {
		s.Turn.TurnNumber++
	}
	s = beginTurn(s, next, rng)
	return s, nil
}

// Surrender marks the human player as surrendered and ends the game.
func Surrender(s State, pid PlayerID) (State, error) {
	if s.Status != StatusPlaying && s.Status != StatusSetup {
		return s, ErrGameOver
	}
	p := playerByID(s, pid)
	if p == nil {
		return s, ErrNotYourTurn
	}
	s.Status = StatusSurrendered
	now := time.Now().UTC()
	s.EndedAt = &now
	s = appendEvent(s, pid, "surrender", map[string]interface{}{})
	return s, nil
}

// isWinner reports whether pid owns every territory on the board.
func isWinner(s State, pid PlayerID) bool {
	for _, ts := range s.Board {
		if ts.OwnerID != pid {
			return false
		}
	}
	return true
}

// IsWinner exports isWinner for tests and external callers.
func IsWinner(s State, pid PlayerID) bool { return isWinner(s, pid) }

// checkElimination marks oldOwner eliminated if they hold no territories and
// transfers their cards to the killer. If the resulting hand has ≥6 cards the
// killer must trade immediately (engine enforces by setting a flag — caller
// invokes TradeCards before any other action).
func checkElimination(s State, oldOwner, killer PlayerID) State {
	if oldOwner == killer {
		return s
	}
	for _, ts := range s.Board {
		if ts.OwnerID == oldOwner {
			return s
		}
	}
	// Eliminated.
	oldIdx := indexOfPlayer(s, oldOwner)
	killIdx := indexOfPlayer(s, killer)
	if oldIdx < 0 || killIdx < 0 {
		return s
	}
	transferred := s.Players[oldIdx].Cards
	s.Players[killIdx].Cards = append(s.Players[killIdx].Cards, transferred...)
	s.Players[oldIdx].Cards = nil
	s.Players[oldIdx].Eliminated = true
	s.Players[oldIdx].Alive = false
	s = appendEvent(s, killer, "eliminate", map[string]interface{}{
		"victim":          oldOwner,
		"cardsTransferred": len(transferred),
	})
	return s
}

// IsEliminated reports whether pid has no remaining territories.
func IsEliminated(s State, pid PlayerID) bool {
	for _, ts := range s.Board {
		if ts.OwnerID == pid {
			return false
		}
	}
	return true
}

// CurrentPlayer returns the player whose turn it is.
func CurrentPlayer(s State) *Player {
	return playerByID(s, s.Turn.CurrentPlayerID)
}

// playerByID is an internal lookup.
func playerByID(s State, id PlayerID) *Player {
	idx := indexOfPlayer(s, id)
	if idx < 0 {
		return nil
	}
	return &s.Players[idx]
}

// indexOfPlayer returns the slot index for a player ID, or -1.
func indexOfPlayer(s State, id PlayerID) int {
	for i, p := range s.Players {
		if p.ID == id {
			return i
		}
	}
	return -1
}

// appendEvent stamps a new event onto the log and bumps the seq counter.
func appendEvent(s State, pid PlayerID, kind string, payload map[string]interface{}) State {
	s.LastEventSeq++
	ev := Event{
		Seq:      s.LastEventSeq,
		TS:       time.Now().UTC(),
		PlayerID: pid,
		Kind:     kind,
		Payload:  payload,
	}
	s.Events = append(s.Events, ev)
	// Bound the in-doc events to the last 200 entries. (Older entries are
	// archived to a subcollection by the persistence layer; the engine just
	// trims the in-memory slice so the doc stays small.)
	const maxEvents = 200
	if len(s.Events) > maxEvents {
		s.Events = s.Events[len(s.Events)-maxEvents:]
	}
	return s
}

// newDefaultRand returns a rand.Rand seeded from the runtime's secure source.
// rand/v2's New(NewPCG) seeded with time + uuid bytes is good enough for game
// dice — we don't need crypto-grade randomness, just unpredictability across
// concurrent games.
func newDefaultRand() *rand.Rand {
	u := uuid.New()
	var seed1, seed2 uint64
	for i := 0; i < 8; i++ {
		seed1 = seed1<<8 | uint64(u[i])
		seed2 = seed2<<8 | uint64(u[8+i])
	}
	seed1 ^= uint64(time.Now().UnixNano())
	return rand.New(rand.NewPCG(seed1, seed2))
}

// NewRand returns a freshly-seeded RNG suitable for one game/operation.
func NewRand() *rand.Rand { return newDefaultRand() }

// PlayerView returns a deep-ish copy of s where every other player's card
// contents are hidden (the count remains). This is what the engine returns to
// the HTTP layer for serialization to the client.
func PlayerView(s State, viewer PlayerID) State {
	cp := s
	cp.Players = make([]Player, len(s.Players))
	for i, p := range s.Players {
		cp.Players[i] = p
		if p.ID == viewer {
			cards := make([]Card, len(p.Cards))
			copy(cards, p.Cards)
			cp.Players[i].Cards = cards
		} else {
			hidden := make([]Card, len(p.Cards))
			for j := range hidden {
				hidden[j] = Card{ID: fmt.Sprintf("hidden-%d-%d", i, j)}
			}
			cp.Players[i].Cards = hidden
		}
	}
	return cp
}
