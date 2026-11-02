.PHONY: help build run test test-race test-e2e docker-up docker-down docker-logs clean

# Default target
.DEFAULT_GOAL := help

help: ## Show available Makefile targets
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the static API binary
	@echo "Building binary to bin/api..."
	@mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags="-w -s" -o bin/api ./cmd/api

run: ## Run the application locally
	go run cmd/api/main.go

test: ## Run unit tests across all packages
	go test -v ./...

test-race: ## Run unit tests with Go data race detector
	go test -v -race ./...

test-e2e: ## Run end-to-end API tests using Hurl against running server
	hurl --test --color tests/hurl/devices.hurl

docker-up: ## Build and start services via Docker Compose in background
	docker compose up --build -d

docker-down: ## Stop and remove all Docker Compose containers and networks
	docker compose down

docker-logs: ## Follow Docker Compose container logs
	docker compose logs -f

clean: ## Clean build artifacts
	rm -rf bin/

