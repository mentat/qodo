package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/mentat/qodo/api/agent"
	"github.com/mentat/qodo/api/chat"
	"github.com/mentat/qodo/api/handlers"
	"github.com/mentat/qodo/api/middleware"
	"github.com/mentat/qodo/api/services"
)

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

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

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	if err != nil {
		log.Fatalf("failed to initialize firebase app: %v", err)
	}
	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("failed to initialize firebase auth: %v", err)
	}

	fsClient, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("failed to initialize firestore: %v", err)
	}
	defer fsClient.Close()

	todoSvc := services.NewTodoService(fsClient)
	todoHandler := handlers.NewTodoHandlerWithService(todoSvc)
	authMw := middleware.NewAuthMiddleware(authClient)

	// Build Marvin. Any failure here is fatal — the agent is a product requirement.
	marvinCfg := agent.Config{
		ProjectID:   projectID,
		NewsAPIKey:  os.Getenv("NEWSAPI_API_KEY"),
		TodoService: todoSvc,
	}
	marvin, err := agent.New(ctx, marvinCfg)
	if err != nil {
		log.Fatalf("failed to initialize agent: %v", err)
	}
	screener, err := agent.NewScreener(ctx, agent.ScreenerConfig{ProjectID: projectID})
	if err != nil {
		log.Fatalf("failed to initialize screener: %v", err)
	}
	chatStore := chat.NewStore(fsClient)
	agentHandler := handlers.NewAgentHandler(marvin, screener, chatStore)

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
		r.Get("/", todoHandler.List)
		r.Post("/", todoHandler.Create)
		r.Post("/reorder", todoHandler.Reorder)
		r.Get("/{id}", todoHandler.Get)
		r.Put("/{id}", todoHandler.Update)
		r.Patch("/{id}", todoHandler.Patch)
		r.Delete("/{id}", todoHandler.Delete)
	})

	r.Route("/api/agent", func(r chi.Router) {
		r.Use(authMw.Verify)
		r.Post("/chat", agentHandler.Chat)
		r.Get("/history", agentHandler.History)
		r.Delete("/history", agentHandler.ClearHistory)
	})

	newsStatus := "DISABLED — set NEWSAPI_API_KEY"
	if marvinCfg.NewsAPIKey != "" {
		newsStatus = fmt.Sprintf("enabled (key ends …%s)", tail(marvinCfg.NewsAPIKey, 4))
	}
	log.Printf("starting server on :%s (marvin model=%s, news=%s, project=%s)",
		port, marvin.ModelName(), newsStatus, projectID)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
