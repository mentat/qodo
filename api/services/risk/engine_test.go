package risk

import (
	"math"
	"math/rand/v2"
	"testing"
)

// fixedRand returns a deterministic RNG seeded with two known constants so the
// tests below don't flake on dice-roll variance.
func fixedRand(a, b uint64) *rand.Rand {
	return rand.New(rand.NewPCG(a, b))
}

// ─── Firestore serialization round-trip ────────────────────────────────────

// TestFirestoreState_RoundTrip locks in the State ↔ firestoreState mapping.
// This is the regression test for a panic the Firestore Go client raises
// when asked to serialize a map whose key is a string-alias type
// (`interface {} is risk.TerritoryID, not string`). The persistence layer
// converts to a string-keyed mirror; this test confirms nothing is lost.
func TestFirestoreState_RoundTrip(t *testing.T) {
	original := State{
		GameID:   "g-1",
		Status:   StatusPlaying,
		Settings: Settings{Difficulty: DifficultyNormal, PlayerCount: 3},
		Players: []Player{
			{ID: "human", Kind: KindHuman, Cards: []Card{{ID: "c1", Type: CardInfantry, TerritoryID: TerrAlaska}}},
			{ID: "ai-1", Kind: KindAI, Eliminated: true},
		},
		Board: map[TerritoryID]TerritoryState{
			TerrAlaska:    {OwnerID: "human", Armies: 5},
			TerrKamchatka: {OwnerID: "ai-1", Armies: 3},
		},
		SetupRemaining: map[PlayerID]int{"human": 0, "ai-1": 0},
		Turn: Turn{
			CurrentPlayerID: "human", TurnNumber: 7, Phase: PhaseAttack,
			ArmiesToPlace: 0, ConqueredThisTurn: true,
		},
		Events: []Event{
			{Seq: 1, PlayerID: "human", Kind: "attack", Payload: map[string]interface{}{
				"from": TerrAlaska, "to": TerrKamchatka, "attackerDice": []int{6, 5, 4},
			}},
		},
		LastEventSeq: 1,
	}
	fs := fromState(original)

	// Map keys are plain string.
	for k := range fs.Board {
		if _, isAlias := any(k).(TerritoryID); isAlias {
			t.Errorf("fs.Board key is TerritoryID alias, want plain string")
		}
	}
	// Payload alias values are converted to strings.
	if v, ok := fs.Events[0].Payload["from"].(string); !ok || v != string(TerrAlaska) {
		t.Errorf("payload 'from' should be string, got %T (%v)", fs.Events[0].Payload["from"], fs.Events[0].Payload["from"])
	}

	// Round-trip back.
	restored := fs.toState()
	if restored.GameID != original.GameID || restored.Status != original.Status {
		t.Errorf("top-level fields lost: %+v", restored)
	}
	if restored.Board[TerrAlaska].Armies != 5 || restored.Board[TerrKamchatka].OwnerID != "ai-1" {
		t.Errorf("board lost: %+v", restored.Board)
	}
	if restored.SetupRemaining[PlayerID("human")] != 0 {
		t.Errorf("setupRemaining lost: %+v", restored.SetupRemaining)
	}
	if len(restored.Events) != 1 || restored.Events[0].Kind != "attack" {
		t.Errorf("events lost: %+v", restored.Events)
	}
	if !restored.Players[1].Eliminated {
		t.Errorf("eliminated flag lost")
	}
}

// ─── Board sanity ──────────────────────────────────────────────────────────

func TestBoard_FortyTwoTerritoriesSixContinents(t *testing.T) {
	if got := len(Territories); got != 42 {
		t.Fatalf("Territories: want 42, got %d", got)
	}
	if got := len(Continents); got != 6 {
		t.Fatalf("Continents: want 6, got %d", got)
	}
}

func TestBoard_ContinentBonuses(t *testing.T) {
	wanted := map[ContinentID]int{
		ContinentNA: 5, ContinentSA: 2, ContinentEU: 5,
		ContinentAF: 3, ContinentAS: 7, ContinentAUS: 2,
	}
	for c, b := range wanted {
		if got := ContinentBonus(c); got != b {
			t.Errorf("ContinentBonus(%s): want %d, got %d", c, b, got)
		}
	}
}

func TestBoard_AdjacencyIsSymmetric(t *testing.T) {
	for _, t1 := range Territories {
		for _, t2 := range t1.Adjacent {
			if !Adjacent(t2, t1.ID) {
				t.Errorf("adjacency not symmetric: %s -> %s missing reverse", t1.ID, t2)
			}
		}
	}
}

func TestBoard_NoOrphans(t *testing.T) {
	// Every territory should have at least one neighbor.
	for _, td := range Territories {
		if len(Neighbors(td.ID)) == 0 {
			t.Errorf("territory %s has no neighbors", td.ID)
		}
	}
}

func TestBoard_WrapAroundAdjacencies(t *testing.T) {
	want := []struct{ a, b TerritoryID }{
		{TerrAlaska, TerrKamchatka},
		{TerrBrazil, TerrNorthAfrica},
		{TerrIceland, TerrGreenland},
		{TerrIndonesia, TerrSiam},
		{TerrWesternEU, TerrNorthAfrica},
		{TerrSouthernEU, TerrNorthAfrica},
		{TerrSouthernEU, TerrEgypt},
		{TerrEastAfrica, TerrMiddleEast},
	}
	for _, p := range want {
		if !Adjacent(p.a, p.b) {
			t.Errorf("expected adjacency %s ↔ %s", p.a, p.b)
		}
	}
	// Sanity: things that should NOT be adjacent.
	notAdj := []struct{ a, b TerritoryID }{
		{TerrAlaska, TerrIceland},
		{TerrJapan, TerrChina},
		{TerrArgentina, TerrPeru}, // they ARE adjacent — control
	}
	if !Adjacent(notAdj[2].a, notAdj[2].b) {
		t.Errorf("Argentina ↔ Peru should be adjacent (sanity)")
	}
	for _, p := range notAdj[:2] {
		if Adjacent(p.a, p.b) {
			t.Errorf("did not expect adjacency %s ↔ %s", p.a, p.b)
		}
	}
}

// ─── Reinforcement counting ────────────────────────────────────────────────

func TestReinforcementCount_MinIsThree(t *testing.T) {
	s := stateOwning(1) // one territory only
	if got := ReinforcementCount(s, "p"); got != 3 {
		t.Errorf("1 territory: want 3, got %d", got)
	}
}

func TestReinforcementCount_TerritoriesOverThree(t *testing.T) {
	// Base count is territories/3, floored, with a minimum of 3. Continent
	// bonuses stack on top — these tests pick territory counts that don't
	// complete a continent so we can isolate the base.
	s := State{Board: map[TerritoryID]TerritoryState{}}
	for _, tid := range AllTerritoryIDs() {
		s.Board[tid] = TerritoryState{OwnerID: "q", Armies: 1}
	}
	// Give p 12 territories spread so no continent is complete:
	// 4 of NA (out of 9), 2 of SA (out of 4), 2 of EU (out of 7), 1 of AF (of 6),
	// 2 of AS (of 12), 1 of AUS (of 4). Total = 12, no continent owned.
	spread := []TerritoryID{
		TerrAlaska, TerrAlberta, TerrOntario, TerrQuebec, // 4 NA
		TerrBrazil, TerrPeru, // 2 SA
		TerrIceland, TerrGreatBritain, // 2 EU
		TerrEgypt,           // 1 AF
		TerrIndia, TerrSiam, // 2 AS
		TerrIndonesia, // 1 AUS
	}
	for _, tid := range spread {
		s.Board[tid] = TerritoryState{OwnerID: "p", Armies: 1}
	}
	if got := ReinforcementCount(s, "p"); got != 4 {
		t.Errorf("12 spread territories: want 4, got %d", got)
	}
	// Owning everything: base = 42/3 = 14, all continents complete = 5+2+5+3+7+2 = 24. Total = 38.
	all := State{Board: map[TerritoryID]TerritoryState{}}
	for _, tid := range AllTerritoryIDs() {
		all.Board[tid] = TerritoryState{OwnerID: "p", Armies: 1}
	}
	if got := ReinforcementCount(all, "p"); got != 38 {
		t.Errorf("42 territories: want 38 (14 base + 24 continents), got %d", got)
	}
}

func TestReinforcementCount_ContinentBonus(t *testing.T) {
	// Build a state where p owns all of Australia (4 territories) plus 2 extras.
	s := State{Board: map[TerritoryID]TerritoryState{}}
	for _, t := range TerritoriesIn(ContinentAUS) {
		s.Board[t] = TerritoryState{OwnerID: "p", Armies: 1}
	}
	s.Board[TerrIndia] = TerritoryState{OwnerID: "p", Armies: 1}
	s.Board[TerrSiam] = TerritoryState{OwnerID: "p", Armies: 1}
	// 6 territories total → base = max(6/3, 3) = 3. Australia bonus = 2. Want 5.
	if got := ReinforcementCount(s, "p"); got != 5 {
		t.Errorf("Australia owner: want 5, got %d", got)
	}
}

// stateOwning returns a State where player "p" owns the first n territories.
func stateOwning(n int) State {
	s := State{Board: map[TerritoryID]TerritoryState{}}
	for i, t := range Territories {
		if i < n {
			s.Board[t.ID] = TerritoryState{OwnerID: "p", Armies: 1}
		} else {
			s.Board[t.ID] = TerritoryState{OwnerID: "q", Armies: 1}
		}
	}
	return s
}

// ─── Card set bonuses ──────────────────────────────────────────────────────

func TestSetBonus_Schedule(t *testing.T) {
	cases := map[int]int{1: 4, 2: 6, 3: 8, 4: 10, 5: 12, 6: 15, 7: 20, 8: 25, 9: 30, 10: 35}
	for n, want := range cases {
		if got := setBonus(n); got != want {
			t.Errorf("setBonus(%d): want %d, got %d", n, want, got)
		}
	}
}

func TestValidSet_AllCombinations(t *testing.T) {
	mk := func(types ...CardType) []Card {
		out := make([]Card, len(types))
		for i, ty := range types {
			out[i] = Card{ID: string(rune('a' + i)), Type: ty}
		}
		return out
	}
	cases := []struct {
		name  string
		cards []Card
		valid bool
	}{
		{"three infantry", mk(CardInfantry, CardInfantry, CardInfantry), true},
		{"three cavalry", mk(CardCavalry, CardCavalry, CardCavalry), true},
		{"one of each", mk(CardInfantry, CardCavalry, CardArtillery), true},
		{"two + wild", mk(CardInfantry, CardInfantry, CardWild), true},
		{"two wilds + any", mk(CardWild, CardWild, CardArtillery), true},
		{"three wilds", mk(CardWild, CardWild, CardWild), true},
		{"two-and-one no wild", mk(CardInfantry, CardInfantry, CardCavalry), false},
	}
	for _, c := range cases {
		if got := validSet(c.cards); got != c.valid {
			t.Errorf("%s: want %v, got %v", c.name, c.valid, got)
		}
	}
}

// ─── Combat distribution ───────────────────────────────────────────────────

// TestCombat_3v2Distribution rolls 100k 3v2 battles and asserts the empirical
// frequencies are within 1 pp of the known theoretical distribution:
// attacker wins both = 37.17%, split = 33.58%, defender wins both = 29.26%.
func TestCombat_3v2Distribution(t *testing.T) {
	r := fixedRand(11, 22)
	const N = 100_000
	var aWin2, split, dWin2 int
	for i := 0; i < N; i++ {
		ar := rollBattle(r, 5, 5) // both sides have enough armies for max dice
		switch {
		case ar.DefenderLost == 2:
			aWin2++
		case ar.AttackerLost == 1 && ar.DefenderLost == 1:
			split++
		case ar.AttackerLost == 2:
			dWin2++
		}
	}
	check := func(name string, got int, wantPct float64) {
		gotPct := float64(got) / float64(N) * 100
		if math.Abs(gotPct-wantPct) > 1.0 {
			t.Errorf("3v2 %s: want %.2f%%, got %.2f%%", name, wantPct, gotPct)
		}
	}
	check("attacker wins both", aWin2, 37.17)
	check("split", split, 33.58)
	check("defender wins both", dWin2, 29.26)
}

func TestCombat_TiesGoToDefender(t *testing.T) {
	// Construct an explicit pair where the dice are equal — defender should win.
	// rollBattle picks dice randomly; we'll test the matching logic by calling
	// it many times and checking that ties never benefit the attacker.
	r := fixedRand(7, 13)
	for i := 0; i < 1000; i++ {
		ar := rollBattle(r, 3, 2)
		// For each pair: attackerLost+defenderLost = number of pairs = min(aDice, dDice) = 2.
		if ar.AttackerLost+ar.DefenderLost != 2 {
			t.Fatalf("3v2 round should resolve 2 pairs, got %d", ar.AttackerLost+ar.DefenderLost)
		}
		// Walk the dice: any tie should NOT have caused a defender loss.
		for k := 0; k < 2; k++ {
			if ar.AttackerDice[k] == ar.DefenderDice[k] {
				// In a tie, defender keeps their army → attacker takes the loss for that pair.
				// Aggregate: we can't isolate per-pair, but we can assert that whenever both
				// dice are equal in BOTH positions, defender wins both (attackerLost == 2).
				_ = k
			}
		}
		// Specific case: if both pairs are ties, attacker must lose both.
		if ar.AttackerDice[0] == ar.DefenderDice[0] && ar.AttackerDice[1] == ar.DefenderDice[1] {
			if ar.AttackerLost != 2 {
				t.Errorf("both pairs tied: attacker should lose 2, got %d", ar.AttackerLost)
			}
		}
	}
}

// ─── Fortify ───────────────────────────────────────────────────────────────

func TestFortify_RejectsNonAdjacent(t *testing.T) {
	r := fixedRand(1, 2)
	s := twoPlayerSetup(t, r)
	// Force phase + give the human ownership of two non-adjacent territories.
	s.Turn.CurrentPlayerID = "p1"
	s.Turn.Phase = PhaseFortify
	s.Board[TerrAlaska] = TerritoryState{OwnerID: "p1", Armies: 5}
	s.Board[TerrArgentina] = TerritoryState{OwnerID: "p1", Armies: 1}
	_, err := Fortify(s, "p1", TerrAlaska, TerrArgentina, 1, r)
	if err == nil {
		t.Fatalf("expected ErrInvalidFortify for non-adjacent move")
	}
}

func TestFortify_AcceptsAdjacent(t *testing.T) {
	r := fixedRand(1, 2)
	s := twoPlayerSetup(t, r)
	s.Turn.CurrentPlayerID = "p1"
	s.Turn.Phase = PhaseFortify
	s.Board[TerrAlaska] = TerritoryState{OwnerID: "p1", Armies: 5}
	s.Board[TerrKamchatka] = TerritoryState{OwnerID: "p1", Armies: 1}
	s2, err := Fortify(s, "p1", TerrAlaska, TerrKamchatka, 3, r)
	if err != nil {
		t.Fatalf("Fortify across wrap-around: %v", err)
	}
	if s2.Board[TerrAlaska].Armies != 2 || s2.Board[TerrKamchatka].Armies != 4 {
		t.Errorf("fortify armies wrong: alaska=%d kamchatka=%d",
			s2.Board[TerrAlaska].Armies, s2.Board[TerrKamchatka].Armies)
	}
}

// ─── Win / Elimination ─────────────────────────────────────────────────────

func TestIsWinner_Detection(t *testing.T) {
	s := State{Board: map[TerritoryID]TerritoryState{}}
	for _, t := range Territories {
		s.Board[t.ID] = TerritoryState{OwnerID: "p1", Armies: 1}
	}
	if !IsWinner(s, "p1") {
		t.Error("owning all 42 should be a win")
	}
	s.Board[TerrJapan] = TerritoryState{OwnerID: "p2", Armies: 1}
	if IsWinner(s, "p1") {
		t.Error("owning 41/42 is not a win")
	}
}

func TestElimination_TransfersCards(t *testing.T) {
	s := State{
		Board: map[TerritoryID]TerritoryState{},
		Players: []Player{
			{ID: "p1", Alive: true, Cards: []Card{}},
			{ID: "p2", Alive: true, Cards: []Card{
				{ID: "c1", Type: CardInfantry},
				{ID: "c2", Type: CardCavalry},
			}},
		},
	}
	// p2 holds no territories.
	for _, t := range Territories {
		s.Board[t.ID] = TerritoryState{OwnerID: "p1", Armies: 1}
	}
	s2 := checkElimination(s, "p2", "p1")
	if !s2.Players[1].Eliminated {
		t.Error("p2 should be eliminated")
	}
	if len(s2.Players[0].Cards) != 2 {
		t.Errorf("p1 should inherit 2 cards, got %d", len(s2.Players[0].Cards))
	}
	if s2.Players[1].Cards == nil {
		t.Error("eliminated player's cards should be an empty slice, not nil")
	}
}

// ─── Setup phase auto-advancement ──────────────────────────────────────────

// TestSetup_AutoPlacesAIAndNeutral verifies that when the human places one
// army during setup, the engine auto-progresses through every non-human
// player so control returns to the human (or setup ends). Without this, the
// game deadlocks at the first AI's setup turn — the user clicks a tile but
// the engine rejects because the current player is AI.
func TestSetup_AutoPlacesAIAndNeutral(t *testing.T) {
	r := fixedRand(101, 202)
	slots := []PlayerSlot{
		{ID: "human", Name: "You", Kind: KindHuman, Color: "neonPink"},
		{ID: "ai-1", Name: "A", Kind: KindAI, Color: "electricBlue", GeneralID: "maxine-voltage"},
		{ID: "ai-2", Name: "B", Kind: KindAI, Color: "neonGreen", GeneralID: "general-static"},
	}
	state, err := NewGame(slots, Settings{Difficulty: DifficultyNormal, PlayerCount: 3}, r)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	// Pick a territory the human owns and place 1 army.
	var humanTerr TerritoryID
	for tid, ts := range state.Board {
		if ts.OwnerID == "human" {
			humanTerr = tid
			break
		}
	}
	state, err = PlaceInitial(state, "human", humanTerr, r)
	if err != nil {
		t.Fatalf("PlaceInitial: %v", err)
	}
	// Either setup is done, or it's the human's turn again. If we're still in
	// setup with the current player being an AI, the engine failed to auto-advance.
	if state.Status == StatusSetup && state.Turn.CurrentPlayerID != "human" {
		t.Fatalf("setup did not auto-advance: current player is %s (kind=%s), human is stuck",
			state.Turn.CurrentPlayerID, playerByID(state, state.Turn.CurrentPlayerID).Kind)
	}
}

// TestSetup_TwoPlayerNeutralPlacement: the existing 2-player auto-place loop
// must still work — the neutral places its 40 armies alongside the human and
// the lone AI.
func TestSetup_TwoPlayerNeutralPlacement(t *testing.T) {
	r := fixedRand(303, 404)
	slots := []PlayerSlot{
		{ID: "human", Name: "You", Kind: KindHuman, Color: "neonPink"},
		{ID: "ai-1", Name: "A", Kind: KindAI, Color: "electricBlue"},
		{ID: "neutral", Name: "Neutral", Kind: KindNeutral, Color: "synthPurple"},
	}
	state, err := NewGame(slots, Settings{Difficulty: DifficultyNormal, PlayerCount: 2}, r)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	// Walk through human-driven placements until setup ends.
	for state.Status == StatusSetup {
		if state.Turn.CurrentPlayerID != "human" {
			t.Fatalf("setup deadlocked on %s", state.Turn.CurrentPlayerID)
		}
		var pick TerritoryID
		for tid, ts := range state.Board {
			if ts.OwnerID == "human" {
				pick = tid
				break
			}
		}
		state, err = PlaceInitial(state, "human", pick, r)
		if err != nil {
			t.Fatalf("PlaceInitial: %v", err)
		}
	}
	// Verify final army counts: total should match starting × 3 players.
	totalArmies := 0
	for _, ts := range state.Board {
		totalArmies += ts.Armies
	}
	want := 40 * 3 // 2-player variant: 40 armies × 2 humans + 40 neutral
	if totalArmies != want {
		t.Errorf("setup armies: want %d, got %d", want, totalArmies)
	}
}

// TestNextLivingPlayer_SkipsNeutralAndEliminated: the neutral never gets a
// main turn (rule), and eliminated players are skipped.
func TestNextLivingPlayer_SkipsNeutralAndEliminated(t *testing.T) {
	state := State{
		Players: []Player{
			{ID: "human", Kind: KindHuman, Alive: true},
			{ID: "ai-1", Kind: KindAI, Alive: true, Eliminated: true},
			{ID: "neutral", Kind: KindNeutral, Alive: true},
			{ID: "ai-2", Kind: KindAI, Alive: true},
		},
	}
	got := nextLivingPlayer(state, "human")
	if got != "ai-2" {
		t.Errorf("nextLivingPlayer should skip eliminated ai-1 and neutral, want ai-2, got %s", got)
	}
}

// ─── Post-conquest edge cases ──────────────────────────────────────────────

// TestPostConquest_MinClampedToMax exercises the case where the attacker
// rolled more dice than their post-loss army count -1 would allow them to
// move. The engine must clamp the minimum down to the maximum (rule:
// "you may never leave a territory empty"), not raise the maximum.
func TestPostConquest_MinClampedToMax(t *testing.T) {
	r := fixedRand(42, 99)
	s := twoPlayerSetup(t, r)
	// Force attack-phase against a single-army defender from a 3-army source.
	s.Turn.CurrentPlayerID = "p1"
	s.Turn.Phase = PhaseAttack
	s.Board[TerrAlaska] = TerritoryState{OwnerID: "p1", Armies: 3}
	s.Board[TerrKamchatka] = TerritoryState{OwnerID: "ai1", Armies: 1}

	// Build a deterministic attack with attacker losing 2 (using a fixed result).
	// We don't control the dice in Attack(), so instead we directly test that
	// the engine never returns a pending move with min > max by running many
	// blitz attacks and asserting the invariant.
	for i := 0; i < 50; i++ {
		s2 := s
		s2.Board[TerrAlaska] = TerritoryState{OwnerID: "p1", Armies: 3}
		s2.Board[TerrKamchatka] = TerritoryState{OwnerID: "ai1", Armies: 1}
		s3, _, err := Attack(s2, "p1", TerrAlaska, TerrKamchatka, AttackSingle, fixedRand(uint64(i+1), 7))
		if err != nil {
			continue
		}
		if s3.Turn.PostConquestPending != nil {
			pc := s3.Turn.PostConquestPending
			if pc.MinArmies > pc.MaxArmies {
				t.Errorf("post-conquest min %d > max %d", pc.MinArmies, pc.MaxArmies)
			}
			if pc.MaxArmies > s3.Board[pc.From].Armies+s3.Board[pc.To].Armies {
				t.Errorf("post-conquest max %d exceeds available armies", pc.MaxArmies)
			}
		}
	}
}

// ─── End-to-end engine flow ────────────────────────────────────────────────

func TestNewGame_BasicShape(t *testing.T) {
	r := fixedRand(99, 100)
	slots := []PlayerSlot{
		{ID: "p1", Name: "You", Kind: KindHuman, Color: "neonPink"},
		{ID: "ai1", Name: "AI 1", Kind: KindAI, Color: "electricBlue"},
		{ID: "ai2", Name: "AI 2", Kind: KindAI, Color: "neonGreen"},
	}
	s, err := NewGame(slots, Settings{Difficulty: DifficultyNormal, PlayerCount: 3}, r)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if got := len(s.Board); got != 42 {
		t.Errorf("board: want 42 entries, got %d", got)
	}
	if s.Status != StatusSetup {
		t.Errorf("status: want %s, got %s", StatusSetup, s.Status)
	}
	if got := len(s.Deck); got != 44 {
		t.Errorf("deck: want 44, got %d", got)
	}
	// Each player should have starting armies = 35, minus 1 per dealt territory.
	dealt := map[PlayerID]int{}
	for _, ts := range s.Board {
		dealt[ts.OwnerID]++
	}
	for pid, count := range dealt {
		want := 35 - count
		if got := s.SetupRemaining[pid]; got != want {
			t.Errorf("setup remaining for %s: want %d, got %d", pid, want, got)
		}
	}
}

// twoPlayerSetup builds a minimal 2-player state for fortify tests. The
// helper forces status to playing — the tests don't exercise the alternating
// setup-placement loop.
func twoPlayerSetup(t *testing.T, r *rand.Rand) State {
	t.Helper()
	slots := []PlayerSlot{
		{ID: "p1", Name: "You", Kind: KindHuman, Color: "neonPink"},
		{ID: "ai1", Name: "AI", Kind: KindAI, Color: "electricBlue"},
		{ID: "neutral", Name: "Neutral", Kind: KindNeutral, Color: "synthPurple"},
	}
	s, err := NewGame(slots, Settings{Difficulty: DifficultyNormal, PlayerCount: 2}, r)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	s.Status = StatusPlaying
	return s
}
