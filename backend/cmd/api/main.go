package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jean/jarvis/backend/internal/ai"
	"github.com/jean/jarvis/backend/internal/config"
	"github.com/jean/jarvis/backend/internal/conversation"
	"github.com/jean/jarvis/backend/internal/database"
	"github.com/jean/jarvis/backend/internal/handlers"
	"github.com/jean/jarvis/backend/internal/services"
	"github.com/jean/jarvis/backend/internal/tools"
	"github.com/jean/jarvis/backend/internal/voice"
	"github.com/jean/jarvis/backend/migrations"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()
	lease, err := db.Pool().Acquire(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer lease.Release()
	var exclusive bool
	if err := lease.QueryRow(ctx, "SELECT pg_try_advisory_lock(74206114)").Scan(&exclusive); err != nil || !exclusive {
		log.Fatal("another Jarvis Core owns this database")
	}
	if err := migrations.Apply(ctx, db.Pool()); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	toolRegistry := tools.NewRegistry()
	mustRegisterTool(toolRegistry, tools.NewTimeTool())
	mustRegisterTool(toolRegistry, tools.NewCalculatorTool())
	if os.Getenv("JARVIS_ALLOW_SYSTEM_TOOLS") == "true" {
		mustRegisterTool(toolRegistry, tools.NewDefaultSystemTool())
	}
	toolService := services.NewToolService(toolRegistry)

	llm := ai.NewOllamaProvider(cfg.OllamaURL, cfg.OllamaModel)
	chatService := services.NewChatService(llm, toolService)
	voiceRecognizer := newSpeechRecognizer(cfg)
	voiceProvider := voice.NewCommandVoiceProvider(cfg.VoiceTTSCommand, voice.ParseArgs(cfg.VoiceTTSArgs), cfg.VoiceTTSVoice)
	voiceService := services.NewVoiceService(voiceRecognizer, voiceProvider, chatService)

	router := handlers.NewRouter(db, chatService, toolService, voiceService, cfg.CORSAllowedOrigin)

	conversation.Register(router, conversation.Store{Pool: db.Pool()}, toolService)

	server := &http.Server{
		Addr:              cfg.HTTPAddress(),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("jarvis backend listening on %s", cfg.HTTPAddress())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown failed: %v", err)
	}
}

func mustRegisterTool(registry *tools.Registry, tool tools.Tool) {
	if err := registry.Register(tool); err != nil {
		log.Fatalf("tool registration failed: %v", err)
	}
}

func newSpeechRecognizer(cfg config.Config) voice.SpeechRecognizer {
	if cfg.VoiceSTTProvider == "http" {
		return voice.NewHTTPSpeechRecognizer(cfg.VoiceSTTURL, cfg.VoiceSTTStatusURL, cfg.VoiceSTTLanguage)
	}
	return voice.NewCommandSpeechRecognizer(cfg.VoiceSTTCommand, voice.ParseArgs(cfg.VoiceSTTArgs), cfg.VoiceSTTModel, cfg.VoiceSTTLanguage)
}
