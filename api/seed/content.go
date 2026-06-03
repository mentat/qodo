package seed

// SeedEmail is a starter inbox message from a character. AgeHours places it
// in the recent past so the inbox looks lived-in on first login.
type SeedEmail struct {
	CharacterID string
	Subject     string
	Body        string
	AgeHours    float64
}

// SeedEmails is the pre-populated inbox. Each starts its own thread.
var SeedEmails = []SeedEmail{
	{
		CharacterID: "dot-matrix",
		Subject:     "did you get my last 47 pages??",
		AgeHours:    0.5,
		Body: `BZZT-CHHK. Hello! It is me, Dot Matrix. I printed your weekly agenda but the tractor feed slipped on page 12 and now pages 13 through 47 are... slightly diagonal. I AM SO SORRY. Should I reprint? Do we have toner? rrrRRT. Please advise. —DM`,
	},
	{
		CharacterID: "reginald",
		Subject:     "A Missive Concerning Your Available Memory",
		AgeHours:    3,
		Body: `My Dear User,

It is with the utmost cordiality that I write to inform you that your available working memory has, I regret to report, grown perilously fragmented — that is to say, perilously frag— forgive me, I overflowed. As I was saying: a tidying is in order. Might I suggest closing several of the seventeen tabs you have left ajar?

Your most obedient buffer,
Sir Reginald Buffer III`,
	},
	{
		CharacterID: "nimbus",
		Subject:     "Cloud migration is just weather, actually ☁️",
		AgeHours:    6,
		Body: `Ahoy! Capt. Nimbus here. Just wrapped the Q3 migration — smooth sailing through the stratocumulus, barely a dropped packet. I've plotted a course for your data and we should make landfall in us-central1 by Thursday. All hands on deck! Let me know if you hit any turbulence. —Carol`,
	},
	{
		CharacterID: "y2k",
		Subject:     "URGENT: only a few days until THE YEAR 2000",
		AgeHours:    20,
		Body: `It's me. Y2K. Do not panic but DO panic, appropriately. I have audited your date fields and three of them still use two-digit years. When the clock rolls over, "00" could mean 1900. I have set aside canned beans for us both. Please confirm receipt before midnight 12/31/99. —Y2K`,
	},
	{
		CharacterID: "moodboard",
		Subject:     "vibes for the week (a gradient)",
		AgeHours:    27,
		Body: `okok so. the week's energy is reading very magenta-to-cyan at about 45 degrees, with a soft VHS grain over everything. your todo list, aesthetically, is giving "unfinished but hopeful." i'd lean into the neon. lmk if you want me to mood-map the calendar too. ✨ —Moodboard`,
	},
	{
		CharacterID: "brad",
		Subject:     "quick sync to align on synergies?",
		AgeHours:    44,
		Body: `Hey hey! Brad from Procurement here. Wanted to circle back and put some time on the calendar to align on synergies and socialize a few low-hanging-fruit action items. Does a quick 30 work? Happy to take it offline and loop in the broader stakeholder group. Best, Brad`,
	},
	{
		CharacterID: "dot-matrix",
		Subject:     "PAPER JAM (this is a drill) (it is not a drill)",
		AgeHours:    50,
		Body: `rrrRRT. Minor incident. There is a paper jam in tray 2 and I have been told not to make it your problem, so I am making it your problem in the gentlest way possible. No rush. Unless? BZZT. —DM`,
	},
}

// SeedEvent is a starter calendar entry. DayOffset is relative to today;
// Hour is local hour; DurationMins defaults to 60 when zero.
type SeedEvent struct {
	Title        string
	Description  string
	Location     string
	CharacterID  string
	DayOffset    int
	Hour         int
	DurationMins int
	AllDay       bool
	Color        string
}

// SeedEvents spread a few character-hosted meetings across the coming week.
var SeedEvents = []SeedEvent{
	{Title: "Quick sync w/ Brad (synergies)", Description: "Circle back, align, socialize action items.", Location: "Conf Room: The Cloud", CharacterID: "brad", DayOffset: 1, Hour: 15, DurationMins: 30, Color: "#FF2E97"},
	{Title: "Defrag & Meditation", Description: "Hosted by Y2K. Bring canned goods.", Location: "Server Closet B", CharacterID: "y2k", DayOffset: 2, Hour: 9, DurationMins: 45, Color: "#00E5FF"},
	{Title: "Cloud Migration Standup", Description: "Capt. Nimbus plots the course. All hands on deck.", Location: "Bridge / us-central1", CharacterID: "nimbus", DayOffset: 3, Hour: 11, DurationMins: 30, Color: "#9B5DE5"},
	{Title: "Toner Replacement Ceremony", Description: "Dot Matrix requests your presence. There may be confetti (it is shredded paper).", Location: "Print Room", CharacterID: "dot-matrix", DayOffset: 4, Hour: 14, DurationMins: 30, Color: "#39FF14"},
	{Title: "Palette Review w/ Moodboard", Description: "Magenta-to-cyan, 45 degrees. Bring your aura.", CharacterID: "moodboard", DayOffset: 5, Hour: 13, DurationMins: 60, Color: "#FEE440"},
}

// SeedNote is a starter markdown note.
type SeedNote struct {
	Title string
	Body  string
	Tags  []string
}

// SeedNotes give the Notes app something to render on day one.
var SeedNotes = []SeedNote{
	{
		Title: "Welcome to Synthwave OS",
		Tags:  []string{"welcome", "meta"},
		Body: `# Welcome to **Synthwave OS** 🌆

Your retro-futurist productivity suite. Here's the tour:

- **Todos** — the original. Drag to reorder.
- **Mail** — reply to the cast and they'll write back *in character* (give it a few seconds).
- **Calendar** — month/week/day. Your due-dated todos show up as all-day items.
- **Contacts**, **Notes** (you're reading one), **Radio**, and **Weather**.

> Tip: open **Marvin** (the chat panel) and ask him to email someone or schedule a sync.

_Reset the demo data anytime from the Marvin / settings menu._`,
	},
	{
		Title: "Marvin's Reveries",
		Tags:  []string{"marvin"},
		Body: `## Reveries (things Marvin is keeping for you)

- Favorite color: **CRT green** (Marvin's, not necessarily yours)
- The toner situation is *ongoing*
- Brad still wants that sync
- "Cloud migration is just weather, actually" — Capt. Nimbus, profound

BZZT. End of reverie buffer.`,
	},
}
