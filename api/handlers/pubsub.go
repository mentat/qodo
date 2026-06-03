package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"hash/fnv"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mentat/qodo/api/seed"
	"github.com/mentat/qodo/api/services"
)

// ── Collaborator interfaces (real services satisfy these; tests fake them) ──

type mailStore interface {
	ListThread(ctx context.Context, userID, threadID string) ([]services.Email, error)
	CreateInbound(ctx context.Context, userID string, in services.InboundInput) (services.Email, error)
}

type replyGenerator interface {
	Reply(ctx context.Context, characterID string, thread []services.Email) (subject, body string, err error)
	Greeting(ctx context.Context, characterID string) (subject, body string, err error)
}

// receiptStore provides at-least-once dedup. Claim returns true when the key
// is newly claimed (proceed); Release lets a transient failure be retried.
type receiptStore interface {
	Claim(ctx context.Context, key string) (bool, error)
	Release(ctx context.Context, key string) error
}

type userLister interface {
	SeededUserIDs(ctx context.Context, limit int) ([]string, error)
}

type eventCreator interface {
	Create(ctx context.Context, userID string, in services.EventInput) (services.Event, error)
}

// dripMaxUsers caps the per-tick fan-out so the Vertex cost stays bounded.
const dripMaxUsers = 25

// PubsubHandler serves Cloud Pub/Sub push deliveries: the reply pipeline and
// the business-hours drip. These endpoints are NOT behind the Firebase auth
// middleware (the caller is Pub/Sub, not a user) — they verify a shared
// secret and, optionally, the push service account's OIDC token.
type PubsubHandler struct {
	mail       mailStore
	gen        replyGenerator
	receipts   receiptStore
	users      userLister
	events     eventCreator
	pushToken  string
	verifyOIDC func(ctx context.Context, token string) error
}

// PubsubConfig wires a PubsubHandler.
type PubsubConfig struct {
	Mail       mailStore
	Gen        replyGenerator
	Receipts   receiptStore
	Users      userLister
	Events     eventCreator
	PushToken  string
	VerifyOIDC func(ctx context.Context, token string) error
}

func NewPubsubHandler(cfg PubsubConfig) *PubsubHandler {
	return &PubsubHandler{
		mail: cfg.Mail, gen: cfg.Gen, receipts: cfg.Receipts, users: cfg.Users,
		events: cfg.Events, pushToken: cfg.PushToken, verifyOIDC: cfg.VerifyOIDC,
	}
}

// pushEnvelope is the body Pub/Sub POSTs to a push endpoint.
type pushEnvelope struct {
	Message struct {
		Data       string            `json:"data"` // base64
		MessageID  string            `json:"messageId"`
		Attributes map[string]string `json:"attributes"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

type pubsubPayload struct {
	Kind        string `json:"kind"`
	UserID      string `json:"userId"`
	ThreadID    string `json:"threadId"`
	EmailID     string `json:"emailId"`
	CharacterID string `json:"characterId"`
}

// EmailReply generates and stores an in-character reply to the user's mail.
func (h *PubsubHandler) EmailReply(w http.ResponseWriter, r *http.Request) {
	if !h.authOK(r) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	env, ok := decodeEnvelope(r)
	if !ok {
		w.WriteHeader(http.StatusOK) // malformed → ack to avoid poison redelivery
		return
	}
	key := "reply:" + env.Message.MessageID
	fresh, err := h.receipts.Claim(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "receipt") // transient → retry
		return
	}
	if !fresh {
		w.WriteHeader(http.StatusOK) // duplicate delivery
		return
	}

	p := payloadFromEnvelope(env)
	if p.UserID == "" || p.CharacterID == "" || p.ThreadID == "" {
		w.WriteHeader(http.StatusOK) // nothing actionable
		return
	}
	thread, err := h.mail.ListThread(r.Context(), p.UserID, p.ThreadID)
	if err != nil {
		_ = h.receipts.Release(r.Context(), key)
		writeError(w, http.StatusInternalServerError, "thread")
		return
	}
	subject, body, err := h.gen.Reply(r.Context(), p.CharacterID, thread)
	if err != nil {
		log.Printf("pubsub reply: generate failed: %v", err)
		_ = h.receipts.Release(r.Context(), key)
		writeError(w, http.StatusInternalServerError, "generate")
		return
	}
	ch, _ := seed.CharacterByID(p.CharacterID)
	if _, err := h.mail.CreateInbound(r.Context(), p.UserID, services.InboundInput{
		From:        ch.Email,
		FromName:    ch.Name,
		Subject:     subject,
		Body:        body,
		ThreadID:    p.ThreadID,
		CharacterID: p.CharacterID,
	}); err != nil {
		_ = h.receipts.Release(r.Context(), key)
		writeError(w, http.StatusInternalServerError, "write")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Drip is invoked by Cloud Scheduler (business hours). It sends each seeded
// user a fresh, unprompted in-character email and occasionally an event.
func (h *PubsubHandler) Drip(w http.ResponseWriter, r *http.Request) {
	if !h.authOK(r) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	env, ok := decodeEnvelope(r)
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}
	key := "drip:" + env.Message.MessageID
	fresh, err := h.receipts.Claim(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "receipt")
		return
	}
	if !fresh {
		w.WriteHeader(http.StatusOK)
		return
	}

	users, err := h.users.SeededUserIDs(r.Context(), dripMaxUsers)
	if err != nil {
		_ = h.receipts.Release(r.Context(), key)
		writeError(w, http.StatusInternalServerError, "users")
		return
	}
	pick := int(hash32(env.Message.MessageID))
	for i, uid := range users {
		ch := seed.Characters[(pick+i)%len(seed.Characters)]
		subject, body, err := h.gen.Greeting(r.Context(), ch.ID)
		if err != nil {
			log.Printf("pubsub drip: greeting failed for %s: %v", ch.ID, err)
			continue
		}
		if _, err := h.mail.CreateInbound(r.Context(), uid, services.InboundInput{
			From:        ch.Email,
			FromName:    ch.Name,
			Subject:     subject,
			Body:        body,
			CharacterID: ch.ID,
		}); err != nil {
			log.Printf("pubsub drip: inbound failed for %s: %v", uid, err)
			continue
		}
		// Occasionally drop a calendar invite from the same character.
		if h.events != nil && i%3 == 0 {
			start := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Hour)
			_, _ = h.events.Create(r.Context(), uid, services.EventInput{
				Title:       "Check-in with " + ch.Name,
				Description: "Auto-scheduled by " + ch.Name + " via the Synthwave OS drip.",
				Start:       start,
				End:         start.Add(30 * time.Minute),
				Color:       "#9B5DE5",
				CharacterID: ch.ID,
			})
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (h *PubsubHandler) authOK(r *http.Request) bool {
	if h.pushToken != "" && r.URL.Query().Get("token") != h.pushToken {
		return false
	}
	if h.verifyOIDC != nil {
		tok := bearerToken(r)
		if tok == "" {
			return false
		}
		if err := h.verifyOIDC(r.Context(), tok); err != nil {
			return false
		}
	}
	return true
}

func decodeEnvelope(r *http.Request) (pushEnvelope, bool) {
	var env pushEnvelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		return env, false
	}
	if env.Message.MessageID == "" {
		return env, false
	}
	return env, true
}

// payloadFromEnvelope prefers Pub/Sub attributes (always set by our
// publisher) and falls back to the base64 JSON data field.
func payloadFromEnvelope(env pushEnvelope) pubsubPayload {
	a := env.Message.Attributes
	p := pubsubPayload{
		Kind:        a["kind"],
		UserID:      a["userId"],
		ThreadID:    a["threadId"],
		EmailID:     a["emailId"],
		CharacterID: a["characterId"],
	}
	if p.UserID == "" && env.Message.Data != "" {
		if raw, err := base64.StdEncoding.DecodeString(env.Message.Data); err == nil {
			var d pubsubPayload
			if json.Unmarshal(raw, &d) == nil {
				return d
			}
		}
	}
	return p
}

func bearerToken(r *http.Request) string {
	parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		return parts[1]
	}
	return ""
}

func hash32(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

// ── Firestore-backed receipt store (production) ────────────────────────────

// FirestoreReceipts dedups Pub/Sub deliveries via a doc-per-key collection.
type FirestoreReceipts struct {
	fs         *firestore.Client
	collection string
}

// NewFirestoreReceipts returns a receipt store in the given collection.
func NewFirestoreReceipts(fs *firestore.Client, collection string) FirestoreReceipts {
	if collection == "" {
		collection = "pubsubReceipts"
	}
	return FirestoreReceipts{fs: fs, collection: collection}
}

// Claim creates the receipt doc; AlreadyExists means a duplicate delivery.
func (r FirestoreReceipts) Claim(ctx context.Context, key string) (bool, error) {
	_, err := r.fs.Collection(r.collection).Doc(key).Create(ctx, map[string]any{"at": time.Now().UTC()})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Release deletes a receipt so a transient failure can be retried.
func (r FirestoreReceipts) Release(ctx context.Context, key string) error {
	_, err := r.fs.Collection(r.collection).Doc(key).Delete(ctx)
	return err
}
