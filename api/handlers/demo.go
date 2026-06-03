package handlers

import (
	"net/http"

	"github.com/mentat/qodo/api/middleware"
	"github.com/mentat/qodo/api/services"
)

// DemoHandler exposes per-user demo seeding + reset.
type DemoHandler struct {
	seed *services.SeedService
}

func NewDemoHandler(seed *services.SeedService) *DemoHandler {
	return &DemoHandler{seed: seed}
}

// Seed plants demo content if the user hasn't been seeded. Idempotent — the
// frontend calls it on every login.
func (h *DemoHandler) Seed(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	seeded, err := h.seed.Seed(r.Context(), uid)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"seeded": seeded})
}

// Reset wipes the user's seeded suite content and re-plants it.
func (h *DemoHandler) Reset(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if err := h.seed.Reset(r.Context(), uid); err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
