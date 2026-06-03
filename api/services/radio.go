package services

import (
	"encoding/json"
	"os"
)

// Track is one synthwave radio track. URLs must be CORS-enabled for the
// frontend's Web Audio AnalyserNode to read frequency data; if a host taints
// the stream, the visualizer falls back to its synthetic animation.
type Track struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	URL    string `json:"url"`
}

// defaultTracks point at freely-usable sample audio (SoundHelix). They are
// generic instrumentals dressed up with synthwave titles — swap the URLs (or
// set RADIO_TRACKS) for real synthwave. No copyrighted source is hardcoded.
var defaultTracks = []Track{
	{ID: "1", Title: "Midnight Driver", Artist: "NEON//GRID", URL: "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3"},
	{ID: "2", Title: "Chrome Sunset", Artist: "Vector Cassette", URL: "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-2.mp3"},
	{ID: "3", Title: "Afterglow Protocol", Artist: "Marvin & The Buffers", URL: "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-3.mp3"},
	{ID: "4", Title: "Starlit Boulevard", Artist: "Capt. Nimbus", URL: "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-8.mp3"},
}

// RadioTracks returns the configured track list. If the RADIO_TRACKS env var
// holds a JSON array of tracks, it overrides the defaults.
func RadioTracks() []Track {
	if raw := os.Getenv("RADIO_TRACKS"); raw != "" {
		var tracks []Track
		if err := json.Unmarshal([]byte(raw), &tracks); err == nil && len(tracks) > 0 {
			return tracks
		}
	}
	return defaultTracks
}
