package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/go-chi/chi/v5"

	"github.com/mentat/qodo/api/middleware"
	"github.com/mentat/qodo/api/services"
)

// EventHandler adapts services.EventService to HTTP.
type EventHandler struct {
	svc *services.EventService
}

// NewEventHandler constructs a handler using a Firestore-backed service.
func NewEventHandler(fs *firestore.Client) *EventHandler {
	return &EventHandler{svc: services.NewEventService(fs)}
}

// NewEventHandlerWithService injects a service (tests).
func NewEventHandlerWithService(svc *services.EventService) *EventHandler {
	return &EventHandler{svc: svc}
}

// List returns events. With ?from=&to= (RFC3339) it returns only that range;
// otherwise it returns all of the user's events.
func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr != "" && toStr != "" {
		from, err1 := time.Parse(time.RFC3339, fromStr)
		to, err2 := time.Parse(time.RFC3339, toStr)
		if err1 != nil || err2 != nil {
			writeError(w, http.StatusBadRequest, "from/to must be RFC3339")
			return
		}
		events, err := h.svc.ListRange(r.Context(), uid, from, to)
		if err != nil {
			s, m := statusFor(err)
			writeError(w, s, m)
			return
		}
		writeJSON(w, http.StatusOK, events)
		return
	}
	events, err := h.svc.List(r.Context(), uid)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var in services.Event
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	e, err := h.svc.Create(r.Context(), uid, eventInputFrom(in))
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	e, err := h.svc.Get(r.Context(), uid, id)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// Update replaces the mutable fields of an event.
func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	var in services.Event
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	e, err := h.svc.Replace(r.Context(), uid, id, eventInputFrom(in))
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// Move reschedules an event (drag-and-drop). Body: {start, end?} RFC3339.
func (h *EventHandler) Move(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	var req struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	e, err := h.svc.Move(r.Context(), uid, id, req.Start, req.End)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *EventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func eventInputFrom(in services.Event) services.EventInput {
	return services.EventInput{
		Title:       in.Title,
		Description: in.Description,
		Location:    in.Location,
		Start:       in.Start,
		End:         in.End,
		AllDay:      in.AllDay,
		Color:       in.Color,
		CharacterID: in.CharacterID,
	}
}
