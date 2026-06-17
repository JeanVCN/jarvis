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
	"github.com/jean/jarvis/backend/internal/database"
	"github.com/jean/jarvis/backend/internal/handlers"
	"github.com/jean/jarvis/backend/internal/services"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	llm := ai.NewOllamaProvider(cfg.OllamaURL, cfg.OllamaModel)
	chatService := services.NewChatService(llm)
	router := handlers.NewRouter(db, chatService, cfg.CORSAllowedOrigin)

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
