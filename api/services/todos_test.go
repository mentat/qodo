package services_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/mentat/qodo/api/services"
)

// These tests talk to real Firestore. They're gated on the same
// GOOGLE_APPLICATION_CREDENTIALS + GOOGLE_CLOUD_PROJECT the API uses
// locally, so they align with how the app actually runs. Each test uses
// a unique userID so composite indexes apply and no production data is
// touched; cleanup deletes only that user's docs.

func newTestService(t *testing.T) (*services.TodoService, func()) {
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
	svc := services.NewTodoService(fs)
	return svc, func() { fs.Close() }
}

func uniqueUID(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("test-%s-%d", strings.ReplaceAll(t.Name(), "/", "-"), time.Now().UnixNano())
}

func cleanupUser(t *testing.T, svc *services.TodoService, uid string) {
	t.Helper()
	ctx := context.Background()
	todos, err := svc.List(ctx, uid, services.ListFilter{})
	if err != nil {
		t.Logf("cleanup list: %v", err)
		return
	}
	for _, td := range todos {
		_ = svc.Delete(ctx, uid, td.ID)
	}
}

func TestTodoService_CreateListGet(t *testing.T) {
	svc, closeFn := newTestService(t)
	defer closeFn()
	ctx := context.Background()
	uid := uniqueUID(t)
	defer cleanupUser(t, svc, uid)

	t1, err := svc.Create(ctx, uid, services.CreateInput{Title: "buy milk", Priority: "high"})
	if err != nil {
		t.Fatal(err)
	}
	if t1.ID == "" || t1.Position != 0 || t1.Priority != "high" || t1.Completed {
		t.Fatalf("bad first todo: %+v", t1)
	}
	t2, err := svc.Create(ctx, uid, services.CreateInput{Title: "walk dog"})
	if err != nil {
		t.Fatal(err)
	}
	if t2.Position != 1 || t2.Priority != "medium" {
		t.Fatalf("bad second todo: %+v", t2)
	}

	list, err := svc.List(ctx, uid, services.ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("list len = %d, want 2", len(list))
	}
	if list[0].ID != t1.ID || list[1].ID != t2.ID {
		t.Errorf("ordering wrong: %v / %v", list[0].ID, list[1].ID)
	}

	got, err := svc.Get(ctx, uid, t1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "buy milk" {
		t.Errorf("got: %+v", got)
	}
}

func TestTodoService_UserIsolation(t *testing.T) {
	svc, closeFn := newTestService(t)
	defer closeFn()
	ctx := context.Background()
	uid1 := uniqueUID(t) + "-a"
	uid2 := uniqueUID(t) + "-b"
	defer cleanupUser(t, svc, uid1)
	defer cleanupUser(t, svc, uid2)

	mine, _ := svc.Create(ctx, uid1, services.CreateInput{Title: "mine"})

	if _, err := svc.Get(ctx, uid2, mine.ID); !errors.Is(err, services.ErrNotFound) {
		t.Errorf("cross-user Get should be NotFound, got %v", err)
	}
	if err := svc.Delete(ctx, uid2, mine.ID); !errors.Is(err, services.ErrNotFound) {
		t.Errorf("cross-user Delete should be NotFound, got %v", err)
	}
}

func TestTodoService_PatchAndRejectProtectedFields(t *testing.T) {
	svc, closeFn := newTestService(t)
	defer closeFn()
	ctx := context.Background()
	uid := uniqueUID(t)
	defer cleanupUser(t, svc, uid)

	t1, _ := svc.Create(ctx, uid, services.CreateInput{Title: "original"})
	orig := t1.CreatedAt

	patched, err := svc.Patch(ctx, uid, t1.ID, map[string]any{
		"title":     "updated",
		"completed": true,
		"userId":    "hacker",
		"createdAt": time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if patched.Title != "updated" || !patched.Completed {
		t.Errorf("patch not applied: %+v", patched)
	}
	if patched.UserID != uid {
		t.Errorf("userId overwritten: %s", patched.UserID)
	}
	if !patched.CreatedAt.Equal(orig) {
		t.Errorf("createdAt overwritten: %v != %v", patched.CreatedAt, orig)
	}
	if !patched.UpdatedAt.After(orig) {
		t.Errorf("updatedAt not bumped")
	}
}

func TestTodoService_PatchValidatesPriority(t *testing.T) {
	svc, closeFn := newTestService(t)
	defer closeFn()
	ctx := context.Background()
	uid := uniqueUID(t)
	defer cleanupUser(t, svc, uid)
	t1, _ := svc.Create(ctx, uid, services.CreateInput{Title: "x"})
	_, err := svc.Patch(ctx, uid, t1.ID, map[string]any{"priority": "urgent"})
	if !errors.Is(err, services.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestTodoService_Delete(t *testing.T) {
	svc, closeFn := newTestService(t)
	defer closeFn()
	ctx := context.Background()
	uid := uniqueUID(t)
	defer cleanupUser(t, svc, uid)
	t1, _ := svc.Create(ctx, uid, services.CreateInput{Title: "x"})
	if err := svc.Delete(ctx, uid, t1.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, uid, t1.ID); !errors.Is(err, services.ErrNotFound) {
		t.Errorf("post-delete Get should be NotFound, got %v", err)
	}
}

func TestTodoService_ListFiltersAndSearch(t *testing.T) {
	svc, closeFn := newTestService(t)
	defer closeFn()
	ctx := context.Background()
	uid := uniqueUID(t)
	defer cleanupUser(t, svc, uid)
	_, _ = svc.Create(ctx, uid, services.CreateInput{Title: "groceries: milk and eggs", Priority: "high"})
	_, _ = svc.Create(ctx, uid, services.CreateInput{Title: "walk dog", Priority: "low"})
	done, _ := svc.Create(ctx, uid, services.CreateInput{Title: "inbox zero"})
	_, _ = svc.Patch(ctx, uid, done.ID, map[string]any{"completed": true})

	found, err := svc.List(ctx, uid, services.ListFilter{Search: "milk"})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || !strings.Contains(found[0].Title, "milk") {
		t.Errorf("search: %+v", found)
	}

	high, _ := svc.List(ctx, uid, services.ListFilter{Priority: "high"})
	if len(high) != 1 {
		t.Errorf("priority: %+v", high)
	}

	tru := true
	completed, _ := svc.List(ctx, uid, services.ListFilter{Completed: &tru})
	if len(completed) != 1 {
		t.Errorf("completed filter: %+v", completed)
	}
}

func TestTodoService_CreateRequiresUserAndTitle(t *testing.T) {
	svc, closeFn := newTestService(t)
	defer closeFn()
	ctx := context.Background()
	if _, err := svc.Create(ctx, "", services.CreateInput{Title: "x"}); !errors.Is(err, services.ErrUnauthenticated) {
		t.Errorf("empty user: %v", err)
	}
	uid := uniqueUID(t)
	defer cleanupUser(t, svc, uid)
	if _, err := svc.Create(ctx, uid, services.CreateInput{Title: ""}); !errors.Is(err, services.ErrInvalidInput) {
		t.Errorf("empty title: %v", err)
	}
}
