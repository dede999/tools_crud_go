package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dede999/tools_crud_go/internal/config"
	"github.com/dede999/tools_crud_go/internal/database"
	"github.com/dede999/tools_crud_go/internal/handler"
	"github.com/dede999/tools_crud_go/internal/notifier"
	"github.com/dede999/tools_crud_go/internal/repository"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load application configuration: %v", err)
	}

	log.Printf("Starting application in [%s] mode on port %s...", cfg.AppEnv, cfg.Port)
	log.Printf("Connecting to database at %s...", sanitizeDBURL(cfg.DatabaseURL))

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection established successfully")

	// Apply embedded migrations (single source of truth in migrations/*.sql)
	if err := database.RunMigrations(ctx, db); err != nil {
		log.Fatalf("Failed to apply schema migrations: %v", err)
	}
	log.Println("Database schema migrations verified successfully")

	repo := repository.NewPostgresRepository(db)
	eventBroker := notifier.NewBroker()
	deviceHandler := handler.NewDeviceHandler(repo, eventBroker)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	deviceHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
		// Note: WriteTimeout is intentionally not set globally so that
		// long-lived SSE streaming connections are not prematurely terminated.
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("HTTP server listening on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-stopChan
	log.Println("Shutting down HTTP server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	log.Println("Server gracefully stopped")
}

func sanitizeDBURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "postgres://[invalid-database-url]"
	}
	return parsed.Redacted()
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}
