package risk

// This file is the source of truth for the static Risk board: 42 territories
// in 6 continents and the adjacency graph that ties them together.
//
// Adjacencies are the classic Hasbro rules — including the wrap-around
// trans-oceanic connections Alaska↔Kamchatka, Brazil↔North Africa,
// Western/Southern Europe↔North Africa, Southern Europe↔Egypt, East Africa↔
// Middle East, and Indonesia↔Siam. The adjacency map below is symmetric (an
// edge is encoded on both ends) and is validated by board_test.go.

// TerritoryDef describes one territory's static metadata. Adjacency is the
// list of every TerritoryID that shares a border (including ocean crossings).
type TerritoryDef struct {
	ID        TerritoryID
	Name      string
	Continent ContinentID
	Lat       float64 // approximate geographic centroid for the 3D globe
	Lon       float64
	Adjacent  []TerritoryID
}

// ContinentDef describes one continent's static metadata.
type ContinentDef struct {
	ID    ContinentID
	Name  string
	Bonus int // armies awarded per turn for fully controlling it
}

// Continent IDs.
const (
	ContinentNA   ContinentID = "north-america"
	ContinentSA   ContinentID = "south-america"
	ContinentEU   ContinentID = "europe"
	ContinentAF   ContinentID = "africa"
	ContinentAS   ContinentID = "asia"
	ContinentAUS  ContinentID = "australia"
)

// Continents is the 6-element static continent table.
var Continents = []ContinentDef{
	{ID: ContinentNA, Name: "North America", Bonus: 5},
	{ID: ContinentSA, Name: "South America", Bonus: 2},
	{ID: ContinentEU, Name: "Europe", Bonus: 5},
	{ID: ContinentAF, Name: "Africa", Bonus: 3},
	{ID: ContinentAS, Name: "Asia", Bonus: 7},
	{ID: ContinentAUS, Name: "Australia", Bonus: 2},
}

// Territory IDs. Lower-case kebab-case so they're stable URL/Firestore keys.
const (
	// North America
	TerrAlaska       TerritoryID = "alaska"
	TerrNWTerritory  TerritoryID = "northwest-territory"
	TerrAlberta      TerritoryID = "alberta"
	TerrOntario      TerritoryID = "ontario"
	TerrQuebec       TerritoryID = "quebec"
	TerrGreenland    TerritoryID = "greenland"
	TerrWesternUS    TerritoryID = "western-us"
	TerrEasternUS    TerritoryID = "eastern-us"
	TerrCentralAm    TerritoryID = "central-america"

	// South America
	TerrVenezuela TerritoryID = "venezuela"
	TerrBrazil    TerritoryID = "brazil"
	TerrPeru      TerritoryID = "peru"
	TerrArgentina TerritoryID = "argentina"

	// Europe
	TerrIceland       TerritoryID = "iceland"
	TerrGreatBritain  TerritoryID = "great-britain"
	TerrScandinavia   TerritoryID = "scandinavia"
	TerrUkraine       TerritoryID = "ukraine"
	TerrNorthernEU    TerritoryID = "northern-europe"
	TerrSouthernEU    TerritoryID = "southern-europe"
	TerrWesternEU     TerritoryID = "western-europe"

	// Africa
	TerrNorthAfrica TerritoryID = "north-africa"
	TerrEgypt       TerritoryID = "egypt"
	TerrEastAfrica  TerritoryID = "east-africa"
	TerrCongo       TerritoryID = "congo"
	TerrSouthAfrica TerritoryID = "south-africa"
	TerrMadagascar  TerritoryID = "madagascar"

	// Asia
	TerrUral        TerritoryID = "ural"
	TerrSiberia     TerritoryID = "siberia"
	TerrYakutsk     TerritoryID = "yakutsk"
	TerrIrkutsk     TerritoryID = "irkutsk"
	TerrKamchatka   TerritoryID = "kamchatka"
	TerrMongolia    TerritoryID = "mongolia"
	TerrJapan       TerritoryID = "japan"
	TerrChina       TerritoryID = "china"
	TerrAfghanistan TerritoryID = "afghanistan"
	TerrMiddleEast  TerritoryID = "middle-east"
	TerrIndia       TerritoryID = "india"
	TerrSiam        TerritoryID = "siam"

	// Australia
	TerrIndonesia        TerritoryID = "indonesia"
	TerrNewGuinea        TerritoryID = "new-guinea"
	TerrWesternAustralia TerritoryID = "western-australia"
	TerrEasternAustralia TerritoryID = "eastern-australia"
)

// Territories is the 42-element static territory table. Adjacencies must be
// symmetric — board_test.go enforces this.
var Territories = []TerritoryDef{
	// North America (5)
	{TerrAlaska, "Alaska", ContinentNA, 65, -150, []TerritoryID{TerrNWTerritory, TerrAlberta, TerrKamchatka}},
	{TerrNWTerritory, "NW Territory", ContinentNA, 70, -110, []TerritoryID{TerrAlaska, TerrAlberta, TerrOntario, TerrGreenland}},
	{TerrAlberta, "Alberta", ContinentNA, 55, -115, []TerritoryID{TerrAlaska, TerrNWTerritory, TerrOntario, TerrWesternUS}},
	{TerrOntario, "Ontario", ContinentNA, 55, -90, []TerritoryID{TerrNWTerritory, TerrAlberta, TerrWesternUS, TerrEasternUS, TerrQuebec, TerrGreenland}},
	{TerrQuebec, "Quebec", ContinentNA, 55, -75, []TerritoryID{TerrOntario, TerrEasternUS, TerrGreenland}},
	{TerrGreenland, "Greenland", ContinentNA, 75, -40, []TerritoryID{TerrNWTerritory, TerrOntario, TerrQuebec, TerrIceland}},
	{TerrWesternUS, "Western US", ContinentNA, 38, -110, []TerritoryID{TerrAlberta, TerrOntario, TerrEasternUS, TerrCentralAm}},
	{TerrEasternUS, "Eastern US", ContinentNA, 38, -85, []TerritoryID{TerrOntario, TerrQuebec, TerrWesternUS, TerrCentralAm}},
	{TerrCentralAm, "Central America", ContinentNA, 17, -90, []TerritoryID{TerrWesternUS, TerrEasternUS, TerrVenezuela}},

	// South America (2)
	{TerrVenezuela, "Venezuela", ContinentSA, 8, -65, []TerritoryID{TerrCentralAm, TerrBrazil, TerrPeru}},
	{TerrBrazil, "Brazil", ContinentSA, -10, -55, []TerritoryID{TerrVenezuela, TerrPeru, TerrArgentina, TerrNorthAfrica}},
	{TerrPeru, "Peru", ContinentSA, -10, -75, []TerritoryID{TerrVenezuela, TerrBrazil, TerrArgentina}},
	{TerrArgentina, "Argentina", ContinentSA, -35, -65, []TerritoryID{TerrPeru, TerrBrazil}},

	// Europe (5)
	{TerrIceland, "Iceland", ContinentEU, 65, -19, []TerritoryID{TerrGreenland, TerrGreatBritain, TerrScandinavia}},
	{TerrGreatBritain, "Great Britain", ContinentEU, 54, -3, []TerritoryID{TerrIceland, TerrScandinavia, TerrNorthernEU, TerrWesternEU}},
	{TerrScandinavia, "Scandinavia", ContinentEU, 62, 15, []TerritoryID{TerrIceland, TerrGreatBritain, TerrNorthernEU, TerrUkraine}},
	{TerrUkraine, "Ukraine", ContinentEU, 50, 30, []TerritoryID{TerrScandinavia, TerrNorthernEU, TerrSouthernEU, TerrUral, TerrAfghanistan, TerrMiddleEast}},
	{TerrNorthernEU, "Northern Europe", ContinentEU, 52, 15, []TerritoryID{TerrGreatBritain, TerrScandinavia, TerrUkraine, TerrSouthernEU, TerrWesternEU}},
	{TerrSouthernEU, "Southern Europe", ContinentEU, 44, 12, []TerritoryID{TerrNorthernEU, TerrUkraine, TerrWesternEU, TerrNorthAfrica, TerrEgypt, TerrMiddleEast}},
	{TerrWesternEU, "Western Europe", ContinentEU, 46, 2, []TerritoryID{TerrGreatBritain, TerrNorthernEU, TerrSouthernEU, TerrNorthAfrica}},

	// Africa (3)
	{TerrNorthAfrica, "North Africa", ContinentAF, 22, 8, []TerritoryID{TerrBrazil, TerrWesternEU, TerrSouthernEU, TerrEgypt, TerrEastAfrica, TerrCongo}},
	{TerrEgypt, "Egypt", ContinentAF, 27, 30, []TerritoryID{TerrSouthernEU, TerrNorthAfrica, TerrEastAfrica, TerrMiddleEast}},
	{TerrEastAfrica, "East Africa", ContinentAF, 5, 35, []TerritoryID{TerrEgypt, TerrNorthAfrica, TerrCongo, TerrSouthAfrica, TerrMadagascar, TerrMiddleEast}},
	{TerrCongo, "Congo", ContinentAF, -2, 22, []TerritoryID{TerrNorthAfrica, TerrEastAfrica, TerrSouthAfrica}},
	{TerrSouthAfrica, "South Africa", ContinentAF, -28, 25, []TerritoryID{TerrCongo, TerrEastAfrica, TerrMadagascar}},
	{TerrMadagascar, "Madagascar", ContinentAF, -20, 47, []TerritoryID{TerrEastAfrica, TerrSouthAfrica}},

	// Asia (7)
	{TerrUral, "Ural", ContinentAS, 60, 60, []TerritoryID{TerrUkraine, TerrSiberia, TerrChina, TerrAfghanistan}},
	{TerrSiberia, "Siberia", ContinentAS, 65, 95, []TerritoryID{TerrUral, TerrYakutsk, TerrIrkutsk, TerrMongolia, TerrChina}},
	{TerrYakutsk, "Yakutsk", ContinentAS, 65, 130, []TerritoryID{TerrSiberia, TerrIrkutsk, TerrKamchatka}},
	{TerrIrkutsk, "Irkutsk", ContinentAS, 58, 110, []TerritoryID{TerrSiberia, TerrYakutsk, TerrKamchatka, TerrMongolia}},
	{TerrKamchatka, "Kamchatka", ContinentAS, 55, 160, []TerritoryID{TerrYakutsk, TerrIrkutsk, TerrMongolia, TerrJapan, TerrAlaska}},
	{TerrMongolia, "Mongolia", ContinentAS, 47, 105, []TerritoryID{TerrSiberia, TerrIrkutsk, TerrKamchatka, TerrChina, TerrJapan}},
	{TerrJapan, "Japan", ContinentAS, 36, 138, []TerritoryID{TerrKamchatka, TerrMongolia}},
	{TerrChina, "China", ContinentAS, 35, 105, []TerritoryID{TerrUral, TerrSiberia, TerrMongolia, TerrAfghanistan, TerrIndia, TerrSiam}},
	{TerrAfghanistan, "Afghanistan", ContinentAS, 35, 65, []TerritoryID{TerrUral, TerrUkraine, TerrChina, TerrIndia, TerrMiddleEast}},
	{TerrMiddleEast, "Middle East", ContinentAS, 30, 50, []TerritoryID{TerrUkraine, TerrSouthernEU, TerrEgypt, TerrEastAfrica, TerrAfghanistan, TerrIndia}},
	{TerrIndia, "India", ContinentAS, 22, 78, []TerritoryID{TerrChina, TerrAfghanistan, TerrMiddleEast, TerrSiam}},
	{TerrSiam, "Siam", ContinentAS, 13, 100, []TerritoryID{TerrChina, TerrIndia, TerrIndonesia}},

	// Australia (4)
	{TerrIndonesia, "Indonesia", ContinentAUS, -3, 115, []TerritoryID{TerrSiam, TerrNewGuinea, TerrWesternAustralia}},
	{TerrNewGuinea, "New Guinea", ContinentAUS, -5, 142, []TerritoryID{TerrIndonesia, TerrWesternAustralia, TerrEasternAustralia}},
	{TerrWesternAustralia, "Western Australia", ContinentAUS, -25, 122, []TerritoryID{TerrIndonesia, TerrNewGuinea, TerrEasternAustralia}},
	{TerrEasternAustralia, "Eastern Australia", ContinentAUS, -25, 145, []TerritoryID{TerrNewGuinea, TerrWesternAustralia}},
}

// territoryByID is the lookup index built at init().
var territoryByID = map[TerritoryID]TerritoryDef{}

// adjacency is the symmetric undirected graph built at init() — adjacency[a]
// is the set of every territory that borders a.
var adjacency = map[TerritoryID]map[TerritoryID]struct{}{}

// continentTerritories[c] is the list of every territory in continent c.
var continentTerritories = map[ContinentID][]TerritoryID{}

func init() {
	for _, t := range Territories {
		territoryByID[t.ID] = t
		continentTerritories[t.Continent] = append(continentTerritories[t.Continent], t.ID)
	}
	for _, t := range Territories {
		if adjacency[t.ID] == nil {
			adjacency[t.ID] = map[TerritoryID]struct{}{}
		}
		for _, n := range t.Adjacent {
			adjacency[t.ID][n] = struct{}{}
			// Symmetrize defensively even though Territories already lists both ends.
			if adjacency[n] == nil {
				adjacency[n] = map[TerritoryID]struct{}{}
			}
			adjacency[n][t.ID] = struct{}{}
		}
	}
}

// Adjacent reports whether territories a and b share a border (or
// trans-oceanic route).
func Adjacent(a, b TerritoryID) bool {
	_, ok := adjacency[a][b]
	return ok
}

// Neighbors returns the territories that border t.
func Neighbors(t TerritoryID) []TerritoryID {
	out := make([]TerritoryID, 0, len(adjacency[t]))
	for n := range adjacency[t] {
		out = append(out, n)
	}
	return out
}

// TerritoryByID returns the static definition of a territory.
func TerritoryByID(id TerritoryID) (TerritoryDef, bool) {
	t, ok := territoryByID[id]
	return t, ok
}

// ContinentOf returns the continent the given territory belongs to.
func ContinentOf(t TerritoryID) ContinentID {
	td, ok := territoryByID[t]
	if !ok {
		return ""
	}
	return td.Continent
}

// TerritoriesIn returns the territories on continent c.
func TerritoriesIn(c ContinentID) []TerritoryID {
	out := make([]TerritoryID, len(continentTerritories[c]))
	copy(out, continentTerritories[c])
	return out
}

// ContinentBonus returns the army bonus for fully controlling continent c.
func ContinentBonus(c ContinentID) int {
	for _, def := range Continents {
		if def.ID == c {
			return def.Bonus
		}
	}
	return 0
}

// AllTerritoryIDs returns every territory ID, ordered as in Territories.
func AllTerritoryIDs() []TerritoryID {
	out := make([]TerritoryID, len(Territories))
	for i, t := range Territories {
		out[i] = t.ID
	}
	return out
}
