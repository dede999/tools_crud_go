package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
)

type Config struct {
	Port        string
	DatabaseURL string
	AppEnv      string
}

// Load reads and validates configuration from environment variables.
// In non-development environments, critical variables like DATABASE_URL are strictly required (fail-fast).
func Load() (*Config, error) {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	} else {
		portNum, err := strconv.Atoi(port)
		if err != nil || portNum < 1 || portNum > 65535 {
			return nil, fmt.Errorf("invalid PORT value %q: must be between 1 and 65535", port)
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		if appEnv == "development" {
			dbURL = "postgres://postgres:postgrespassword@localhost:5432/devices_db?sslmode=disable"
		} else {
			return nil, errors.New("DATABASE_URL is required in non-development environments")
		}
	}

	parsedURL, err := url.Parse(dbURL)
	if err != nil || parsedURL.Scheme == "" {
		return nil, fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	return &Config{
		Port:        port,
		DatabaseURL: dbURL,
		AppEnv:      appEnv,
	}, nil
}

