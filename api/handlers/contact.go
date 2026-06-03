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

// ContactHandler adapts services.ContactService to HTTP.
type ContactHandler struct {
	svc *services.ContactService
}

func NewContactHandler(fs *firestore.Client) *ContactHandler {
	return &ContactHandler{svc: services.NewContactService(fs)}
}

func NewContactHandlerWithService(svc *services.ContactService) *ContactHandler {
	return &ContactHandler{svc: svc}
}

func (h *ContactHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	contacts, err := h.svc.List(r.Context(), uid)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, contacts)
}

func (h *ContactHandler) Search(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	q := r.URL.Query().Get("q")
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	contacts, err := h.svc.Search(r.Context(), uid, q, limit)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, contacts)
}

func (h *ContactHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var in services.Contact
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c, err := h.svc.Create(r.Context(), uid, services.ContactInput{
		Name:        in.Name,
		Email:       in.Email,
		Phone:       in.Phone,
		Company:     in.Company,
		Notes:       in.Notes,
		CharacterID: in.CharacterID,
	})
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *ContactHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	c, err := h.svc.Get(r.Context(), uid, id)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *ContactHandler) Patch(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	var patch map[string]any
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c, err := h.svc.Patch(r.Context(), uid, id, patch)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *ContactHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
