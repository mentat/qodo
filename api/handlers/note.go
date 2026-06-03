package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"cloud.google.com/go/firestore"
	"github.com/go-chi/chi/v5"

	"github.com/mentat/qodo/api/middleware"
	"github.com/mentat/qodo/api/services"
)

// NoteHandler adapts services.NoteService to HTTP.
type NoteHandler struct {
	svc *services.NoteService
}

func NewNoteHandler(fs *firestore.Client) *NoteHandler {
	return &NoteHandler{svc: services.NewNoteService(fs)}
}

func NewNoteHandlerWithService(svc *services.NoteService) *NoteHandler {
	return &NoteHandler{svc: svc}
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	notes, err := h.svc.List(r.Context(), uid)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, notes)
}

func (h *NoteHandler) Search(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	q := r.URL.Query().Get("q")
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	notes, err := h.svc.Search(r.Context(), uid, q, limit)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, notes)
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var in services.Note
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	n, err := h.svc.Create(r.Context(), uid, services.NoteInput{Title: in.Title, Body: in.Body, Tags: in.Tags})
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusCreated, n)
}

func (h *NoteHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	n, err := h.svc.Get(r.Context(), uid, id)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

// Update replaces a note's mutable fields.
func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	var in services.Note
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	n, err := h.svc.Replace(r.Context(), uid, id, services.NoteInput{Title: in.Title, Body: in.Body, Tags: in.Tags})
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
