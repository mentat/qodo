package agent

import (
	"strings"
	"testing"
	"time"
)

// instructionWithDate must append a concrete date anchor so Marvin can answer
// "what's due today?" instead of claiming he has no clock. This is a pure,
// credential-free unit test (the integration tests in agent_test.go need
// live Vertex AI).
func TestInstructionWithDate(t *testing.T) {
	now := time.Date(2026, time.June, 3, 9, 30, 0, 0, time.UTC)
	out := instructionWithDate("BASE PROMPT", now)

	if !strings.HasPrefix(out, "BASE PROMPT") {
		t.Errorf("base prompt should be preserved at the start; got: %q", out)
	}
	// The weekday matters for relative dates like "next Friday"; the ISO date
	// is what Marvin should echo when writing due dates.
	for _, want := range []string{"CURRENT DATE", "Wednesday", "2026-06-03"} {
		if !strings.Contains(out, want) {
			t.Errorf("anchor missing %q; got: %q", want, out)
		}
	}
}
