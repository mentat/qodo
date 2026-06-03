package handlers

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/mentat/qodo/api/middleware"
	"github.com/mentat/qodo/api/services"
)

// ── Weather (mocked) ──────────────────────────────────────────────────────

// WeatherHandler serves the deterministic mocked forecast.
type WeatherHandler struct{}

func NewWeatherHandler() *WeatherHandler { return &WeatherHandler{} }

// Forecast returns a mocked forecast. Query: ?location=&days=.
func (h *WeatherHandler) Forecast(w http.ResponseWriter, r *http.Request) {
	location := r.URL.Query().Get("location")
	days := 5
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			days = n
		}
	}
	writeJSON(w, http.StatusOK, services.MockForecast(location, days))
}

// ── Radio ─────────────────────────────────────────────────────────────────

// RadioHandler serves the static synthwave track list.
type RadioHandler struct{}

func NewRadioHandler() *RadioHandler { return &RadioHandler{} }

// Tracks returns the playlist with each track's URL rewritten to our own
// streaming proxy (a relative path the client resolves against API_BASE), so
// the browser plays same-origin audio and the upstream stays hidden.
func (h *RadioHandler) Tracks(w http.ResponseWriter, r *http.Request) {
	tracks := services.RadioTracks()
	out := make([]services.Track, len(tracks))
	for i, t := range tracks {
		t.URL = "/api/radio/stream?id=" + t.ID
		out[i] = t
	}
	writeJSON(w, http.StatusOK, out)
}

// Stream proxies a track's upstream MP3, relaying Range requests. Because it's
// served from our origin (with the API's permissive CORS), the Web Audio
// AnalyserNode reads real samples instead of a CORS-tainted (silent) stream.
// The id maps to a fixed configured track, so there's no open-proxy/SSRF risk.
func (h *RadioHandler) Stream(w http.ResponseWriter, r *http.Request) {
	track, ok := services.RadioTrackByID(r.URL.Query().Get("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, track.URL, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "bad upstream url")
		return
	}
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream fetch failed")
		return
	}
	defer resp.Body.Close()
	for _, hn := range []string{"Content-Type", "Content-Length", "Accept-Ranges", "Content-Range", "Last-Modified", "ETag"} {
		if v := resp.Header.Get(hn); v != "" {
			w.Header().Set(hn, v)
		}
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "audio/mpeg")
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// ── Calendar agenda (events + todos with due dates) ───────────────────────

// CalendarHandler joins events and due-dated todos into one agenda view —
// primarily for Marvin's calendar tool and an "agenda" endpoint.
type CalendarHandler struct {
	events *services.EventService
	todos  *services.TodoService
}

func NewCalendarHandler(events *services.EventService, todos *services.TodoService) *CalendarHandler {
	return &CalendarHandler{events: events, todos: todos}
}

// AgendaItem is a unified calendar entry; Kind is "event" or "todo".
type AgendaItem struct {
	Kind     string    `json:"kind"`
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	AllDay   bool      `json:"allDay"`
	Location string    `json:"location,omitempty"`
	Priority string    `json:"priority,omitempty"`
}

// Agenda returns events + due-dated todos in [from, to). Defaults to the next
// 30 days when from/to are omitted.
func (h *CalendarHandler) Agenda(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	now := time.Now().UTC()
	from, to := now.AddDate(0, 0, -7), now.AddDate(0, 0, 30)
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t
		}
	}

	events, err := h.events.ListRange(r.Context(), uid, from, to)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	items := make([]AgendaItem, 0, len(events))
	for _, e := range events {
		items = append(items, AgendaItem{
			Kind: "event", ID: e.ID, Title: e.Title, Start: e.Start, End: e.End,
			AllDay: e.AllDay, Location: e.Location,
		})
	}

	todos, err := h.todos.List(r.Context(), uid, services.ListFilter{})
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	for _, t := range todos {
		if t.DueDate == nil {
			continue
		}
		due := t.DueDate.UTC()
		if due.Before(from) || !due.Before(to) {
			continue
		}
		items = append(items, AgendaItem{
			Kind: "todo", ID: t.ID, Title: t.Title, Start: due, End: due,
			AllDay: true, Priority: t.Priority,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
