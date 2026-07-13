.PHONY: help build test run-search run-cron run-api docker-up docker-down clean build-api test-api start-all

help:
	@echo "Available commands:"
	@echo "  make build        - Build both CLI commands and API"
	@echo "  make build-api    - Build API server"
	@echo "  make test         - Run tests"
	@echo "  make test-api     - Test API endpoints"
	@echo "  make run-search   - Run search command (requires KEYWORD variable)"
	@echo "  make run-cron     - Run cron command"
	@echo "  make run-api      - Run API server locally"
	@echo "  make docker-up    - Start all services (PostgreSQL + API + Frontend)"
	@echo "  make docker-down  - Stop Docker containers"
	@echo "  make start-all    - Quick start everything (recommended)"
	@echo "  make clean        - Clean build artifacts"
	@echo ""
	@echo "Example: make run-search KEYWORD=laptop"
	@echo ""
	@echo "Quick Start: make start-all"
	@echo ""
	@echo "Access Points:"
	@echo "  Frontend:  http://localhost:8092"
	@echo "  API:       http://localhost:8091/api/v1"
	@echo "  DB Admin:  http://localhost:8099"

build:
	@echo "Building search command..."
	go build -o bin/search ./cmd/search
	@echo "Building cron command..."
	go build -o bin/cron ./cmd/cron
	@echo "Building API server..."
	go build -o bin/api ./cmd/api
	@echo "Build complete!"

build-api:
	@echo "Building API server..."
	go build -o bin/api ./cmd/api
	@echo "API build complete!"

run-api: build-api
	@echo "Starting API server on port 8091..."
	./bin/api

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

test-api:
	@echo "Testing API endpoints..."
	./test_api.sh

run-search:
	@if [ -z "$(KEYWORD)" ]; then \
		echo "Error: KEYWORD variable is required"; \
		echo "Usage: make run-search KEYWORD=laptop"; \
		exit 1; \
	fi
	go run ./cmd/search -keyword="$(KEYWORD)" -verbose=$(VERBOSE)

run-cron:
	go run ./cmd/cron -verbose=$(VERBOSE) -output=$(OUTPUT)

docker-up:
	@echo "Starting PostgreSQL..."
	docker-compose up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 3
	@echo "PostgreSQL is ready!"

docker-down:
	@echo "Stopping Docker containers..."
	docker-compose down

docker-clean:
	@echo "Stopping and removing Docker containers and volumes..."
	docker-compose down -v

start-all:
	@echo "🚀 Starting complete Second-Hand Shop Scraper..."
	@./start.sh

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.txt
	rm -f *.coverprofile
	@echo "Clean complete!"

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Quick start - sets up everything
setup: docker-up deps
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Setup complete! You can now run: make run-search KEYWORD=laptop"
