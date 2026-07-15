.PHONY: help build test run-search run-cron run-api docker-up docker-down docker-clean clean build-api deps setup

BACKEND_DIR := src/backend

help:
	@echo "Available commands:"
	@echo "  make build        - Build search, cron, and API binaries"
	@echo "  make build-api    - Build API server binary"
	@echo "  make test         - Run tests (scoped to src/..., see CLAUDE.md)"
	@echo "  make run-search   - Run search command (requires KEYWORD variable)"
	@echo "  make run-cron     - Run cron command"
	@echo "  make run-api      - Run API server locally"
	@echo "  make docker-up    - Start PostgreSQL + API + Frontend via Docker Compose"
	@echo "  make docker-down  - Stop Docker containers"
	@echo "  make docker-clean - Stop containers and remove volumes"
	@echo "  make setup        - Quick start: docker-up + deps"
	@echo "  make clean        - Clean build artifacts"
	@echo ""
	@echo "Example: make run-search KEYWORD=laptop"
	@echo ""
	@echo "Quick Start: make setup"
	@echo ""
	@echo "Access Points:"
	@echo "  Frontend:  http://localhost:8092"
	@echo "  API:       http://localhost:8091/api/v1"
	@echo "  DB Admin:  http://localhost:8099"

build:
	@echo "Building search command..."
	go build -o bin/search ./src/backend/cmd/search
	@echo "Building cron command..."
	go build -o bin/cron ./src/backend/cmd/cron
	@echo "Building API server..."
	go build -o bin/api ./src/backend/cmd/api
	@echo "Build complete!"

build-api:
	@echo "Building API server..."
	go build -o bin/api ./src/backend/cmd/api
	@echo "API build complete!"

# api/search/cron all hardcode "migrations" relative to the process's
# working directory, so they must run with CWD=src/backend (see CLAUDE.md)
# - hence `cd $(BACKEND_DIR) &&` below. Their "-config" default
# ("config.json") also only resolves from CWD=src/backend if pointed at
# config/config.json explicitly - the bare default is only correct inside
# the Docker images, which flatten config.json into the runtime root.
run-api: build-api
	@echo "Starting API server on port 8091..."
	cd $(BACKEND_DIR) && $(CURDIR)/bin/api -config=config/config.json

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./src/...

run-search:
	@if [ -z "$(KEYWORD)" ]; then \
		echo "Error: KEYWORD variable is required"; \
		echo "Usage: make run-search KEYWORD=laptop"; \
		exit 1; \
	fi
	cd $(BACKEND_DIR) && go run ./cmd/search -config=config/config.json -keyword="$(KEYWORD)" $(if $(VERBOSE),-verbose=$(VERBOSE))

run-cron:
	cd $(BACKEND_DIR) && go run ./cmd/cron -config=config/config.json $(if $(VERBOSE),-verbose=$(VERBOSE)) $(if $(OUTPUT),-output=$(OUTPUT))

docker-up:
	@echo "Starting PostgreSQL, API, and frontend via Docker Compose..."
	docker-compose up -d --build
	@echo "Waiting for services to be ready..."
	@sleep 3
	@echo "Up! Frontend: http://localhost:8092  API: http://localhost:8091/api/v1  DB Admin: http://localhost:8099"

docker-down:
	@echo "Stopping Docker containers..."
	docker-compose down

docker-clean:
	@echo "Stopping and removing Docker containers and volumes..."
	docker-compose down -v

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
