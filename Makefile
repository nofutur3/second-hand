.PHONY: help build test lint run-search run-cron run-api up down docker-clean frontend-dev clean build-api deps setup

# --no-deps: build/lint/deps don't touch the database, so don't require
# postgres to be reachable just to run them. Targets that do need the
# database (test, run-*) use BACKEND_RUN_DB instead, which lets compose
# start postgres via backend-dev's depends_on.
BACKEND_RUN := docker compose run --rm --no-deps backend-dev
BACKEND_RUN_DB := docker compose run --rm backend-dev

help:
	@echo "Available commands:"
	@echo "  make build        - Build search, cron, and API binaries (Docker)"
	@echo "  make build-api    - Build API server binary (Docker)"
	@echo "  make test         - Run tests (scoped to src/..., see CLAUDE.md)"
	@echo "  make lint         - Check gofmt/goimports formatting (scoped to src/..., see CLAUDE.md)"
	@echo "  make run-search   - Run search command (requires KEYWORD variable)"
	@echo "  make run-cron     - Run cron command"
	@echo "  make run-api      - Run API server locally"
	@echo "  make up           - Start PostgreSQL + API + Frontend via Docker Compose"
	@echo "  make down         - Stop Docker containers"
	@echo "  make docker-clean - Stop containers and remove volumes"
	@echo "  make frontend-dev - Hot-reloading frontend dev server (Docker, no local npm needed)"
	@echo "  make setup        - Quick start: up + deps"
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
	$(BACKEND_RUN) sh -c 'cd src/backend && go build -o ../../bin/search ./cmd/search'
	@echo "Building cron command..."
	$(BACKEND_RUN) sh -c 'cd src/backend && go build -o ../../bin/cron ./cmd/cron'
	@echo "Building API server..."
	$(BACKEND_RUN) sh -c 'cd src/backend && go build -o ../../bin/api ./cmd/api'
	@echo "Build complete!"

build-api:
	@echo "Building API server..."
	$(BACKEND_RUN) sh -c 'cd src/backend && go build -o ../../bin/api ./cmd/api'
	@echo "API build complete!"

test:
	@echo "Running tests..."
	$(BACKEND_RUN_DB) sh -c 'cd src/backend && go vet ./... && go test -v -race -coverprofile=../../coverage.txt -covermode=atomic ./...'

lint:
	@echo "Checking gofmt/goimports formatting..."
	$(BACKEND_RUN) sh -c '\
		files=$$(find src -name "*.go"); \
		bad=$$(gofmt -l $$files; goimports -l $$files); \
		if [ -n "$$bad" ]; then echo "$$bad"; exit 1; fi'

# api/search/cron all hardcode "migrations" relative to the process's
# working directory, so they must run with CWD=src/backend (see
# CLAUDE.md) - hence `cd src/backend &&` below. Their "-config" default
# ("config.json") also only resolves from CWD=src/backend if pointed at
# config/config.json explicitly - the bare default is only correct
# inside the Docker images, which flatten config.json into the runtime
# root. --service-ports publishes 8091 to the host (docker compose run
# doesn't by default).
run-api:
	@echo "Starting API server on port 8091..."
	docker compose run --rm --service-ports backend-dev sh -c 'cd src/backend && go run ./cmd/api -config=config/config.json'

run-search:
	@if [ -z "$(KEYWORD)" ]; then \
		echo "Error: KEYWORD variable is required"; \
		echo "Usage: make run-search KEYWORD=laptop"; \
		exit 1; \
	fi
	$(BACKEND_RUN_DB) sh -c 'cd src/backend && go run ./cmd/search -config=config/config.json -keyword="$(KEYWORD)" $(if $(VERBOSE),-verbose=$(VERBOSE))'

run-cron:
	$(BACKEND_RUN_DB) sh -c 'cd src/backend && go run ./cmd/cron -config=config/config.json $(if $(VERBOSE),-verbose=$(VERBOSE)) $(if $(OUTPUT),-output=$(OUTPUT))'

up:
	@echo "Starting PostgreSQL, API, and frontend via Docker Compose..."
	docker compose up -d --build
	@echo "Waiting for services to be ready..."
	@sleep 3
	@echo "Up! Frontend: http://localhost:8092  API: http://localhost:8091/api/v1  DB Admin: http://localhost:8099"

down:
	@echo "Stopping Docker containers..."
	docker compose down

docker-clean:
	@echo "Stopping and removing Docker containers and volumes..."
	docker compose down -v

# Runs Nuxt's own dev server (HMR) inside a container - never needs npm
# on the host. Source is bind-mounted; node_modules is a named volume
# (see compose.yaml) so it survives across restarts without needing a
# host install.
frontend-dev:
	@echo "Starting frontend dev server (Docker) on port 8092..."
	docker compose --profile dev up --build frontend-dev

# Runs in the container (not the host) because build artifacts are
# owned by the container's root user, which the host user can't delete
# directly.
clean:
	@echo "Cleaning build artifacts..."
	$(BACKEND_RUN) sh -c 'rm -rf bin coverage.txt *.coverprofile'
	@echo "Clean complete!"

deps:
	@echo "Downloading dependencies..."
	$(BACKEND_RUN) sh -c 'cd src/backend && go mod download && go mod tidy'

# Quick start - sets up everything
setup: up deps
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Setup complete! You can now run: make run-search KEYWORD=laptop"
