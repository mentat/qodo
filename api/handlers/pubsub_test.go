package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mentat/qodo/api/services"
)

// ── fakes ──────────────────────────────────────────────────────────────────

type fakeMail struct {
	thread  []services.Email
	created []services.InboundInput
}

func (f *fakeMail) ListThread(ctx context.Context, userID, threadID string) ([]services.Email, error) {
	return f.thread, nil
}
func (f *fakeMail) CreateInbound(ctx context.Context, userID string, in services.InboundInput) (services.Email, error) {
	f.created = append(f.created, in)
	return services.Email{ID: "new", UserID: userID}, nil
}

type fakeGen struct {
	calls int
	fail  bool
}

func (g *fakeGen) Reply(ctx context.Context, characterID string, thread []services.Email) (string, string, error) {
	g.calls++
	if g.fail {
		return "", "", fmt.Errorf("generate boom")
	}
	return "Re: test", "in-character reply", nil
}
func (g *fakeGen) Greeting(ctx context.Context, characterID string) (string, string, error) {
	g.calls++
	if g.fail {
		return "", "", fmt.Errorf("greeting boom")
	}
	return "a thought", "unprompted note", nil
}

type fakeReceipts struct {
	claimed map[string]bool
}

func newFakeReceipts() *fakeReceipts { return &fakeReceipts{claimed: map[string]bool{}} }
func (r *fakeReceipts) Claim(ctx context.Context, key string) (bool, error) {
	if r.claimed[key] {
		return false, nil
	}
	r.claimed[key] = true
	return true, nil
}
func (r *fakeReceipts) Release(ctx context.Context, key string) error {
	delete(r.claimed, key)
	return nil
}

type fakeUsers struct{ ids []string }

func (u fakeUsers) SeededUserIDs(ctx context.Context, limit int) ([]string, error) {
	return u.ids, nil
}

func newHandler(mail *fakeMail, gen *fakeGen, rcpt *fakeReceipts, users fakeUsers) *PubsubHandler {
	return NewPubsubHandler(PubsubConfig{
		Mail: mail, Gen: gen, Receipts: rcpt, Users: users, Events: nil,
		PushToken: "secret", VerifyOIDC: nil,
	})
}

func pushReq(path, token, messageID string, attrs map[string]string) *http.Request {
	env := map[string]any{"message": map[string]any{"messageId": messageID, "attributes": attrs}}
	body, _ := json.Marshal(env)
	url := path
	if token != "" {
		url += "?token=" + token
	}
	return httptest.NewRequest(http.MethodPost, url, strings.NewReader(string(body)))
}

func replyAttrs() map[string]string {
	return map[string]string{"kind": "reply", "userId": "u1", "threadId": "t1", "characterId": "dot-matrix"}
}

// ── tests ────────────────────────────────────────────────────────────────

func TestEmailReply_Forbidden(t *testing.T) {
	h := newHandler(&fakeMail{}, &fakeGen{}, newFakeReceipts(), fakeUsers{})
	rec := httptest.NewRecorder()
	h.EmailReply(rec, pushReq("/api/pubsub/email-reply", "", "m1", replyAttrs())) // no token
	if rec.Code != http.StatusForbidden {
		t.Errorf("missing token: want 403, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.EmailReply(rec, pushReq("/api/pubsub/email-reply", "wrong", "m1", replyAttrs()))
	if rec.Code != http.StatusForbidden {
		t.Errorf("wrong token: want 403, got %d", rec.Code)
	}
}

func TestEmailReply_GeneratesOnceAndWrites(t *testing.T) {
	mail := &fakeMail{thread: []services.Email{{FromName: "You", Subject: "hi", Body: "hello"}}}
	gen := &fakeGen{}
	h := newHandler(mail, gen, newFakeReceipts(), fakeUsers{})
	rec := httptest.NewRecorder()
	h.EmailReply(rec, pushReq("/api/pubsub/email-reply", "secret", "m1", replyAttrs()))
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if gen.calls != 1 {
		t.Errorf("gen calls = %d, want 1", gen.calls)
	}
	if len(mail.created) != 1 {
		t.Fatalf("created = %d, want 1", len(mail.created))
	}
	got := mail.created[0]
	if got.Subject != "Re: test" || got.CharacterID != "dot-matrix" || got.ThreadID != "t1" {
		t.Errorf("inbound wrong: %+v", got)
	}
	if got.FromName != "Dot Matrix" {
		t.Errorf("From resolved from seed: %+v", got)
	}
}

func TestEmailReply_DuplicateNoSecondWrite(t *testing.T) {
	mail := &fakeMail{}
	gen := &fakeGen{}
	rcpt := newFakeReceipts()
	h := newHandler(mail, gen, rcpt, fakeUsers{})

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		h.EmailReply(rec, pushReq("/api/pubsub/email-reply", "secret", "dup", replyAttrs()))
		if rec.Code != http.StatusOK {
			t.Fatalf("call %d: want 200, got %d", i, rec.Code)
		}
	}
	if gen.calls != 1 {
		t.Errorf("duplicate delivery generated twice: calls=%d", gen.calls)
	}
	if len(mail.created) != 1 {
		t.Errorf("duplicate delivery wrote twice: created=%d", len(mail.created))
	}
}

func TestEmailReply_MalformedAcks(t *testing.T) {
	gen := &fakeGen{}
	h := newHandler(&fakeMail{}, gen, newFakeReceipts(), fakeUsers{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/pubsub/email-reply?token=secret", strings.NewReader("{}"))
	h.EmailReply(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("malformed should ack 200, got %d", rec.Code)
	}
	if gen.calls != 0 {
		t.Errorf("malformed should not generate, calls=%d", gen.calls)
	}
}

func TestEmailReply_GenerateFailureReleasesForRetry(t *testing.T) {
	mail := &fakeMail{}
	gen := &fakeGen{fail: true}
	rcpt := newFakeReceipts()
	h := newHandler(mail, gen, rcpt, fakeUsers{})

	rec := httptest.NewRecorder()
	h.EmailReply(rec, pushReq("/api/pubsub/email-reply", "secret", "retry", replyAttrs()))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("generate failure should 500, got %d", rec.Code)
	}
	// The claim must have been released so a redelivery can retry.
	if rcpt.claimed["reply:retry"] {
		t.Error("claim not released after failure")
	}
	// Retry now succeeds.
	gen.fail = false
	rec = httptest.NewRecorder()
	h.EmailReply(rec, pushReq("/api/pubsub/email-reply", "secret", "retry", replyAttrs()))
	if rec.Code != http.StatusOK || len(mail.created) != 1 {
		t.Errorf("retry should succeed: code=%d created=%d", rec.Code, len(mail.created))
	}
}

func TestDrip_OneEmailPerSeededUser(t *testing.T) {
	mail := &fakeMail{}
	gen := &fakeGen{}
	h := newHandler(mail, gen, newFakeReceipts(), fakeUsers{ids: []string{"u1", "u2", "u3"}})
	rec := httptest.NewRecorder()
	h.Drip(rec, pushReq("/api/pubsub/drip", "secret", "d1", map[string]string{"kind": "drip"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if len(mail.created) != 3 {
		t.Errorf("drip should write one email per user: got %d", len(mail.created))
	}
	if gen.calls != 3 {
		t.Errorf("drip greeting calls = %d, want 3", gen.calls)
	}
}

func TestDrip_DuplicateSkips(t *testing.T) {
	mail := &fakeMail{}
	gen := &fakeGen{}
	rcpt := newFakeReceipts()
	h := newHandler(mail, gen, rcpt, fakeUsers{ids: []string{"u1"}})
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		h.Drip(rec, pushReq("/api/pubsub/drip", "secret", "same", map[string]string{"kind": "drip"}))
		if rec.Code != http.StatusOK {
			t.Fatalf("call %d: want 200, got %d", i, rec.Code)
		}
	}
	if len(mail.created) != 1 {
		t.Errorf("duplicate drip wrote twice: %d", len(mail.created))
	}
}
