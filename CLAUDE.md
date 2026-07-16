# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Go application that scrapes Czech second-hand marketplaces (Bazos, Sbazar, Avizo, Inzeruj, Aukro), stores listings in PostgreSQL, and tracks price/availability changes over time. Three entry points share the same backend: a `search` CLI, a `cron` change-detector, and an HTTP `api` served by a Nuxt 3 frontend. There's also an eBay-specific watcher (Browse API adapter + Telegram "good offer" notifications) layered on top of the same pipeline — see "eBay watcher" below.

## Important: paths differ from the README

The Go module root is the repo root (`module secondHand` in `go.mod`), but all Go source lives under `src/backend/`, not under a top-level `cmd/`/`internal/` as the README describes in places. Use the real paths shown below, e.g. `./src/backend/cmd/search`. The Makefile's targets already use these real paths and build into `./bin/` (gitignored) — `make build`/`make test`/`make run-search` etc. are safe to trust.

`go build ./...` / `go test ./...` from the repo root work fine — the `temp/` directory of scratch/debug artifacts and the root-level `api`/`cron`/`search` binaries that used to break this have been removed. `./src/...` and `./...` are now equivalent for build/test purposes.

## Common commands

Build and test can run from the repo root. **The three binaries (`search`, `cron`, `api`) must be run with CWD = `src/backend/`** — each hardcodes `"migrations"` as a path relative to the process's working directory (and defaults `-config` to `config.json`), so running them from the repo root will fail once they get past the DB connection step.

```bash
# Build (from repo root)
go build ./src/...

# Test everything (from repo root)
go test ./src/...
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./src/...

# Test a single package / single test
go test ./src/backend/internal/adapter/...
go test ./src/backend/internal/adapter/ -run TestName -v

# Search CLI (mock adapters — no network, no anti-bot issues)
cd src/backend && go run ./cmd/search -config=config/config.test.json -keyword="hemingway"

# Search CLI against real shop sites (may fail — anti-bot protection)
cd src/backend && go run ./cmd/search -config=config/config.json -keyword="laptop"

# Cron (checks saved searches for changes; output: cli|html|email)
cd src/backend && go run ./cmd/cron -config=config/config.test.json -verbose -output=cli

# API server
cd src/backend && go run ./cmd/api

# Frontend (Nuxt 3)
cd src/frontend && npm install && npm run dev      # http://localhost:3000
cd src/frontend && npm run build

# Full stack via Docker
docker compose up -d --build   # postgres:5432, api:8091, frontend:8092, adminer:8099
```

Config files live in `src/backend/config/`: `config.json` lists real shop URLs, `config.test.json` lists `mock-*` URLs that route to the in-memory `MockAdapter` (no real HTTP requests — this is the reliable way to exercise the full pipeline). DB/SMTP/scraping settings come from environment variables (`.env`, see `.env.example`), not the JSON config.

## Architecture

**Flow:** `cmd/{search,cron,api}` → `internal/service` → `internal/adapter` (scraping) + `internal/database` (persistence) → `internal/output` (formatting).

- **`internal/domain`** — the shared vocabulary: `Product`, `Search`, `ProductDiff`, and the `ShopAdapter`/`Repository`/`OutputFormatter` interfaces everything else implements or consumes. Start here to understand data shapes.
- **`internal/adapter`** — one file per shop (`bazos.go`, `sbazar.go`, `avizo.go`, `inzeruj.go`, `aukro.go`), each embedding `BaseAdapter` (colly-based HTTP collector with rate limiting/user-agent/domain restriction) and implementing `Search(ctx, keyword) ([]domain.Product, error)`. `registry.go` builds the active adapter set from `config.Shops`, dispatching by matching the shop URL's hostname substring (`bazos.cz`, `sbazar.cz`, ...) or, for testing, any URL containing `mock-`, which routes to `mock.go`'s `MockAdapter` (generates fake but realistic products, no network I/O). To add a shop: implement `ShopAdapter`, add a case in `registry.createAdapter`, add its URL to `config.json`.
- **`internal/service`** — `SearchService.SearchWithFilter` fans out to all (or one filtered) adapter concurrently via goroutines/channels, upserts results into the DB (dedup by product URL, price-change detection), and links products to the search. `DiffService` compares a search's previously stored products against a fresh search run to classify each listing as new/removed/price-up/price-down/unchanged; `GetDiffForAllSearches` runs this for every saved search (this is what `cmd/cron` drives).
- **`internal/database`** — `postgres.go` implements a `Repository` against pgx (`pgxpool`); `migrate.go` is a minimal hand-rolled migration runner (applies `*.up.sql` from `migrations/` in filename order, tracked in a `schema_migrations` table — no down-migration execution path, `.down.sql` files exist but aren't auto-applied). Note there are two near-duplicate `Repository` interfaces: `domain.Repository` (used by services) and `database.Repository` (used by `cmd/`, adds `Close()`, omits the "new/checked products" methods) — keep both in sync if you change the contract.
- **`internal/output`** — `OutputFormatter` implementations for CLI (colored terminal text), HTML (styled report, used for file output and email body), and `EmailSender` (SMTP via gomail, used by `cmd/cron -output=email`).
- **`cmd/api`** — a small `gorilla/mux` REST API (`/api/v1/health`, `/api/v1/searches`, `/api/v1/searches/{searchId}/products`) reading directly from the repository (bypasses `internal/service`); CORS-open by default. This is what the frontend consumes.
- **`src/frontend`** — Nuxt 4 / Vue 3 app with two pages: `pages/index.vue` (list of saved searches) and `pages/search/[id].vue` (products for a search), sharing `layouts/default.vue`, fetching from `NUXT_PUBLIC_API_BASE` (defaults to the Docker-network API host). Styled with Tailwind CSS v4 (CSS-first config, see `assets/css/main.css`'s `@theme` block) and the IBM Plex type family, self-hosted via `@nuxt/fonts`.

## Database

Two tables plus a join table, defined in `src/backend/migrations/001_initial_schema.up.sql`: `searches`, `products`, `search_products` (many-to-many, also tracks `is_new`/`found_at` per search). Migrations auto-run on startup of `cmd/search`, `cmd/cron`, and `cmd/api` (see the CWD requirement above). `002_good_offer_config.up.sql` adds nullable `searches.max_price`/`searches.avg_discount_pct` for the eBay watcher below.

## eBay watcher

On top of the generic scrape/diff pipeline, `internal/adapter/ebay.go` (`EbayAdapter`, Browse API OAuth2 client-credentials, shop name `ebay.com`) and `internal/output/telegram.go` (`TelegramNotifier`) implement a "good offer" alert: `cmd/cron`, after computing diffs as usual, separately evaluates each new/price-dropped `ebay.com` listing against its search's `max_price`/`avg_discount_pct` (`internal/service/goodoffer.go`'s `EvaluateGoodOffer`, either threshold sufficient, neither configured means silent) and sends a Telegram message on a match — this runs unconditionally, independent of `cmd/cron`'s `-output` flag, not as another output-format case. Thresholds are set per saved search via `cmd/search -max-price=... -avg-discount-pct=...` (not through `config.json`, which only configures which shop *adapters* are active). Config: `EBAY_CLIENT_ID`/`EBAY_CLIENT_SECRET`/`EBAY_API_BASE` and `TELEGRAM_BOT_TOKEN`/`TELEGRAM_CHAT_ID`/`TELEGRAM_API_BASE` env vars (see `.env.example`). Deployed as its own Kubernetes `CronJob` (`k8s/ebay-cronjob.yaml`, every 30 minutes, dedicated `docker/cron/Dockerfile` image) rather than folded into the `api` deployment; secrets via `k8s/ebay-secret.yaml.example`.
