package chat_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/mentat/qodo/api/chat"
)

// These integration tests write to the real chatMessages collection with a
// unique per-test userID so the composite index applies.

func newStore(t *testing.T) (*chat.Store, func()) {
	t.Helper()
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		t.Skip("GOOGLE_APPLICATION_CREDENTIALS not set")
	}
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		project = "qodo-demo"
	}
	ctx := context.Background()
	fs, err := firestore.NewClient(ctx, project)
	if err != nil {
		t.Fatalf("firestore client: %v", err)
	}
	s := chat.NewStore(fs)
	return s, func() { fs.Close() }
}

func uid(t *testing.T) string {
	return fmt.Sprintf("test-%s-%d", strings.ReplaceAll(t.Name(), "/", "-"), time.Now().UnixNano())
}

func TestChatStore_AppendAndHistory(t *testing.T) {
	s, closeFn := newStore(t)
	defer closeFn()
	ctx := context.Background()
	u := uid(t)
	defer s.Clear(ctx, u)

	for i, text := range []string{"hello", "how are you", "goodbye"} {
		role := chat.RoleUser
		if i%2 == 1 {
			role = chat.RoleAssistant
		}
		if _, err := s.Append(ctx, chat.Message{UserID: u, Role: role, Content: text}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(5 * time.Millisecond) // ensure createdAt ordering
	}

	msgs, err := s.History(ctx, u, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 3 {
		t.Fatalf("len=%d", len(msgs))
	}
	// Oldest-first ordering
	if msgs[0].Content != "hello" || msgs[2].Content != "goodbye" {
		t.Errorf("ordering wrong: %+v", msgs)
	}
	if msgs[1].Role != chat.RoleAssistant {
		t.Errorf("role not persisted: %+v", msgs[1])
	}
}

func TestChatStore_HistoryLimit(t *testing.T) {
	s, closeFn := newStore(t)
	defer closeFn()
	ctx := context.Background()
	u := uid(t)
	defer s.Clear(ctx, u)
	for i := 0; i < 5; i++ {
		s.Append(ctx, chat.Message{UserID: u, Role: chat.RoleUser, Content: fmt.Sprintf("m%d", i)})
		time.Sleep(3 * time.Millisecond)
	}
	msgs, err := s.History(ctx, u, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("len=%d", len(msgs))
	}
	// The last 2 messages (newest) are returned, ordered oldest-first.
	if msgs[0].Content != "m3" || msgs[1].Content != "m4" {
		t.Errorf("unexpected: %+v", msgs)
	}
}

func TestChatStore_Clear(t *testing.T) {
	s, closeFn := newStore(t)
	defer closeFn()
	ctx := context.Background()
	u := uid(t)
	s.Append(ctx, chat.Message{UserID: u, Role: chat.RoleUser, Content: "x"})
	s.Append(ctx, chat.Message{UserID: u, Role: chat.RoleAssistant, Content: "y"})
	if err := s.Clear(ctx, u); err != nil {
		t.Fatal(err)
	}
	msgs, _ := s.History(ctx, u, 50)
	if len(msgs) != 0 {
		t.Errorf("history not cleared: %d", len(msgs))
	}
}

func TestChatStore_Append_Validation(t *testing.T) {
	s, closeFn := newStore(t)
	defer closeFn()
	ctx := context.Background()
	if _, err := s.Append(ctx, chat.Message{Role: chat.RoleUser, Content: "x"}); err == nil {
		t.Error("want error for empty userId")
	}
	if _, err := s.Append(ctx, chat.Message{UserID: "u"}); err == nil {
		t.Error("want error for empty role")
	}
}
