package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/mentat/qodo/api/handlers"
	"github.com/mentat/qodo/api/middleware"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = "qodo-demo"
	}

	ctx := context.Background()

	// Initialize Firebase Admin SDK
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	if err != nil {
		log.Fatalf("failed to initialize firebase app: %v", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("failed to initialize firebase auth: %v", err)
	}

	// Initialize Firestore
	fsClient, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("failed to initialize firestore: %v", err)
	}
	defer fsClient.Close()

	h := handlers.NewTodoHandler(fsClient)
	authMw := middleware.NewAuthMiddleware(authClient)

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/todos", func(r chi.Router) {
		r.Use(authMw.Verify)
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Post("/reorder", h.Reorder)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Patch("/{id}", h.Patch)
		r.Delete("/{id}", h.Delete)
	})

	log.Printf("starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
