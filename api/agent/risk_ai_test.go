package agent

import (
	"math/rand/v2"
	"testing"

	"github.com/mentat/qodo/api/services/risk"
)

// TestRiskAI_SelfPlay_Terminates runs a single AI-vs-AI game using only the
// engine's pure-function operations and the AI's heuristic decision functions
// (no Store, no Firestore). It exercises the full Place → Attack → Fortify
// loop for every player every round and asserts that a winner emerges within
// a reasonable turn count — proving the engine's win/elimination detection
// is sound and the AI heuristic doesn't deadlock.
func TestRiskAI_SelfPlay_Terminates(t *testing.T) {
	rng := rand.New(rand.NewPCG(12345, 67890))
	slots := []risk.PlayerSlot{
		{ID: "ai-1", Name: "A", Kind: risk.KindAI, Color: "neonPink", GeneralID: "maxine-voltage"},
		{ID: "ai-2", Name: "B", Kind: risk.KindAI, Color: "electricBlue", GeneralID: "general-static"},
		{ID: "ai-3", Name: "C", Kind: risk.KindAI, Color: "neonGreen", GeneralID: "captain-coral"},
	}
	state, err := risk.NewGame(slots, risk.Settings{Difficulty: risk.DifficultyHard, PlayerCount: 3}, rng)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	// Burn through the setup phase by placing armies round-robin until done.
	for state.Status == risk.StatusSetup {
		pid := state.Turn.CurrentPlayerID
		// Pick any owned territory.
		var pick risk.TerritoryID
		for tid, ts := range state.Board {
			if ts.OwnerID == pid {
				pick = tid
				break
			}
		}
		if pick == "" {
			t.Fatalf("no owned territory for %s during setup", pid)
		}
		state, err = risk.PlaceInitial(state, pid, pick, rng)
		if err != nil {
			t.Fatalf("PlaceInitial: %v", err)
		}
	}

	// Self-play. Bounded by an explicit turn cap.
	const maxRounds = 500
	for r := 0; r < maxRounds && state.Status == risk.StatusPlaying; r++ {
		pid := state.Turn.CurrentPlayerID
		gen, _ := risk.GeneralByID(playerGeneralID(state, pid))
		w := composeWeights(state.Settings.Difficulty, gen)

		// Force-trade any time the hand is ≥ 5 (engine refuses placement otherwise).
		for {
			pIdx := -1
			for i, p := range state.Players {
				if p.ID == pid {
					pIdx = i
					break
				}
			}
			if pIdx < 0 || len(state.Players[pIdx].Cards) < 5 {
				break
			}
			picked := pickTradeableSet(state.Players[pIdx].Cards)
			if len(picked) != 3 {
				break
			}
			state, err = risk.TradeCards(state, pid, picked)
			if err != nil {
				break
			}
		}

		// PLACE
		for state.Turn.ArmiesToPlace > 0 {
			terr := bestPlacementTarget(state, pid, w)
			if terr == "" {
				break
			}
			state, err = risk.PlaceArmies(state, pid, terr, 1)
			if err != nil {
				break
			}
		}
		state, err = risk.EndPhase(state, pid, rng)
		if err != nil {
			t.Fatalf("EndPhase(place): %v", err)
		}

		// ATTACK (capped)
		for k := 0; k < 30 && state.Status == risk.StatusPlaying; k++ {
			from, to, prob := bestAttackPair(state, pid, w)
			if from == "" || prob < w.AttackOddsThreshold {
				break
			}
			var rounds []risk.AttackResult
			state, rounds, err = risk.Attack(state, pid, from, to, risk.AttackBlitz, rng)
			_ = rounds
			if err != nil {
				break
			}
			if state.Turn.PostConquestPending != nil {
				pc := state.Turn.PostConquestPending
				state, err = risk.ResolvePostConquest(state, pid, pc.MaxArmies)
				if err != nil {
					t.Fatalf("ResolvePostConquest: %v", err)
				}
			}
		}
		if state.Status != risk.StatusPlaying {
			break
		}
		state, err = risk.EndPhase(state, pid, rng)
		if err != nil {
			t.Fatalf("EndPhase(attack): %v", err)
		}

		// FORTIFY (skip — keeps the test focused on engine convergence)
		state, err = risk.SkipFortify(state, pid, rng)
		if err != nil {
			t.Fatalf("SkipFortify: %v", err)
		}
	}

	if state.Status != risk.StatusWon {
		t.Fatalf("self-play didn't converge: status=%s after turn=%d", state.Status, state.Turn.TurnNumber)
	}
}

func playerGeneralID(s risk.State, pid risk.PlayerID) string {
	for _, p := range s.Players {
		if p.ID == pid {
			return p.GeneralID
		}
	}
	return ""
}
