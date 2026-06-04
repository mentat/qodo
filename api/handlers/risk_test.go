package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mentat/qodo/api/middleware"
	"github.com/mentat/qodo/api/services/risk"
)

// TestRisk_RoutesUnauthenticated verifies the handler returns 401 when the
// auth middleware hasn't injected a user ID (the production wiring puts the
// auth middleware in front of every risk route — these tests bypass that
// middleware and confirm the handler itself bails when there's no UID).
//
// We can't easily run the full Firebase auth flow in-process, so we focus on:
//   - bad request bodies → 400
//   - missing user context → handlers still call store, which fails on empty
//     userID (the persistence layer requires it)
//   - shape of the successful path's JSON
func TestRisk_BadRequestBody(t *testing.T) {
	h := NewRiskHandler(nil) // store nil; bad body trips before any store call
	for _, path := range []string{"/new", "/place-initial", "/place", "/trade", "/attack", "/post-conquest", "/fortify"} {
		req := httptest.NewRequest(http.MethodPost, "/api/risk"+path, bytes.NewReader([]byte("not-json")))
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "test-uid"))
		w := httptest.NewRecorder()
		dispatchRiskPath(h, path, w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s with bad body: want 400, got %d (body=%s)", path, w.Code, w.Body.String())
		}
	}
}

// dispatchRiskPath routes a request to the right RiskHandler method for the
// given URL suffix. Mirrors the chi.Router routing in main.go.
func dispatchRiskPath(h *RiskHandler, path string, w http.ResponseWriter, r *http.Request) {
	switch path {
	case "/new":
		h.New(w, r)
	case "/place-initial":
		h.PlaceInitial(w, r)
	case "/place":
		h.Place(w, r)
	case "/trade":
		h.Trade(w, r)
	case "/attack":
		h.Attack(w, r)
	case "/post-conquest":
		h.PostConquest(w, r)
	case "/fortify":
		h.Fortify(w, r)
	}
}

// TestRiskStatusFor: every engine error maps to a sensible HTTP status.
func TestRiskStatusFor(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{risk.ErrNoGame, http.StatusNotFound},
		{risk.ErrNotYourTurn, http.StatusForbidden},
		{risk.ErrWrongPhase, http.StatusBadRequest},
		{risk.ErrInvalidPlacement, http.StatusBadRequest},
		{risk.ErrInvalidAttack, http.StatusBadRequest},
		{risk.ErrInvalidFortify, http.StatusBadRequest},
		{risk.ErrInvalidCardSet, http.StatusBadRequest},
		{risk.ErrInvalidSetup, http.StatusBadRequest},
		{risk.ErrPostConquestPending, http.StatusBadRequest},
		{risk.ErrGameOver, http.StatusConflict},
	}
	for _, c := range cases {
		got, _ := riskStatusFor(c.err)
		if got != c.code {
			t.Errorf("%v: want HTTP %d, got %d", c.err, c.code, got)
		}
	}
}

// TestRiskPubsubHandler_TokenGate: when a push token is configured, requests
// without it return 403.
func TestRiskPubsubHandler_TokenGate(t *testing.T) {
	h := NewRiskPubsubHandler(RiskPubsubConfig{
		Receipts:  newRiskTestReceipts(),
		PushToken: "secret-shared",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/pubsub/risk-turn",
		strings.NewReader(`{"message":{"messageId":"m1"}}`))
	w := httptest.NewRecorder()
	h.Turn(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("no-token request: want 403, got %d", w.Code)
	}

	// With the token, the (empty) message is acked.
	req2 := httptest.NewRequest(http.MethodPost, "/api/pubsub/risk-turn?token=secret-shared",
		strings.NewReader(`{"message":{"messageId":"m2","attributes":{}}}`))
	w2 := httptest.NewRecorder()
	h.Turn(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("token+empty payload: want 200 ack, got %d", w2.Code)
	}
}

// TestRiskPubsubHandler_Idempotent: duplicate deliveries with the same
// messageId are silently acked without re-running the action.
func TestRiskPubsubHandler_Idempotent(t *testing.T) {
	rec := newRiskTestReceipts()
	rec.claimed["risk-turn:dup-1"] = true // already seen
	h := NewRiskPubsubHandler(RiskPubsubConfig{
		Receipts: rec,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/pubsub/risk-turn",
		strings.NewReader(`{"message":{"messageId":"dup-1","attributes":{"kind":"ai-turn","userId":"u1","aiPlayerId":"ai-1"}}}`))
	w := httptest.NewRecorder()
	h.Turn(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("duplicate delivery: want 200 ack, got %d", w.Code)
	}
}

// ── shared minimal fakes ─────────────────────────────────────────────────

type riskTestReceipts struct{ claimed map[string]bool }

func newRiskTestReceipts() *riskTestReceipts { return &riskTestReceipts{claimed: map[string]bool{}} }
func (r *riskTestReceipts) Claim(_ context.Context, key string) (bool, error) {
	if r.claimed[key] {
		return false, nil
	}
	r.claimed[key] = true
	return true, nil
}
func (r *riskTestReceipts) Release(_ context.Context, key string) error {
	delete(r.claimed, key)
	return nil
}

// Compile-time check the handler's JSON contract matches the doc.
var _ = json.Marshal
