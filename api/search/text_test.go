package search

import (
	"reflect"
	"strings"
	"testing"
)

func TestTokenize(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"Hello, world!", []string{"hello", "world"}},
		{"Foo-bar_baz qux.", []string{"foo", "bar", "baz", "qux"}},
		// Short tokens dropped.
		{"I am a dog", []string{"am", "dog"}},
		// Digits kept.
		{"run 5 miles", []string{"run", "miles"}},
		{"Project Q3-2026", []string{"project", "q3", "2026"}},
	}
	for _, c := range cases {
		got := Tokenize(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Tokenize(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestStem(t *testing.T) {
	in := []string{"running", "runs", "ran", "buy", "buying", "bought"}
	got := Stem(in)
	// Both "running" and "runs" should collapse to "run".
	if got[0] != got[1] {
		t.Errorf("running/runs differ: %v", got)
	}
	// "buy" and "buying" should share a stem (Porter stems "buying" → "buy").
	if got[3] != got[4] {
		t.Errorf("buy/buying differ: %v", got)
	}
}

func TestStopwordRemoval(t *testing.T) {
	got := Stem(Tokenize("the quick brown fox jumps over the lazy dog"))
	for _, s := range got {
		if IsStopword(s) {
			t.Errorf("stopword survived: %q", s)
		}
	}
	// "the" and "over" should be gone (both are stopwords), but actual
	// content words remain.
	joined := strings.Join(got, " ")
	for _, want := range []string{"quick", "brown", "fox", "lazi", "dog"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected %q in stems, got %v", want, got)
		}
	}
}

func TestBuild_TopFive(t *testing.T) {
	got := Build("buy groceries", "remember the milk and eggs from the store")
	if len(got) > MaxTokens {
		t.Errorf("Build returned %d tokens, expected ≤ %d: %v", len(got), MaxTokens, got)
	}
	if len(got) == 0 {
		t.Fatal("Build returned no tokens")
	}
	// No stopwords should leak in.
	for _, s := range got {
		if IsStopword(s) {
			t.Errorf("stopword in Build output: %q (%v)", s, got)
		}
	}
}

func TestBuild_Stable(t *testing.T) {
	// Re-saving a todo with the same text must produce the same tokens so
	// Firestore doesn't write churn.
	a := Build("Pick up milk", "Need 2% milk from the store")
	b := Build("Pick up milk", "Need 2% milk from the store")
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Build is not stable: a=%v b=%v", a, b)
	}
}

func TestBuild_DedupesAcrossFields(t *testing.T) {
	got := Build("buy milk", "go buy milk")
	seen := map[string]int{}
	for _, t := range got {
		seen[t]++
	}
	for tok, n := range seen {
		if n > 1 {
			t.Errorf("duplicate %q occurs %d times in %v", tok, n, got)
		}
	}
}

func TestBuildQuery(t *testing.T) {
	// Users type naturally — "find me milk" should reduce to ~["milk"].
	got := BuildQuery("find me the milk please")
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "milk") {
		t.Errorf("query lost core term: %v", got)
	}
	// Stopwords gone.
	for _, s := range got {
		if IsStopword(s) {
			t.Errorf("stopword survived: %q", s)
		}
	}
}

func TestBuildQuery_MatchesBuildStems(t *testing.T) {
	// Round-trip: a word in the doc should match its stemmed form from the
	// query, so searching "running" finds a todo titled "run 5 miles"
	// (both stem to "run").
	idx := Build("run 5 miles today")
	q := BuildQuery("running")
	if len(q) != 1 {
		t.Fatalf("expected 1 query token, got %v", q)
	}
	found := false
	for _, t := range idx {
		if t == q[0] {
			found = true
		}
	}
	if !found {
		t.Errorf("query stem %q not in index stems %v", q[0], idx)
	}
}
