package agent

import (
	"strings"
	"testing"

	"github.com/mentat/qodo/api/seed"
	"github.com/mentat/qodo/api/services"
)

func TestParseReplyJSON(t *testing.T) {
	s, b, err := parseReplyJSON(`{"subject":"Re: hi","body":"hello there"}`)
	if err != nil {
		t.Fatalf("valid json: %v", err)
	}
	if s != "Re: hi" || b != "hello there" {
		t.Errorf("got subject=%q body=%q", s, b)
	}

	// Leading/trailing whitespace + field trimming.
	s, b, err = parseReplyJSON("  {\"subject\":\" S \",\"body\":\"B\"}  ")
	if err != nil || s != "S" || b != "B" {
		t.Errorf("trim: subject=%q body=%q err=%v", s, b, err)
	}

	if _, _, err := parseReplyJSON(`{"subject":"x","body":"   "}`); err == nil {
		t.Error("empty body should error")
	}
	if _, _, err := parseReplyJSON(`not json`); err == nil {
		t.Error("invalid json should error")
	}
}

func TestBuildReplyPrompt(t *testing.T) {
	ch := seed.Characters[0]
	thread := []services.Email{
		{FromName: "You", Subject: "Hello", Body: "are you there?"},
		{FromName: ch.Name, Subject: "Re: Hello", Body: "BZZT yes"},
	}
	p := buildReplyPrompt(ch, thread)
	for _, want := range []string{ch.Name, "are you there?", "BZZT yes", "MOST RECENT"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt missing %q\n%s", want, p)
		}
	}
}

func TestBuildGreetingPrompt(t *testing.T) {
	ch := seed.Characters[1]
	p := buildGreetingPrompt(ch)
	if !strings.Contains(p, ch.Name) || !strings.Contains(p, "unprompted") {
		t.Errorf("greeting prompt: %s", p)
	}
}
