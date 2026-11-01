package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dede999/tools_crud_go/internal/handler"
	"github.com/dede999/tools_crud_go/internal/notifier"
	"github.com/dede999/tools_crud_go/internal/repository"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	port := getEnv("PORT", "8080")
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgrespassword@localhost:5432/devices_db?sslmode=disable")

	log.Printf("Connecting to database at %s...", sanitizeDBURL(dbURL))
	db, err := sql.Open("pgx", dbURL)
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

	// Ensure schema exists (self-bootstrapping for local run and CI)
	if err := runMigrations(db); err != nil {
		log.Fatalf("Failed to apply schema migrations: %v", err)
	}

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
		Addr:         ":" + port,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("HTTP server listening on port %s", port)
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

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func sanitizeDBURL(dbURL string) string {
	// Simple display sanitizer to hide password in logs
	return "postgresql://.../"
}

func runMigrations(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS devices (
		id UUID PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		brand VARCHAR(255) NOT NULL,
		state VARCHAR(50) NOT NULL CHECK (state IN ('available', 'in-use', 'inactive')),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_devices_brand ON devices (brand);
	CREATE INDEX IF NOT EXISTS idx_devices_state ON devices (state);
	`
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("migration error: %w", err)
	}
	log.Println("Database schema verified")
	return nil
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

