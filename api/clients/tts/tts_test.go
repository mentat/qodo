package tts

import (
	"strings"
	"testing"
)

func TestToSSML_WrapsInSpeak(t *testing.T) {
	got := ToSSML("hello")
	if !strings.HasPrefix(got, "<speak>") || !strings.HasSuffix(got, "</speak>") {
		t.Fatalf("expected <speak>...</speak> wrapper, got %q", got)
	}
}

func TestToSSML_EscapesXML(t *testing.T) {
	got := ToSSML(`5 < 6 & "yes" said 'Marvin'`)
	for _, s := range []string{"&lt;", "&amp;", "&quot;", "&apos;"} {
		if !strings.Contains(got, s) {
			t.Errorf("expected %q in escaped SSML, got %q", s, got)
		}
	}
	if strings.ContainsAny(got[len("<speak>"):len(got)-len("</speak>")], "<>") {
		// The only < and > in the body should come from inserted prosody tags,
		// which this input has none of.
		t.Errorf("unescaped angle brackets in body: %q", got)
	}
}

func TestToSSML_StripsMarkdownAroundTics(t *testing.T) {
	got := ToSSML("*whirrrr* the system boots")
	if strings.Contains(got, "*") || strings.Contains(got, "_") {
		t.Errorf("expected markdown markers stripped, got %q", got)
	}
	if !strings.Contains(got, "<prosody") {
		t.Errorf("expected prosody markup, got %q", got)
	}
}

func TestToSSML_RobotTicsAreProsodyWrapped(t *testing.T) {
	cases := []struct{ in, mustContain string }{
		{"BEEP boop", `pitch="+8st"`},
		{"BEEP boop", `pitch="-6st"`},
		{"BZZT something", `pitch="-7st"`},
		{"BZZT something", `<sub alias="buzz't">bzzt</sub>`},
		{"whirrrr...", `rate="x-slow"`},
		{"AFFIRMATIVE, human", `prosody`},
		{"does not compute", `prosody`},
	}
	for _, tc := range cases {
		got := ToSSML(tc.in)
		if !strings.Contains(got, tc.mustContain) {
			t.Errorf("ToSSML(%q) = %q; want substring %q", tc.in, got, tc.mustContain)
		}
	}
}

func TestToSSML_NoBleedAcrossWords(t *testing.T) {
	// Word-boundary matching: don't mangle "beeper" into "<prosody>beep</prosody>er".
	got := ToSSML("the beeper rings")
	if strings.Contains(got, "<prosody") {
		t.Errorf("expected no prosody on 'beeper', got %q", got)
	}
}

func TestToSSML_WhirrFlexibleLength(t *testing.T) {
	for _, in := range []string{"whirr", "whirrr", "whirrrrrr", "WHIRRRRR"} {
		got := ToSSML(in)
		if !strings.Contains(got, "whirrrr") {
			t.Errorf("ToSSML(%q) = %q; want canonical 'whirrrr' substitution", in, got)
		}
	}
}
