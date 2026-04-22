// Package search provides text processing utilities for the full-text
// search over todos. The pipeline is deliberately tiny:
//
//	tokenize → lowercase → drop stopwords → Porter stem → dedupe → cap
//
// Build() returns the resulting slice which is persisted as a
// repeated-string field on each Todo. Firestore's `array-contains-any`
// operator then functions as a small inverted index: any todo whose
// `fullText` contains one of the stemmed query tokens shows up in a
// single index scan.
package search

import (
	"sort"
	"strings"
	"unicode"

	"github.com/kljensen/snowball/english"
)

// MaxTokens caps how many stemmed terms we persist per todo. Capping
// keeps the inverted-index entries small (Firestore charges per field and
// array-contains-any caps at 30 values anyway) and in practice a todo's
// signal concentrates in its first few content words.
const MaxTokens = 5

// stopwords is a small set of extremely common English function words.
// We don't need a full stopword list (NLTK has ~180) — losing "the", "a",
// "of", and similar is what matters so real content tokens take the
// limited MaxTokens slots.
var stopwords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {},
	"but": {}, "by": {}, "for": {}, "from": {}, "has": {}, "have": {},
	"i": {}, "if": {}, "in": {}, "into": {}, "is": {}, "it": {}, "its": {},
	"me": {}, "my": {}, "no": {}, "not": {}, "of": {}, "off": {}, "on": {},
	"or": {}, "our": {}, "so": {}, "than": {}, "that": {}, "the": {},
	"their": {}, "them": {}, "there": {}, "these": {}, "they": {},
	"this": {}, "those": {}, "to": {}, "too": {}, "very": {}, "was": {},
	"we": {}, "were": {}, "will": {}, "with": {}, "would": {}, "you": {},
	"your": {}, "do": {}, "does": {}, "did": {}, "done": {}, "just": {},
	"about": {}, "can": {}, "could": {}, "should": {}, "up": {}, "down": {},
	"out": {}, "over": {}, "under": {}, "been": {},
}

// Tokenize lowercases `s` and splits it on any non-letter/non-digit rune.
// Short tokens (< 2 chars) are dropped — "a", "I", etc.
func Tokenize(s string) []string {
	if s == "" {
		return nil
	}
	lower := strings.ToLower(s)
	fields := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := fields[:0]
	for _, f := range fields {
		if len(f) >= 2 {
			out = append(out, f)
		}
	}
	return out
}

// IsStopword reports whether t (already lowercased) is in the stopword set.
func IsStopword(t string) bool {
	_, ok := stopwords[t]
	return ok
}

// Stem applies the Snowball Porter2 stemmer to each non-stopword token
// and returns the stemmed forms. It preserves input order and allows
// duplicates — callers that need uniqueness should Dedupe afterward.
func Stem(tokens []string) []string {
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if IsStopword(t) {
			continue
		}
		out = append(out, english.Stem(t, false))
	}
	return out
}

// Dedupe preserves first-occurrence order.
func Dedupe(tokens []string) []string {
	seen := make(map[string]struct{}, len(tokens))
	out := tokens[:0]
	for _, t := range tokens {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// Cap truncates to at most n tokens, preserving order.
func Cap(tokens []string, n int) []string {
	if n <= 0 || len(tokens) <= n {
		return tokens
	}
	return tokens[:n]
}

// Build is the full pipeline: join the parts with spaces, tokenize,
// drop stopwords, stem, dedupe, cap to MaxTokens. The returned slice is
// what lives in Firestore as the todo's fullText index entries.
func Build(parts ...string) []string {
	raw := strings.Join(parts, " ")
	tokens := Tokenize(raw)
	stems := Stem(tokens)
	stems = Dedupe(stems)
	// Sort alphabetically before capping so the chosen subset is stable
	// across updates that don't change the underlying text. (Without
	// sorting, a re-save that just trims whitespace could rotate which
	// 5 tokens are kept.)
	sort.Strings(stems)
	return Cap(stems, MaxTokens)
}

// BuildQuery prepares a user's search string for matching against the
// inverted index: tokenize, drop stopwords, stem, dedupe. We do NOT cap
// the query at MaxTokens — but Firestore's array-contains-any caps at
// 30, so we clamp to that.
func BuildQuery(q string) []string {
	tokens := Tokenize(q)
	stems := Stem(tokens)
	stems = Dedupe(stems)
	return Cap(stems, 30)
}
