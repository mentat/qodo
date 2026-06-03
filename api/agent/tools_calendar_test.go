package agent

import (
	"testing"
	"time"
)

func TestEventRange(t *testing.T) {
	now := time.Date(2026, time.June, 3, 12, 0, 0, 0, time.UTC)

	// Single-day query (from == to, date-only) must cover the WHOLE day, so a
	// 3pm event tomorrow is included — the bug that made Marvin say "empty".
	from, to := eventRange("2026-06-04", "2026-06-04", now)
	wantFrom := time.Date(2026, time.June, 4, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, time.June, 5, 0, 0, 0, 0, time.UTC)
	if !from.Equal(wantFrom) || !to.Equal(wantTo) {
		t.Fatalf("single-day range = [%v, %v), want [%v, %v)", from, to, wantFrom, wantTo)
	}
	afternoon := time.Date(2026, time.June, 4, 15, 0, 0, 0, time.UTC)
	if !(afternoon.After(from) && afternoon.Before(to)) {
		t.Errorf("3pm event %v should fall within [%v, %v)", afternoon, from, to)
	}

	// Empty args → sensible default window bracketing now.
	from, to = eventRange("", "", now)
	if !from.Before(now) || !to.After(now) {
		t.Errorf("default range [%v, %v) should bracket %v", from, to, now)
	}

	// Inverted from/to is guarded into a valid forward window.
	from, to = eventRange("2026-06-10", "2026-06-01", now)
	if !to.After(from) {
		t.Errorf("inverted range should be guarded: [%v, %v)", from, to)
	}
}
