package handlers

import (
	"log"
	"net/http"

	"github.com/mentat/qodo/api/agent"
	"github.com/mentat/qodo/api/services/risk"
)

// RiskPubsubHandler runs an AI turn in response to a Cloud Pub/Sub push from
// the risk-turns topic. It mirrors PubsubHandler.EmailReply exactly:
// envelope decode → receipt claim → action → release on transient error.
//
// Unlike the email reply pipeline, it doesn't need a mail store — the AI
// worker (agent.RiskAI) does its own Firestore writes for each sub-step so the
// frontend can animate the turn live. After the AI finishes, if the next
// player is *also* AI, this handler re-publishes via the store so the next
// turn is queued as a fresh job.
type RiskPubsubHandler struct {
	store      *risk.Store
	ai         *agent.RiskAI
	receipts   receiptStore
	pushToken  string
	verifyOIDC func(ctx interface{ Done() <-chan struct{} }, token string) error
}

// RiskPubsubConfig wires a RiskPubsubHandler.
type RiskPubsubConfig struct {
	Store     *risk.Store
	AI        *agent.RiskAI
	Receipts  receiptStore
	PushToken string
	// VerifyOIDC has the same signature as PubsubConfig.VerifyOIDC; passing nil
	// disables the OIDC bearer check (token query-param still enforced if set).
}

// NewRiskPubsubHandler builds a handler.
func NewRiskPubsubHandler(cfg RiskPubsubConfig) *RiskPubsubHandler {
	return &RiskPubsubHandler{
		store:     cfg.Store,
		ai:        cfg.AI,
		receipts:  cfg.Receipts,
		pushToken: cfg.PushToken,
	}
}

// Turn handles POST /api/pubsub/risk-turn.
func (h *RiskPubsubHandler) Turn(w http.ResponseWriter, r *http.Request) {
	// Token gate (the OIDC variant is checked by the standalone PubsubHandler
	// when present; for the Risk push we only require the shared secret).
	if h.pushToken != "" && r.URL.Query().Get("token") != h.pushToken {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	env, ok := decodeEnvelope(r)
	if !ok {
		w.WriteHeader(http.StatusOK) // malformed → ack
		return
	}
	key := "risk-turn:" + env.Message.MessageID
	fresh, err := h.receipts.Claim(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "receipt")
		return
	}
	if !fresh {
		w.WriteHeader(http.StatusOK) // duplicate delivery
		return
	}

	attrs := env.Message.Attributes
	userID := attrs["userId"]
	aiPlayerID := risk.PlayerID(attrs["aiPlayerId"])
	if userID == "" || aiPlayerID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	st, err := h.ai.PlayTurn(r.Context(), userID, aiPlayerID)
	if err != nil {
		log.Printf("risk-turn: PlayTurn for %s/%s failed: %v", userID, aiPlayerID, err)
		_ = h.receipts.Release(r.Context(), key)
		writeError(w, http.StatusInternalServerError, "play")
		return
	}
	// If the next player is also AI, queue them up.
	if st.Status == risk.StatusPlaying {
		cur := currentPlayer(st)
		if cur != nil && cur.Kind == risk.KindAI {
			h.store.PublishAITurn(r.Context(), userID, st)
		}
	}
	w.WriteHeader(http.StatusOK)
}

func currentPlayer(s risk.State) *risk.Player {
	for i, p := range s.Players {
		if p.ID == s.Turn.CurrentPlayerID {
			return &s.Players[i]
		}
	}
	return nil
}
