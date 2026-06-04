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

	"google.golang.org/api/idtoken"

	"github.com/mentat/qodo/api/agent"
	"github.com/mentat/qodo/api/chat"
	pubsubclient "github.com/mentat/qodo/api/clients/pubsub"
	ttsclient "github.com/mentat/qodo/api/clients/tts"
	"github.com/mentat/qodo/api/handlers"
	"github.com/mentat/qodo/api/middleware"
	"github.com/mentat/qodo/api/services"
	"github.com/mentat/qodo/api/services/risk"
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

	// Suite domains: mail, calendar, contacts, notes, plus mocked weather +
	// static radio. Each mirrors the todo service/handler pattern.
	emailSvc := services.NewEmailService(fsClient)
	// Attach a Cloud Pub/Sub publisher so sending mail to a character kicks
	// off an async in-character reply. Best-effort: without creds/topic the
	// server still boots and simply skips auto-replies.
	if pub, perr := pubsubclient.NewPublisher(ctx, projectID, "email-replies"); perr != nil {
		log.Printf("pubsub publisher disabled: %v", perr)
	} else {
		emailSvc = emailSvc.WithPublisher(pub)
		defer pub.Close()
		log.Printf("pubsub publisher ready (topic=email-replies)")
	}
	eventSvc := services.NewEventService(fsClient)
	contactSvc := services.NewContactService(fsClient)
	noteSvc := services.NewNoteService(fsClient)

	emailHandler := handlers.NewEmailHandlerWithService(emailSvc)
	eventHandler := handlers.NewEventHandlerWithService(eventSvc)
	contactHandler := handlers.NewContactHandlerWithService(contactSvc)
	noteHandler := handlers.NewNoteHandlerWithService(noteSvc)
	weatherHandler := handlers.NewWeatherHandler()
	radioHandler := handlers.NewRadioHandler()
	calendarHandler := handlers.NewCalendarHandler(eventSvc, todoSvc)

	seedSvc := services.NewSeedService(fsClient, emailSvc, eventSvc, contactSvc, noteSvc)
	demoHandler := handlers.NewDemoHandler(seedSvc)

	// Risk: world-conquest game. The Store wraps the Firestore-backed
	// Persistence layer and (optionally) a Pub/Sub publisher so AI turns run
	// asynchronously on the backend — same pattern as the email-replies
	// pipeline above. The AI worker reads/writes the same Persistence and
	// streams sub-step updates to the client via onSnapshot.
	riskPersist := risk.NewPersistence(fsClient)
	riskStore := risk.NewStore(riskPersist)
	if pub, perr := pubsubclient.NewPublisher(ctx, projectID, "risk-turns"); perr != nil {
		log.Printf("risk pubsub publisher disabled: %v", perr)
	} else {
		riskStore = riskStore.WithPublisher(pub)
		defer pub.Close()
		log.Printf("risk pubsub publisher ready (topic=risk-turns)")
	}
	riskAI := agent.NewRiskAI(riskStore)
	riskHandler := handlers.NewRiskHandler(riskStore)
	var verifyPubsubOIDC func(context.Context, string) error
	if aud := os.Getenv("PUBSUB_PUSH_AUDIENCE"); aud != "" {
		verifyPubsubOIDC = func(c context.Context, tok string) error {
			_, e := idtoken.Validate(c, tok, aud)
			return e
		}
	}
	riskPubsubHandler := handlers.NewRiskPubsubHandler(handlers.RiskPubsubConfig{
		Store:      riskStore,
		AI:         riskAI,
		Receipts:   handlers.NewFirestoreReceipts(fsClient, "pubsubReceipts"),
		PushToken:  os.Getenv("PUBSUB_PUSH_TOKEN"),
		VerifyOIDC: verifyPubsubOIDC,
	})

	// Build Marvin. Any failure here is fatal — the agent is a product requirement.
	marvinCfg := agent.Config{
		ProjectID:      projectID,
		NewsAPIKey:     os.Getenv("NEWSAPI_API_KEY"),
		TodoService:    todoSvc,
		EmailService:   emailSvc,
		EventService:   eventSvc,
		ContactService: contactSvc,
		NoteService:    noteSvc,
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

	// Text-to-speech for Marvin's voice replies. Best-effort: if creds aren't
	// available locally the handler just omits audio from chat responses.
	var marvinTTS handlers.TTSSynthesizer
	if t, terr := ttsclient.New(ctx); terr != nil {
		log.Printf("tts disabled: %v", terr)
	} else {
		marvinTTS = t
		defer t.Close()
		log.Printf("tts ready (voice=%s)", ttsclient.DefaultVoiceName)
	}
	agentHandler := handlers.NewAgentHandler(marvin, screener, chatStore, marvinTTS)

	// Reply worker for the async email pipeline. Best-effort: if Vertex init
	// fails we just don't register the Pub/Sub push routes.
	var pubsubHandler *handlers.PubsubHandler
	if replyAgent, rerr := agent.NewReplyAgent(ctx, agent.ReplyConfig{ProjectID: projectID}); rerr != nil {
		log.Printf("reply agent disabled: %v", rerr)
	} else {
		pubsubHandler = handlers.NewPubsubHandler(handlers.PubsubConfig{
			Mail:       emailSvc,
			Gen:        replyAgent,
			Receipts:   handlers.NewFirestoreReceipts(fsClient, "pubsubReceipts"),
			Users:      seedSvc,
			Events:     eventSvc,
			PushToken:  os.Getenv("PUBSUB_PUSH_TOKEN"),
			VerifyOIDC: verifyPubsubOIDC,
		})
	}

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

	// Public radio stream proxy — the <audio> element can't send an auth
	// header, and it serves only fixed, non-sensitive tracks.
	r.Get("/api/radio/stream", radioHandler.Stream)

	r.Route("/api/todos", func(r chi.Router) {
		r.Use(authMw.Verify)
		r.Get("/", todoHandler.List)
		r.Post("/", todoHandler.Create)
		r.Post("/reorder", todoHandler.Reorder)
		r.Get("/search", todoHandler.Search)
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

	r.Route("/api/emails", func(r chi.Router) {
		r.Use(authMw.Verify)
		r.Get("/", emailHandler.List)
		r.Post("/", emailHandler.Send)
		r.Get("/threads/{threadId}", emailHandler.Thread)
		r.Get("/{id}", emailHandler.Get)
		r.Post("/{id}/read", emailHandler.MarkRead)
		r.Post("/{id}/star", emailHandler.Star)
		r.Delete("/{id}", emailHandler.Delete)
	})

	r.Route("/api/events", func(r chi.Router) {
		r.Use(authMw.Verify)
		r.Get("/", eventHandler.List)
		r.Post("/", eventHandler.Create)
		r.Get("/{id}", eventHandler.Get)
		r.Put("/{id}", eventHandler.Update)
		r.Post("/{id}/move", eventHandler.Move)
		r.Delete("/{id}", eventHandler.Delete)
	})

	r.Route("/api/contacts", func(r chi.Router) {
		r.Use(authMw.Verify)
		r.Get("/", contactHandler.List)
		r.Post("/", contactHandler.Create)
		r.Get("/search", contactHandler.Search)
		r.Get("/{id}", contactHandler.Get)
		r.Patch("/{id}", contactHandler.Patch)
		r.Delete("/{id}", contactHandler.Delete)
	})

	r.Route("/api/notes", func(r chi.Router) {
		r.Use(authMw.Verify)
		r.Get("/", noteHandler.List)
		r.Post("/", noteHandler.Create)
		r.Get("/search", noteHandler.Search)
		r.Get("/{id}", noteHandler.Get)
		r.Put("/{id}", noteHandler.Update)
		r.Delete("/{id}", noteHandler.Delete)
	})

	r.Route("/api/risk", func(r chi.Router) {
		r.Use(authMw.Verify)
		r.Get("/", riskHandler.Get)
		r.Post("/new", riskHandler.New)
		r.Get("/stats", riskHandler.Stats)
		r.Post("/place-initial", riskHandler.PlaceInitial)
		r.Post("/place", riskHandler.Place)
		r.Post("/trade", riskHandler.Trade)
		r.Post("/attack", riskHandler.Attack)
		r.Post("/post-conquest", riskHandler.PostConquest)
		r.Post("/fortify", riskHandler.Fortify)
		r.Post("/end-phase", riskHandler.EndPhase)
		r.Post("/skip-fortify", riskHandler.SkipFortify)
		r.Post("/surrender", riskHandler.Surrender)
	})

	r.Route("/api/demo", func(r chi.Router) {
		r.Use(authMw.Verify)
		r.Post("/seed", demoHandler.Seed)
		r.Post("/reset", demoHandler.Reset)
	})

	r.Group(func(r chi.Router) {
		r.Use(authMw.Verify)
		r.Get("/api/weather", weatherHandler.Forecast)
		r.Get("/api/radio/tracks", radioHandler.Tracks)
		r.Get("/api/calendar/agenda", calendarHandler.Agenda)
	})

	// Pub/Sub push endpoints — NOT behind Firebase auth (the caller is
	// Pub/Sub). They verify a shared secret + optional OIDC token in-handler.
	if pubsubHandler != nil {
		r.Post("/api/pubsub/email-reply", pubsubHandler.EmailReply)
		r.Post("/api/pubsub/drip", pubsubHandler.Drip)
	}
	r.Post("/api/pubsub/risk-turn", riskPubsubHandler.Turn)

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
