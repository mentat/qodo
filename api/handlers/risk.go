package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mentat/qodo/api/middleware"
	"github.com/mentat/qodo/api/services/risk"
)

// RiskHandler adapts the Risk Store to HTTP. All endpoints are
// Firebase-auth-protected via middleware.NewAuthMiddleware — the user ID is
// the active-game key.
type RiskHandler struct {
	store *risk.Store
}

// NewRiskHandler constructs a handler from a Store.
func NewRiskHandler(store *risk.Store) *RiskHandler {
	return &RiskHandler{store: store}
}

// riskStatusFor maps engine errors to HTTP responses.
func riskStatusFor(err error) (int, string) {
	switch {
	case errors.Is(err, risk.ErrNoGame):
		return http.StatusNotFound, "no active risk game"
	case errors.Is(err, risk.ErrNotYourTurn):
		return http.StatusForbidden, "not your turn"
	case errors.Is(err, risk.ErrWrongPhase),
		errors.Is(err, risk.ErrInvalidPlacement),
		errors.Is(err, risk.ErrInvalidAttack),
		errors.Is(err, risk.ErrInvalidFortify),
		errors.Is(err, risk.ErrInvalidCardSet),
		errors.Is(err, risk.ErrInvalidSetup),
		errors.Is(err, risk.ErrPostConquestPending):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, risk.ErrGameOver):
		return http.StatusConflict, "game is over"
	}
	return http.StatusInternalServerError, "internal error"
}

func (h *RiskHandler) writeErr(w http.ResponseWriter, err error) {
	s, m := riskStatusFor(err)
	writeError(w, s, m)
}

// New starts a fresh game, replacing any existing one.
func (h *RiskHandler) New(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var req struct {
		Difficulty  string `json:"difficulty"`
		PlayerCount int    `json:"playerCount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	state, err := h.store.StartGame(r.Context(), uid, risk.Settings{
		Difficulty:  risk.Difficulty(req.Difficulty),
		PlayerCount: req.PlayerCount,
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, state)
}

// Get returns the current game state (also available via Firestore onSnapshot
// on the client; this endpoint is the REST fallback / first-load fetch).
func (h *RiskHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	state, err := h.store.Get(r.Context(), uid)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// Stats returns lifetime aggregates for the start screen.
func (h *RiskHandler) Stats(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	stats, err := h.store.Stats(r.Context(), uid)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// PlaceInitial places one army during the alternating setup phase.
func (h *RiskHandler) PlaceInitial(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var req struct {
		Territory string `json:"territory"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	state, err := h.store.PlaceInitialAction(r.Context(), uid, risk.TerritoryID(req.Territory))
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// Place places N armies on a territory during the reinforcement phase.
func (h *RiskHandler) Place(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var req struct {
		Territory string `json:"territory"`
		Count     int    `json:"count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Count <= 0 {
		req.Count = 1
	}
	state, err := h.store.PlaceAction(r.Context(), uid, risk.TerritoryID(req.Territory), req.Count)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// Trade trades a set of 3 cards.
func (h *RiskHandler) Trade(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var req struct {
		CardIDs []string `json:"cardIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	state, err := h.store.TradeAction(r.Context(), uid, req.CardIDs)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// Attack performs one or more dice rounds between two adjacent territories.
func (h *RiskHandler) Attack(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
		Mode string `json:"mode"` // "single" | "blitz"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	mode := risk.AttackBlitz
	if req.Mode == "single" {
		mode = risk.AttackSingle
	}
	state, rounds, err := h.store.AttackAction(r.Context(), uid, risk.TerritoryID(req.From), risk.TerritoryID(req.To), mode)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"state":  state,
		"rounds": rounds,
	})
}

// PostConquest finalizes the army move into a freshly-captured territory.
func (h *RiskHandler) PostConquest(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var req struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	state, err := h.store.PostConquestAction(r.Context(), uid, req.Count)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// Fortify performs one fortification (single adjacent edge, classic rule).
func (h *RiskHandler) Fortify(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var req struct {
		From  string `json:"from"`
		To    string `json:"to"`
		Count int    `json:"count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	state, err := h.store.FortifyAction(r.Context(), uid, risk.TerritoryID(req.From), risk.TerritoryID(req.To), req.Count)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// EndPhase advances Place → Attack → Fortify.
func (h *RiskHandler) EndPhase(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	state, err := h.store.EndPhaseAction(r.Context(), uid)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// SkipFortify ends the human turn without fortifying.
func (h *RiskHandler) SkipFortify(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	state, err := h.store.SkipFortifyAction(r.Context(), uid)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// Surrender concedes the game.
func (h *RiskHandler) Surrender(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	state, err := h.store.SurrenderAction(r.Context(), uid)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}
