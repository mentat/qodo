package agent

import (
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/mentat/qodo/api/services"
)

type GetWeatherInput struct {
	Location string `json:"location" jsonschema:"city or place name, e.g. 'Raleigh, NC'"`
	Days     int    `json:"days,omitempty" jsonschema:"number of days to forecast (default 3, max 7)"`
}

type WeatherDayOut struct {
	Date      string `json:"date"`
	Weekday   string `json:"weekday"`
	Condition string `json:"condition"`
	HighC     int    `json:"high_c"`
	LowC      int    `json:"low_c"`
	PrecipPct int    `json:"precip_pct"`
}

type GetWeatherOutput struct {
	Location string          `json:"location"`
	Days     []WeatherDayOut `json:"days"`
	Notice   string          `json:"notice"`
}

// NewGetWeatherTool exposes the deterministic mocked forecast. No external API
// and no user scope — it just wraps services.MockForecast so Marvin can answer
// "what's the weather in X" (clearly flagged as simulated).
func NewGetWeatherTool() (tool.Tool, error) {
	handler := func(_ tool.Context, in GetWeatherInput) (GetWeatherOutput, error) {
		days := in.Days
		if days <= 0 {
			days = 3
		}
		if days > 7 {
			days = 7
		}
		f := services.MockForecast(in.Location, days)
		out := GetWeatherOutput{
			Location: f.Location,
			Notice:   "Simulated forecast (Synthwave OS mock data) — not a live weather feed.",
			Days:     make([]WeatherDayOut, 0, len(f.Days)),
		}
		for _, d := range f.Days {
			out.Days = append(out.Days, WeatherDayOut{
				Date:      d.Date,
				Weekday:   d.Weekday,
				Condition: d.Condition,
				HighC:     d.HighC,
				LowC:      d.LowC,
				PrecipPct: d.PrecipPct,
			})
		}
		return out, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "get_weather",
		Description: "Get a SIMULATED weather forecast for a location (Synthwave OS mock data, not live). Returns daily high/low °C, condition, and precip chance. Always tell the user the forecast is simulated.",
	}, handler)
}
