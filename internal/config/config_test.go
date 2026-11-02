package config_test

import (
	"os"
	"testing"

	"github.com/dede999/tools_crud_go/internal/config"
)

func TestLoadConfig(t *testing.T) {
	// Clean env before each test
	t.Cleanup(func() {
		os.Unsetenv("PORT")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("APP_ENV")
	})

	t.Run("default to development values when env is empty", func(t *testing.T) {
		os.Unsetenv("PORT")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("APP_ENV")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("expected no error in development defaults, got %v", err)
		}
		if cfg.Port != "8080" {
			t.Errorf("expected default port 8080, got %s", cfg.Port)
		}
		if cfg.AppEnv != "development" {
			t.Errorf("expected default env development, got %s", cfg.AppEnv)
		}
		if cfg.DatabaseURL == "" {
			t.Errorf("expected development fallback database URL")
		}
	})

	t.Run("fail-fast when DATABASE_URL is missing in production", func(t *testing.T) {
		os.Setenv("APP_ENV", "production")
		os.Unsetenv("DATABASE_URL")

		_, err := config.Load()
		if err == nil {
			t.Fatal("expected error when DATABASE_URL is missing in production, got nil")
		}
	})

	t.Run("load valid custom configuration", func(t *testing.T) {
		os.Setenv("PORT", "9090")
		os.Setenv("DATABASE_URL", "postgres://user:pass@remote:5432/db")
		os.Setenv("APP_ENV", "staging")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Port != "9090" || cfg.AppEnv != "staging" {
			t.Errorf("unexpected config: %+v", cfg)
		}
	})

	t.Run("reject invalid port number", func(t *testing.T) {
		os.Setenv("PORT", "99999")
		_, err := config.Load()
		if err == nil {
			t.Fatal("expected error for invalid port 99999, got nil")
		}
	})
}

