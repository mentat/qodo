// Backfill the FullText field on every todo in Firestore. Safe to re-run:
// it only updates docs whose current FullText doesn't match the recomputed
// value, so unchanged records aren't rewritten.
//
// Usage:
//
//	GOOGLE_APPLICATION_CREDENTIALS=../../../service-account.json \
//	GOOGLE_CLOUD_PROJECT=qodo-demo \
//	go run ./cmd/backfill-fulltext
package main

import (
	"context"
	"log"
	"os"
	"reflect"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/mentat/qodo/api/search"
	"github.com/mentat/qodo/api/services"
)

func main() {
	ctx := context.Background()
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		project = "qodo-demo"
	}
	fs, err := firestore.NewClient(ctx, project)
	if err != nil {
		log.Fatalf("firestore: %v", err)
	}
	defer fs.Close()

	iter := fs.Collection("todos").Documents(ctx)
	defer iter.Stop()
	var updated, skipped, total int
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("iter: %v", err)
		}
		total++
		var t services.Todo
		if err := doc.DataTo(&t); err != nil {
			log.Printf("skip %s: decode: %v", doc.Ref.ID, err)
			continue
		}
		want := search.Build(t.Title, t.Description, t.Category)
		if reflect.DeepEqual(t.FullText, want) {
			skipped++
			continue
		}
		if _, err := doc.Ref.Update(ctx, []firestore.Update{{Path: "fullText", Value: want}}); err != nil {
			log.Printf("update %s: %v", doc.Ref.ID, err)
			continue
		}
		updated++
		log.Printf("[%s] %q -> %v", doc.Ref.ID, t.Title, want)
	}
	log.Printf("done: total=%d updated=%d skipped=%d", total, updated, skipped)
}
