package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"cloud.google.com/go/firestore"
	"github.com/go-chi/chi/v5"

	"github.com/mentat/qodo/api/middleware"
	"github.com/mentat/qodo/api/seed"
	"github.com/mentat/qodo/api/services"
)

// EmailHandler adapts services.EmailService to HTTP.
type EmailHandler struct {
	svc *services.EmailService
}

// NewEmailHandler constructs a handler using a Firestore-backed service.
func NewEmailHandler(fs *firestore.Client) *EmailHandler {
	return &EmailHandler{svc: services.NewEmailService(fs)}
}

// NewEmailHandlerWithService injects a service (e.g. one with a Publisher).
func NewEmailHandlerWithService(svc *services.EmailService) *EmailHandler {
	return &EmailHandler{svc: svc}
}

// List returns the user's inbox (newest first). Optional ?limit=.
func (h *EmailHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	emails, err := h.svc.ListInbox(r.Context(), uid, limit)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, emails)
}

// Send persists an outbound email and may trigger an async in-character reply.
func (h *EmailHandler) Send(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var req struct {
		To          string `json:"to"`
		ToName      string `json:"toName"`
		Subject     string `json:"subject"`
		Body        string `json:"body"`
		ThreadID    string `json:"threadId"`
		CharacterID string `json:"characterId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Resolve the recipient to a character so the async reply fires, even when
	// the client didn't supply a characterId (e.g. a manually-typed address).
	characterID := req.CharacterID
	toName := req.ToName
	if characterID == "" {
		if ch, ok := seed.CharacterByEmail(req.To); ok {
			characterID = ch.ID
			if toName == "" {
				toName = ch.Name
			}
		}
	}
	e, err := h.svc.Send(r.Context(), uid, services.SendInput{
		To:          req.To,
		ToName:      toName,
		Subject:     req.Subject,
		Body:        req.Body,
		ThreadID:    req.ThreadID,
		CharacterID: characterID,
	})
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// Thread returns all messages in a thread, oldest first.
func (h *EmailHandler) Thread(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	threadID := chi.URLParam(r, "threadId")
	emails, err := h.svc.ListThread(r.Context(), uid, threadID)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, emails)
}

// Get returns a single email.
func (h *EmailHandler) Get(w http.ResponseWriter, r *http.Request) {
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

// MarkRead flips read=true.
func (h *EmailHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	e, err := h.svc.MarkRead(r.Context(), uid, id)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// Delete removes an email.
func (h *EmailHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
