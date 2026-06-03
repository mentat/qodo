package risk

// General is one of the synthwave-themed AI opponents. Difficulty + Bias
// together shape the AI's strategy weights at runtime; the persona is also
// surfaced on the frontend for character art + flavor.
type General struct {
	ID    string
	Name  string
	Title string
	Blurb string
	Color string // Mantine palette key
	Bias  Bias
}

// Bias is a per-general personality vector layered on top of the difficulty
// knob. All values are 0..1.
type Bias struct {
	Aggression      float64 // higher → attacks more often and at lower odds
	ContinentFocus  float64 // higher → harder to dislodge from a continent goal
	RiskTolerance   float64 // higher → accepts marginal EV attacks
	BorderHardening float64 // higher → stacks borders, leaves interiors thin
}

// Generals is the synthwave-themed Risk cast. Eight entries — enough variety
// for any 2-6 player game with no repeats. Each general's color is a hint;
// the engine still assigns a unique board color from the player palette.
var Generals = []General{
	{
		ID: "maxine-voltage", Name: "Maxine Voltage", Title: "Field Marshal",
		Blurb: "A neon-soaked cavalry commander who treats the world like a synth solo — loud, fast, and impossible to ignore.",
		Color: "neonPink",
		Bias:  Bias{Aggression: 0.85, ContinentFocus: 0.45, RiskTolerance: 0.7, BorderHardening: 0.4},
	},
	{
		ID: "general-static", Name: "General Static", Title: "Commander",
		Blurb: "A patient tactician whose silences are louder than her artillery. She finishes continents others abandon.",
		Color: "electricBlue",
		Bias:  Bias{Aggression: 0.35, ContinentFocus: 0.85, RiskTolerance: 0.4, BorderHardening: 0.75},
	},
	{
		ID: "vice-admiral-vector", Name: "Vice-Admiral Vector", Title: "Vice-Admiral",
		Blurb: "Naval strategist with a vendetta against any continent that touches the sea. Especially Australia.",
		Color: "neonGreen",
		Bias:  Bias{Aggression: 0.55, ContinentFocus: 0.75, RiskTolerance: 0.55, BorderHardening: 0.5},
	},
	{
		ID: "commodore-cassette", Name: "Commodore Cassette", Title: "Commodore",
		Blurb: "Rewinds, replays, and refuses to flip sides. Predictable but relentless — once she picks a target, she will not stop.",
		Color: "hotYellow",
		Bias:  Bias{Aggression: 0.5, ContinentFocus: 0.65, RiskTolerance: 0.5, BorderHardening: 0.6},
	},
	{
		ID: "captain-coral", Name: "Captain Coral", Title: "Captain",
		Blurb: "Coastal raider who hops across oceans. Reads the wrap-around adjacencies like nobody else.",
		Color: "synthPurple",
		Bias:  Bias{Aggression: 0.75, ContinentFocus: 0.3, RiskTolerance: 0.8, BorderHardening: 0.3},
	},
	{
		ID: "field-marshal-neon", Name: "Field Marshal Neon", Title: "Field Marshal",
		Blurb: "Old-school continental conquest doctrine in a chrome-pink uniform. Wins continents one by one and never gives them back.",
		Color: "neonPink",
		Bias:  Bias{Aggression: 0.5, ContinentFocus: 0.9, RiskTolerance: 0.45, BorderHardening: 0.85},
	},
	{
		ID: "colonel-chrome", Name: "Colonel Chrome", Title: "Colonel",
		Blurb: "Polished to a mirror. Plays a textbook board with one twist: he opens with Asia and dares you to stop him.",
		Color: "electricBlue",
		Bias:  Bias{Aggression: 0.6, ContinentFocus: 0.7, RiskTolerance: 0.6, BorderHardening: 0.55},
	},
	{
		ID: "lieutenant-laser", Name: "Lt. Laser", Title: "Lieutenant",
		Blurb: "Junior officer with a bright pink targeting grid. Tends to over-commit, but lands the first kill of every game.",
		Color: "hotYellow",
		Bias:  Bias{Aggression: 0.95, ContinentFocus: 0.25, RiskTolerance: 0.85, BorderHardening: 0.25},
	},
}

// GeneralByID returns the persona definition or false.
func GeneralByID(id string) (General, bool) {
	for _, g := range Generals {
		if g.ID == id {
			return g, true
		}
	}
	return General{}, false
}

// PlayerColors is the cycling palette the engine assigns to slots so each
// player has a distinct, theme-matched color regardless of which generals
// were selected.
var PlayerColors = []string{
	"neonPink", "electricBlue", "neonGreen", "hotYellow", "synthPurple", "iceCyan",
}

// PickGenerals selects n distinct AI generals deterministically given a seed
// (so a "Restart" with the same settings can produce the same opponents if
// desired). For variety the order is rotated through the static slice.
func PickGenerals(n, rotation int) []General {
	if n <= 0 {
		return nil
	}
	if n > len(Generals) {
		n = len(Generals)
	}
	out := make([]General, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, Generals[(i+rotation)%len(Generals)])
	}
	return out
}
