// Package tts wraps Google Cloud Text-to-Speech for Marvin's voice replies.
//
// Marvin's persona is a glitchy 90s robot, so we lean on SSML <prosody> to
// pitch-shift his non-word tics — "BEEP", "BOOP", "BZZT", "whirrrr" — into
// something that actually sounds like a robot, not a confused British man
// saying "bee-eep". The MP3 audio is base64-friendly and plays in every
// browser's native <audio>.
//
// Voice: en-GB-Neural2-B. Neural2 supports SSML (Chirp3-HD doesn't) and is
// noticeably faster than WaveNet. British male English fits Marvin's dry,
// Hitchhiker's-Guide-adjacent vibe.
package tts

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	ttspb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
)

// Defaults tuned for Marvin: British male Neural2 voice, MP3 output, a touch
// over normal speed so replies feel snappy without sounding sped-up.
const (
	DefaultLanguageCode = "en-GB"
	DefaultVoiceName    = "en-GB-Neural2-B"
	DefaultSpeakingRate = 1.05
)

// Client is a thin wrapper around the Cloud TTS client preconfigured for
// Marvin's voice. Construct with [New]; release with Close on shutdown.
type Client struct {
	client *texttospeech.Client
	voice  *ttspb.VoiceSelectionParams
	audio  *ttspb.AudioConfig
}

// New constructs a TTS client using ADC. The caller should Close it on
// shutdown. Returns an error if the underlying client fails to initialize.
func New(ctx context.Context) (*Client, error) {
	c, err := texttospeech.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("tts: new client: %w", err)
	}
	return &Client{
		client: c,
		voice: &ttspb.VoiceSelectionParams{
			LanguageCode: DefaultLanguageCode,
			Name:         DefaultVoiceName,
		},
		audio: &ttspb.AudioConfig{
			AudioEncoding: ttspb.AudioEncoding_MP3,
			SpeakingRate:  DefaultSpeakingRate,
		},
	}, nil
}

// Synthesize turns Marvin's plain-text reply into MP3 audio. The input is
// first marked up with SSML (see [ToSSML]) so the robot tics sound right.
// Returns the raw MP3 bytes and the MIME type ("audio/mpeg").
func (c *Client) Synthesize(ctx context.Context, text string) ([]byte, string, error) {
	if c == nil || c.client == nil {
		return nil, "", fmt.Errorf("tts: client not initialized")
	}
	ssml := ToSSML(text)
	resp, err := c.client.SynthesizeSpeech(ctx, &ttspb.SynthesizeSpeechRequest{
		Input:       &ttspb.SynthesisInput{InputSource: &ttspb.SynthesisInput_Ssml{Ssml: ssml}},
		Voice:       c.voice,
		AudioConfig: c.audio,
	})
	if err != nil {
		return nil, "", fmt.Errorf("tts: synthesize: %w", err)
	}
	return resp.GetAudioContent(), "audio/mpeg", nil
}

// Close releases the underlying gRPC connection.
func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

// ToSSML converts Marvin's plain-text reply into SSML, escaping XML special
// characters and replacing his robot tics with <prosody>-wrapped pitch/rate
// substitutions so they sound mechanical instead of mumbled. Exported for
// testing.
func ToSSML(text string) string {
	// Strip markdown emphasis markers — Marvin often writes "*whirrrr*" or
	// "_BEEP_" for visual flair. Speaking the asterisks/underscores is bad.
	cleaned := stripMarkdownMarkers(text)
	// Escape first so user-controlled text can never inject SSML tags.
	escaped := xmlEscape(cleaned)
	// Then apply prosody substitutions. The patterns match alphabetic-only
	// tokens that survived escaping unchanged.
	withProsody := applyRobotProsody(escaped)
	return "<speak>" + withProsody + "</speak>"
}

// stripMarkdownMarkers removes `*` and `_` characters that wrap emphasized
// words — they're visual, not phonetic. We don't try to parse markdown
// fully; a blunt strip is enough for Marvin's tics like "*whirrrr*".
func stripMarkdownMarkers(s string) string {
	return strings.NewReplacer("*", "", "_", "").Replace(s)
}

func xmlEscape(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	).Replace(s)
}

// robotProsody is the ordered list of (pattern, SSML) substitutions applied
// to Marvin's tics. Order matters: longer tokens before shorter ones so
// "WHIRRRR" isn't half-matched by a "WHIR" rule.
var robotProsody = []struct {
	re   *regexp.Regexp
	ssml string
}{
	// "whirrr"/"whirrrrr"/"WHIRR" — extend duration, drop pitch slightly so it
	// sounds like a servo winding down.
	{
		re:   regexp.MustCompile(`(?i)\bwhirr+\b`),
		ssml: `<prosody pitch="-2st" rate="x-slow">whirrrr</prosody><break time="120ms"/>`,
	},
	// "BZZT" — the literal token has no vowels so Neural2 mumbles or swallows
	// it. <sub alias="buzz't"> gives the synthesizer a pronounceable surface
	// form ("buzz" + clipped t) and the heavy low/slow/loud prosody plus a
	// short gap after make it land like a real shorted-out robot buzzer.
	{
		re:   regexp.MustCompile(`(?i)\bbzz+t\b`),
		ssml: `<prosody pitch="-7st" rate="slow" volume="loud"><sub alias="buzz't">bzzt</sub></prosody><break time="140ms"/>`,
	},
	// "beep" — pitch up, faster. Crisp.
	{
		re:   regexp.MustCompile(`(?i)\bbeep\b`),
		ssml: `<prosody pitch="+8st" rate="fast">beep</prosody>`,
	},
	// "boop" — pitch down, faster. Counterpoint to "beep".
	{
		re:   regexp.MustCompile(`(?i)\bboop\b`),
		ssml: `<prosody pitch="-6st" rate="fast">boop</prosody>`,
	},
	// "AFFIRMATIVE" / "NEGATIVE" / "DOES NOT COMPUTE" — keep but flatten
	// prosody so they read as machine declarations rather than excited speech.
	{
		re:   regexp.MustCompile(`(?i)\b(affirmative|negative)\b`),
		ssml: `<prosody pitch="-2st" rate="medium">$1</prosody>`,
	},
	{
		re:   regexp.MustCompile(`(?i)\bdoes not compute\b`),
		ssml: `<prosody pitch="-3st" rate="slow">does not compute</prosody>`,
	},
}

func applyRobotProsody(s string) string {
	for _, p := range robotProsody {
		s = p.re.ReplaceAllString(s, p.ssml)
	}
	return s
}
