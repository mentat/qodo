// Package agent: the Risk AI worker.
//
// risk_ai.go runs a full AI turn end-to-end. It is invoked by the
// /api/pubsub/risk-turn push handler when it's an AI player's turn. The
// worker:
//   1. Loads the current State.
//   2. Repeatedly applies engine transitions until the phase loop completes
//      (Place → Attack → Fortify → next turn) and saves to Firestore after
//      each sub-step with a small delay so the client animates the turn live
//      via onSnapshot.
//   3. Hands off to the next AI's turn by re-publishing, if needed.
//
// The AI is heuristic-only — no LLM, no minimax. Difficulty is a knob on the
// strategy weights; the per-general Bias values nudge those weights further.
package agent

import (
	"context"
	"log"
	"math"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/mentat/qodo/api/services/risk"
)

// RiskAI runs heuristic AI turns. It depends on the Risk store for state I/O.
type RiskAI struct {
	store *risk.Store
	// Sleep is the live-animation gap between sub-steps. Tests override to 0.
	Sleep func(d time.Duration)
}

// NewRiskAI constructs a RiskAI with default tunings.
func NewRiskAI(store *risk.Store) *RiskAI {
	return &RiskAI{store: store, Sleep: time.Sleep}
}

// PlayTurn runs the full turn for the AI player currently active in the
// given user's saved game. Re-loads state from Firestore at the start (so
// the human's last move is reflected). Returns the final state.
func (a *RiskAI) PlayTurn(ctx context.Context, userID string, aiPlayerID risk.PlayerID) (risk.State, error) {
	state, err := a.store.Get(ctx, userID)
	if err != nil {
		return risk.State{}, err
	}
	if state.Status != risk.StatusPlaying {
		return state, nil
	}
	if state.Turn.CurrentPlayerID != aiPlayerID {
		// State has moved on (out-of-order Pub/Sub delivery). No-op.
		return state, nil
	}
	rng := risk.NewRand()
	gen, _ := generalFromState(state, aiPlayerID)
	weights := composeWeights(state.Settings.Difficulty, gen)

	state, err = a.playPlace(ctx, userID, state, aiPlayerID, weights, rng)
	if err != nil {
		return state, err
	}
	if state.Status != risk.StatusPlaying {
		return state, nil
	}
	state, err = a.playAttack(ctx, userID, state, aiPlayerID, weights, rng)
	if err != nil {
		return state, err
	}
	if state.Status != risk.StatusPlaying {
		return state, nil
	}
	state, err = a.playFortify(ctx, userID, state, aiPlayerID, weights, rng)
	if err != nil {
		return state, err
	}
	return state, nil
}

// ────── Strategy weights ──────────────────────────────────────────────────

// weights are the resolved per-turn strategy knobs after composing difficulty
// + per-general bias.
type weights struct {
	// AttackOddsThreshold: AI attacks only when P(conquest) ≥ this value.
	AttackOddsThreshold float64
	// TradeCardSetMinBonus: AI trades cards only when the next set bonus is
	// at least this — Hard saves for the inflection; Easy trades immediately.
	TradeCardSetMinBonus int
	// BorderWeight: armies placed on border territories get this multiplier
	// in the placement scoring.
	BorderWeight float64
	// ContinentWeight: continents the AI is close to completing get this
	// multiplier in the placement + attack scoring.
	ContinentWeight float64
	// MinArmiesForInterior: at the fortify step, an interior is anything with
	// at least this many armies that has no enemy neighbor.
	MinArmiesForInterior int
}

func composeWeights(d risk.Difficulty, gen risk.General) weights {
	w := weights{
		AttackOddsThreshold:  0.55,
		TradeCardSetMinBonus: 8,
		BorderWeight:         2.0,
		ContinentWeight:      1.5,
		MinArmiesForInterior: 2,
	}
	switch d {
	case risk.DifficultyEasy:
		w.AttackOddsThreshold = 0.7
		w.TradeCardSetMinBonus = 4 // trade as soon as possible
		w.BorderWeight = 1.2
		w.ContinentWeight = 1.0
	case risk.DifficultyHard:
		w.AttackOddsThreshold = 0.45
		w.TradeCardSetMinBonus = 12
		w.BorderWeight = 3.0
		w.ContinentWeight = 2.0
		w.MinArmiesForInterior = 1
	}
	// Per-general bias nudges (small enough not to override difficulty).
	w.AttackOddsThreshold -= 0.10 * gen.Bias.Aggression
	if w.AttackOddsThreshold < 0.3 {
		w.AttackOddsThreshold = 0.3
	}
	w.ContinentWeight += 1.0 * gen.Bias.ContinentFocus
	w.BorderWeight += 1.0 * gen.Bias.BorderHardening
	if gen.Bias.RiskTolerance > 0.7 {
		w.AttackOddsThreshold -= 0.05
	}
	return w
}

// generalFromState returns the Risk persona of the given AI player, if any.
func generalFromState(s risk.State, pid risk.PlayerID) (risk.General, bool) {
	for _, p := range s.Players {
		if p.ID == pid {
			return risk.GeneralByID(p.GeneralID)
		}
	}
	return risk.General{}, false
}

// ────── Phase: Place ──────────────────────────────────────────────────────

func (a *RiskAI) playPlace(ctx context.Context, userID string, state risk.State, pid risk.PlayerID, w weights, rng *rand.Rand) (risk.State, error) {
	// Forced trade: if hand >= 5, trade. Otherwise check the trade threshold.
	pIdx := indexOfPlayer(state, pid)
	if pIdx < 0 {
		return state, nil
	}
	for len(state.Players[pIdx].Cards) >= 3 {
		nextSet := state.Players[pIdx].CardSetsTraded + 1
		bonus := setBonusValue(nextSet)
		// Forced when ≥5 cards; otherwise opportunistic if bonus meets threshold.
		mustTrade := len(state.Players[pIdx].Cards) >= 5
		if !mustTrade && bonus < w.TradeCardSetMinBonus {
			break
		}
		picked := pickTradeableSet(state.Players[pIdx].Cards)
		if len(picked) != 3 {
			break // no valid set on hand
		}
		var err error
		state, err = risk.TradeCards(state, pid, picked)
		if err != nil {
			break
		}
		state = a.commit(ctx, userID, state)
		pIdx = indexOfPlayer(state, pid)
	}

	// Place all draft armies on the best territories.
	for state.Turn.ArmiesToPlace > 0 {
		terr := bestPlacementTarget(state, pid, w)
		if terr == "" {
			break
		}
		// Place 1 at a time so the log animates a steady stream rather than a single dump.
		n := 1
		if state.Turn.ArmiesToPlace >= 5 {
			n = 2 // amortize when the AI has lots to place
		}
		var err error
		state, err = risk.PlaceArmies(state, pid, terr, n)
		if err != nil {
			break
		}
		state = a.commit(ctx, userID, state)
	}
	// Advance to attack phase.
	var err error
	state, err = risk.EndPhase(state, pid, rng)
	if err != nil {
		return state, err
	}
	return a.commit(ctx, userID, state), nil
}

// bestPlacementTarget scores every territory the AI owns and returns the
// highest-scoring one. The score rewards bordering enemies and being adjacent
// to a continent the AI is close to completing.
func bestPlacementTarget(s risk.State, pid risk.PlayerID, w weights) risk.TerritoryID {
	type scored struct {
		t     risk.TerritoryID
		score float64
	}
	candidates := []scored{}
	for tid, ts := range s.Board {
		if ts.OwnerID != pid {
			continue
		}
		// Count enemy-army pressure on this territory.
		pressure := 0
		isBorder := false
		for _, n := range risk.Neighbors(tid) {
			if s.Board[n].OwnerID != pid {
				isBorder = true
				pressure += s.Board[n].Armies
			}
		}
		if !isBorder {
			// Interior territories are never a smart placement target.
			continue
		}
		c := risk.ContinentOf(tid)
		continentProgress := continentControlFraction(s, pid, c)
		score := float64(pressure) * w.BorderWeight
		score += continentProgress * w.ContinentWeight * 10
		candidates = append(candidates, scored{tid, score})
	}
	if len(candidates) == 0 {
		// No borders → pick any owned territory.
		for tid, ts := range s.Board {
			if ts.OwnerID == pid {
				return tid
			}
		}
		return ""
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	return candidates[0].t
}

// continentControlFraction returns the fraction of continent c that pid owns.
func continentControlFraction(s risk.State, pid risk.PlayerID, c risk.ContinentID) float64 {
	owned, total := 0, 0
	for _, t := range risk.TerritoriesIn(c) {
		total++
		if s.Board[t].OwnerID == pid {
			owned++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(owned) / float64(total)
}

// ────── Phase: Attack ─────────────────────────────────────────────────────

func (a *RiskAI) playAttack(ctx context.Context, userID string, state risk.State, pid risk.PlayerID, w weights, rng *rand.Rand) (risk.State, error) {
	// Loop: pick the best attack while it's positive-EV; execute one blitz;
	// stop when no good attack remains or we run out of stack.
	const maxAttacks = 20 // hard cap to keep the turn bounded
	for i := 0; i < maxAttacks; i++ {
		from, to, prob := bestAttackPair(state, pid, w)
		if from == "" {
			break
		}
		if prob < w.AttackOddsThreshold {
			break
		}
		var err error
		var rounds []risk.AttackResult
		state, rounds, err = risk.Attack(state, pid, from, to, risk.AttackBlitz, rng)
		_ = rounds
		if err != nil {
			break
		}
		state = a.commit(ctx, userID, state)
		// If we conquered, resolve the post-conquest movement: move all but 1
		// from the source into the new territory (max-pressure strategy).
		if state.Turn.PostConquestPending != nil {
			pc := state.Turn.PostConquestPending
			move := pc.MaxArmies
			state, err = risk.ResolvePostConquest(state, pid, move)
			if err != nil {
				log.Printf("risk-ai: resolve post-conquest failed: %v", err)
				break
			}
			state = a.commit(ctx, userID, state)
		}
		// If we eliminated and inherited cards pushing us to ≥6, trade.
		pIdx := indexOfPlayer(state, pid)
		for pIdx >= 0 && len(state.Players[pIdx].Cards) >= 5 {
			picked := pickTradeableSet(state.Players[pIdx].Cards)
			if len(picked) != 3 {
				break
			}
			// Re-enter Place phase temporarily isn't allowed by the engine, so
			// we accept that we'll trade at the start of the next turn instead.
			break
		}
		if state.Status != risk.StatusPlaying {
			return state, nil
		}
	}
	// Advance to fortify phase.
	var err error
	state, err = risk.EndPhase(state, pid, rng)
	if err != nil {
		return state, err
	}
	return a.commit(ctx, userID, state), nil
}

// bestAttackPair returns the highest-scoring attack the AI can launch (from
// one of its territories, into an enemy adjacent territory). Returns
// from="" when no positive-EV attack exists.
func bestAttackPair(s risk.State, pid risk.PlayerID, w weights) (risk.TerritoryID, risk.TerritoryID, float64) {
	bestFrom, bestTo := risk.TerritoryID(""), risk.TerritoryID("")
	bestScore := -1.0
	for tid, ts := range s.Board {
		if ts.OwnerID != pid || ts.Armies < 2 {
			continue
		}
		for _, n := range risk.Neighbors(tid) {
			tgt := s.Board[n]
			if tgt.OwnerID == pid {
				continue
			}
			prob := conquestProbability(ts.Armies, tgt.Armies)
			c := risk.ContinentOf(n)
			cProg := continentControlFraction(s, pid, c)
			score := prob*10 + cProg*w.ContinentWeight*5 - float64(tgt.Armies)*0.3
			if score > bestScore {
				bestScore = score
				bestFrom = tid
				bestTo = n
			}
		}
	}
	if bestFrom == "" {
		return "", "", 0
	}
	prob := conquestProbability(s.Board[bestFrom].Armies, s.Board[bestTo].Armies)
	return bestFrom, bestTo, prob
}

// conquestProbability returns an approximate probability that an attacker with
// `att` armies eventually conquers a defender with `def` armies, assuming both
// roll max dice every round. This is the closed-form approximation used widely
// in Risk strategy: attacker needs ~1.14x defender armies for ≥50% odds.
func conquestProbability(att, def int) float64 {
	if att <= 1 {
		return 0
	}
	if def <= 0 {
		return 1
	}
	// Use a tabulated approximation. Per Hendel/Hoffman/Manack/Wagaman
	// (Williams College), in large 3v2 battles the attacker loses 1 army for
	// every 1.0 the defender loses on average — so net advantage scales with
	// (att - 1) / def. Empirical fit:
	ratio := float64(att-1) / float64(def)
	// Logistic-ish curve centered at ratio 1.0.
	p := 1.0 / (1.0 + 1.5*pow(0.55, ratio*1.6))
	// Cap.
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	return p
}

func pow(base, exp float64) float64 { return math.Pow(base, exp) }

// ────── Phase: Fortify ────────────────────────────────────────────────────

func (a *RiskAI) playFortify(ctx context.Context, userID string, state risk.State, pid risk.PlayerID, w weights, rng *rand.Rand) (risk.State, error) {
	from, to := bestFortifyPair(state, pid, w)
	if from == "" || to == "" {
		// Nothing useful to do — skip the fortify.
		st, err := risk.SkipFortify(state, pid, rng)
		if err != nil {
			return state, err
		}
		return a.commit(ctx, userID, st), nil
	}
	src := state.Board[from]
	n := src.Armies - 1
	if n < 1 {
		n = 1
	}
	state2, err := risk.Fortify(state, pid, from, to, n, rng)
	if err != nil {
		// Fortify failed (e.g. non-adjacent due to changed state). Skip.
		st, err2 := risk.SkipFortify(state, pid, rng)
		if err2 != nil {
			return state, err
		}
		return a.commit(ctx, userID, st), nil
	}
	return a.commit(ctx, userID, state2), nil
}

// bestFortifyPair finds an interior friendly territory (no enemy neighbors)
// adjacent to a friendly border territory under the most pressure. If no such
// pair exists, returns ("", "") so the AI skips fortify.
func bestFortifyPair(s risk.State, pid risk.PlayerID, w weights) (risk.TerritoryID, risk.TerritoryID) {
	// Find the most-pressed border (highest enemy army pressure adjacent).
	type pressured struct {
		t    risk.TerritoryID
		ep   int
	}
	borders := []pressured{}
	for tid, ts := range s.Board {
		if ts.OwnerID != pid {
			continue
		}
		ep := 0
		isBorder := false
		for _, n := range risk.Neighbors(tid) {
			if s.Board[n].OwnerID != pid {
				isBorder = true
				ep += s.Board[n].Armies
			}
		}
		if isBorder {
			borders = append(borders, pressured{tid, ep})
		}
	}
	sort.Slice(borders, func(i, j int) bool { return borders[i].ep > borders[j].ep })
	// For each border (highest pressure first), find an adjacent friendly
	// interior with extra armies we can shift over.
	for _, b := range borders {
		for _, n := range risk.Neighbors(b.t) {
			ts := s.Board[n]
			if ts.OwnerID != pid {
				continue
			}
			if ts.Armies <= w.MinArmiesForInterior {
				continue
			}
			// Don't drain a border to feed another border.
			if isBorderTerritory(s, pid, n) {
				continue
			}
			return n, b.t
		}
	}
	return "", ""
}

func isBorderTerritory(s risk.State, pid risk.PlayerID, tid risk.TerritoryID) bool {
	for _, n := range risk.Neighbors(tid) {
		if s.Board[n].OwnerID != pid {
			return true
		}
	}
	return false
}

// ────── Helpers ───────────────────────────────────────────────────────────

// commit saves the current state and waits a randomized 300–700ms so the
// client animation reads one sub-step at a time via onSnapshot.
func (a *RiskAI) commit(ctx context.Context, userID string, st risk.State) risk.State {
	if err := a.store.PersistenceRef().SaveGame(ctx, userID, st); err != nil {
		log.Printf("risk-ai: save failed: %v", err)
	}
	if a.Sleep != nil {
		// 300–700ms jitter; deterministic-ish from the event seq so two AIs
		// don't accidentally synchronize.
		base := 300
		extra := (st.LastEventSeq * 37) % 400
		a.Sleep(time.Duration(base+extra) * time.Millisecond)
	}
	return st
}

// pickTradeableSet returns the IDs of a 3-card set the player can trade, or
// nil. Preference: territory-matched (for the +2 bonus) first, then "one of
// each" (lowest opportunity cost), then "three of a kind".
func pickTradeableSet(hand []risk.Card) []string {
	if len(hand) < 3 {
		return nil
	}
	// Try "one of each".
	byType := map[risk.CardType][]risk.Card{}
	var wilds []risk.Card
	for _, c := range hand {
		if c.Type == risk.CardWild {
			wilds = append(wilds, c)
		} else {
			byType[c.Type] = append(byType[c.Type], c)
		}
	}
	// One of each (no wilds needed)
	if len(byType[risk.CardInfantry]) >= 1 && len(byType[risk.CardCavalry]) >= 1 && len(byType[risk.CardArtillery]) >= 1 {
		return []string{
			byType[risk.CardInfantry][0].ID,
			byType[risk.CardCavalry][0].ID,
			byType[risk.CardArtillery][0].ID,
		}
	}
	// Three of a kind
	for _, t := range []risk.CardType{risk.CardInfantry, risk.CardCavalry, risk.CardArtillery} {
		if len(byType[t]) >= 3 {
			return []string{byType[t][0].ID, byType[t][1].ID, byType[t][2].ID}
		}
	}
	// Two of a kind + wild
	if len(wilds) >= 1 {
		for _, t := range []risk.CardType{risk.CardInfantry, risk.CardCavalry, risk.CardArtillery} {
			if len(byType[t]) >= 2 {
				return []string{byType[t][0].ID, byType[t][1].ID, wilds[0].ID}
			}
		}
		// Two distinct + wild (wild = third type)
		nonWild := []risk.Card{}
		for _, t := range []risk.CardType{risk.CardInfantry, risk.CardCavalry, risk.CardArtillery} {
			if len(byType[t]) >= 1 {
				nonWild = append(nonWild, byType[t][0])
			}
		}
		if len(nonWild) >= 2 {
			return []string{nonWild[0].ID, nonWild[1].ID, wilds[0].ID}
		}
		// Two wilds + anything
		if len(wilds) >= 2 {
			for _, c := range hand {
				if c.Type != risk.CardWild {
					return []string{wilds[0].ID, wilds[1].ID, c.ID}
				}
			}
		}
	}
	return nil
}

// setBonusValue is the schedule used to gate when the AI trades — same numbers
// as the engine's internal setBonus. Duplicated here so the AI heuristic can
// peek at "the next set" without exporting the engine helper.
func setBonusValue(n int) int {
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

func indexOfPlayer(s risk.State, id risk.PlayerID) int {
	for i, p := range s.Players {
		if p.ID == id {
			return i
		}
	}
	return -1
}
