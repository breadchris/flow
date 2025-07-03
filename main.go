package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/breadchris/flow/coderunner"
	"github.com/breadchris/flow/config"
	"github.com/breadchris/flow/db"
	"github.com/breadchris/flow/deps"
	"github.com/breadchris/flow/slackbot"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()
	
	// Setup database
	database := db.NewClaudeDB(cfg.DSN)
	
	// Create dependencies
	factory := deps.NewDepsFactory(cfg)
	dependencies := factory.CreateDeps(database, cfg.ShareDir)
	
	// Setup main HTTP mux
	mux := http.NewServeMux()
	
	// Mount coderunner at /coderunner
	coderunnerMux := coderunner.New(dependencies)
	mux.Handle("/coderunner/", http.StripPrefix("/coderunner", coderunnerMux))
	
	// Create HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	
	// Create and start slack bot
	bot, err := slackbot.New(dependencies)
	if err != nil {
		log.Fatalf("Failed to create slack bot: %v", err)
	}
	
	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigCh
		slog.Info("Received shutdown signal")
		cancel()
		
		// Shutdown HTTP server
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Failed to shutdown HTTP server", "error", err)
		}
		
		// Stop slack bot
		bot.Stop()
	}()
	
	// Start HTTP server in background
	go func() {
		slog.Info("Starting HTTP server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()
	
	// Start the slack bot
	slog.Info("Starting Slack bot...")
	if err := bot.Start(ctx); err != nil {
		log.Fatalf("Failed to start slack bot: %v", err)
	}
}