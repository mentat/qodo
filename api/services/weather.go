package services

import (
	"hash/fnv"
	"math/rand"
	"strings"
	"time"
)

// Weather is fully mocked — Marvin (per his persona) refuses real-time
// weather, so this tile is an in-joke. Forecasts are DETERMINISTIC per
// location: the same city always yields the same temps/conditions, derived
// by seeding a PRNG from the location string + day offset. No external API.

// DayForecast is one day's mocked forecast.
type DayForecast struct {
	Date      string `json:"date"` // YYYY-MM-DD
	Weekday   string `json:"weekday"`
	HighC     int    `json:"highC"`
	LowC      int    `json:"lowC"`
	Condition string `json:"condition"`
	Icon      string `json:"icon"`
	PrecipPct int    `json:"precipPct"`
}

// Forecast is a mocked multi-day forecast for a location.
type Forecast struct {
	Location string        `json:"location"`
	Days     []DayForecast `json:"days"`
}

type condition struct {
	name string
	icon string
}

// conditions are synthwave-flavored on purpose.
var conditions = []condition{
	{"Clear Skies", "sun"},
	{"Partly Cloudy", "cloud-sun"},
	{"Overcast", "cloud"},
	{"Neon Drizzle", "cloud-rain"},
	{"Synth Storm", "bolt"},
	{"Retro Haze", "haze"},
	{"Chrome Fog", "fog"},
}

// MockForecast returns a deterministic forecast for the given location and
// number of days (clamped to 1..14). Dates are anchored to today (UTC); the
// temperature/condition values depend only on location + day offset, so they
// are stable across calls.
func MockForecast(location string, days int) Forecast {
	if strings.TrimSpace(location) == "" {
		location = "Neo Kyoto"
	}
	if days <= 0 {
		days = 5
	}
	if days > 14 {
		days = 14
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	f := Forecast{Location: location, Days: make([]DayForecast, 0, days)}
	for i := 0; i < days; i++ {
		high, low, cond, precip := forecastValues(location, i)
		d := today.AddDate(0, 0, i)
		f.Days = append(f.Days, DayForecast{
			Date:      d.Format("2006-01-02"),
			Weekday:   d.Weekday().String(),
			HighC:     high,
			LowC:      low,
			Condition: cond.name,
			Icon:      cond.icon,
			PrecipPct: precip,
		})
	}
	return f
}

// forecastValues is the pure core: deterministic temps/condition/precip for a
// (location, dayOffset). Exposed for testing via the package-internal name.
func forecastValues(location string, offset int) (highC, lowC int, cond condition, precipPct int) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(location))))
	seed := int64(h.Sum32())<<8 ^ int64(offset)*2654435761
	r := rand.New(rand.NewSource(seed))

	high := 8 + r.Intn(26)   // 8..33 C
	low := high - (3 + r.Intn(9)) // 3..11 below high
	c := conditions[r.Intn(len(conditions))]
	precip := r.Intn(101)
	// Bias precip toward the condition so "Clear Skies" rarely shows 90%.
	switch c.name {
	case "Clear Skies":
		precip = r.Intn(15)
	case "Partly Cloudy", "Retro Haze", "Chrome Fog":
		precip = r.Intn(40)
	case "Neon Drizzle":
		precip = 40 + r.Intn(40)
	case "Synth Storm":
		precip = 70 + r.Intn(31)
	}
	return high, low, c, precip
}
