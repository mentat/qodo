package services_test

import (
	"testing"

	"github.com/mentat/qodo/api/services"
)

func TestMockForecast_Deterministic(t *testing.T) {
	a := services.MockForecast("Neo Tokyo", 5)
	b := services.MockForecast("Neo Tokyo", 5)
	if len(a.Days) != 5 || len(b.Days) != 5 {
		t.Fatalf("want 5 days, got %d / %d", len(a.Days), len(b.Days))
	}
	for i := range a.Days {
		if a.Days[i] != b.Days[i] {
			t.Errorf("day %d not deterministic: %+v vs %+v", i, a.Days[i], b.Days[i])
		}
	}
}

func TestMockForecast_LocationVaries(t *testing.T) {
	a := services.MockForecast("Reykjavik", 7)
	b := services.MockForecast("Dubai", 7)
	differs := false
	for i := range a.Days {
		if a.Days[i].HighC != b.Days[i].HighC || a.Days[i].Condition != b.Days[i].Condition {
			differs = true
			break
		}
	}
	if !differs {
		t.Errorf("expected different forecasts for different locations")
	}
}

func TestMockForecast_DaysClampAndConsistency(t *testing.T) {
	if got := len(services.MockForecast("x", 0).Days); got != 5 {
		t.Errorf("days=0 should default to 5, got %d", got)
	}
	if got := len(services.MockForecast("x", 99).Days); got != 14 {
		t.Errorf("days=99 should clamp to 14, got %d", got)
	}
	// High must always be >= Low and precip within [0,100].
	for _, d := range services.MockForecast("Gridport", 10).Days {
		if d.HighC < d.LowC {
			t.Errorf("high < low: %+v", d)
		}
		if d.PrecipPct < 0 || d.PrecipPct > 100 {
			t.Errorf("precip out of range: %+v", d)
		}
		if d.Condition == "" || d.Icon == "" {
			t.Errorf("missing condition/icon: %+v", d)
		}
	}
}

func TestMockForecast_EmptyLocationDefaults(t *testing.T) {
	f := services.MockForecast("   ", 3)
	if f.Location != "Neo Kyoto" {
		t.Errorf("empty location should default, got %q", f.Location)
	}
	if len(f.Days) != 3 {
		t.Errorf("want 3 days, got %d", len(f.Days))
	}
}
