package handlers

import (
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

func (h *RadioHandler) Tracks(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, services.RadioTracks())
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
