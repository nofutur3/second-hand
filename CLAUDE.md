# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Go application that scrapes Czech second-hand marketplaces (Bazos, Sbazar, Avizo, Inzeruj, Aukro), stores listings in PostgreSQL, and tracks price/availability changes over time. Three entry points share the same backend: a `search` CLI, a `cron` change-detector, and an HTTP `api` served by a Nuxt 3 frontend. There's also an eBay-specific watcher (Browse API adapter + Telegram "good offer" notifications) layered on top of the same pipeline — see "eBay watcher" below.

## Important: module root vs. repo root

The Go module root is `src/backend/` (`module secondHand` in `src/backend/go.mod`), not the repo root — packages are referenced as `./cmd/search` etc. from within `src/backend/`, not `./src/backend/cmd/search` from the repo root. The Makefile's targets already `cd src/backend` before invoking `go` and build into `../../bin/` (gitignored, so binaries land at repo-root `./bin/`) — `make build`/`make test`/`make run-search` etc. are safe to trust.

## Common commands

No Go toolchain needs to be installed on the host: every `make` target below runs Go inside the `backend-dev` container (see `compose.yaml`), bind-mounting the repo — call `make <target>`, not `go build`/`go test`/`go run` directly. **The three binaries (`search`, `cron`, `api`) run with CWD = `src/backend/`** — each hardcodes `"migrations"` as a path relative to the process's working directory (and defaults `-config` to `config.json`); the Makefile's `run-*` targets already `cd` into place before invoking them.

```bash
# Build all three binaries into ./bin/
make build

# Test everything
make test

# Lint (gofmt/goimports check)
make lint

# Test a single package / single test (drop to the container directly)
docker compose run --rm --no-deps backend-dev sh -c 'cd src/backend && go test ./internal/adapter/... -run TestName -v'

# Search CLI (mock adapters — no network, no anti-bot issues)
docker compose run --rm backend-dev sh -c 'cd src/backend && go run ./cmd/search -config=config/config.test.json -keyword="hemingway"'

# Search CLI against real shop sites (may fail — anti-bot protection)
make run-search KEYWORD=laptop

# Cron (checks saved searches for changes; output: cli|html|email)
make run-cron OUTPUT=html VERBOSE=true

# API server (standalone, not the `api` compose service - useful for iterating without rebuilding its image)
make run-api

# Frontend (Nuxt 3) - hot reload, no host npm needed
make frontend-dev      # http://localhost:8092

# Full stack via Docker
make up   # postgres:5432, api:8091, frontend:8092, adminer:8099
```

Config files live in `src/backend/config/`: `config.json` lists real shop URLs, `config.test.json` lists `mock-*` URLs that route to the in-memory `MockAdapter` (no real HTTP requests — this is the reliable way to exercise the full pipeline). DB/SMTP/scraping settings come from environment variables (`.env`, see `.env.example`), not the JSON config.

## Architecture

**Flow:** `cmd/{search,cron,api}` → `internal/service` → `internal/adapter` (scraping) + `internal/database` (persistence) → `internal/output` (formatting).

- **`internal/domain`** — the shared vocabulary: `Product`, `Search`, `ProductDiff`, and the `ShopAdapter`/`Repository`/`OutputFormatter` interfaces everything else implements or consumes. Start here to understand data shapes.
- **`internal/adapter`** — one file per shop (`bazos.go`, `sbazar.go`, `avizo.go`, `inzeruj.go`, `aukro.go`), each embedding `BaseAdapter` (colly-based HTTP collector with rate limiting/user-agent/domain restriction) and implementing `Search(ctx, keyword) ([]domain.Product, error)`. `registry.go` builds the active adapter set from `config.Shops`, dispatching by matching the shop URL's hostname substring (`bazos.cz`, `sbazar.cz`, ...) or, for testing, any URL containing `mock-`, which routes to `mock.go`'s `MockAdapter` (generates fake but realistic products, no network I/O). To add a shop: implement `ShopAdapter`, add a case in `registry.createAdapter`, add its URL to `config.json`.
- **`internal/service`** — `SearchService.SearchWithFilter` fans out to all (or one filtered) adapter concurrently via goroutines/channels, upserts results into the DB (dedup by product URL, price-change detection), and links products to the search. `DiffService` compares a search's previously stored products against a fresh search run to classify each listing as new/removed/price-up/price-down/unchanged; `GetDiffForAllSearches` runs this for every saved search (this is what `cmd/cron` drives).
- **`internal/database`** — `postgres.go` implements a `Repository` against pgx (`pgxpool`); `migrate.go` is a minimal hand-rolled migration runner (applies `*.up.sql` from `migrations/` in filename order, tracked in a `schema_migrations` table — no down-migration execution path, `.down.sql` files exist but aren't auto-applied). Note there are two near-duplicate `Repository` interfaces: `domain.Repository` (used by services) and `database.Repository` (used by `cmd/`, adds `Close()`, omits the "new/checked products" methods) — keep both in sync if you change the contract.
- **`internal/output`** — `OutputFormatter` implementations for CLI (colored terminal text), HTML (styled report, used for file output and email body), and `EmailSender` (SMTP via gomail, used by `cmd/cron -output=email`).
- **`cmd/api`** — a small `gorilla/mux` REST API (`/api/v1/health`, `/api/v1/searches`, `/api/v1/searches/{searchId}/products`, `/api/v1/searches/{searchId}/products/{productId}` PATCH, `/api/v1/ebay-description`) reading directly from the repository (bypasses `internal/service`); CORS-open by default. Every route except `/health` sits behind `basicAuthMiddleware` — a single shared password from `APP_PASSWORD` (empty disables the check; see "Access control" below). This is what the frontend consumes.
- **`src/frontend`** — Nuxt 4 / Vue 3 app with two pages: `pages/index.vue` (list of saved searches) and `pages/search/[id].vue` (products for a search, including a per-search shop filter persisted in `localStorage`), sharing `layouts/default.vue`. API base URL is split in `nuxt.config.ts`'s `runtimeConfig`: `apiBaseServer` (internal Docker hostname, `NUXT_API_BASE_SERVER`) for SSR fetches that run inside the Docker network, vs. `public.apiBase` (`NUXT_PUBLIC_API_BASE`) for client-side fetches from the browser — `composables/useApiBase.ts` picks the right one via `import.meta.server`. `server/middleware/auth.ts` mirrors `cmd/api`'s Basic Auth gate (same `APP_PASSWORD`, empty disables it). Styled with Tailwind CSS v4 (CSS-first config, see `assets/css/main.css`'s `@theme` block) and the IBM Plex type family, self-hosted via `@nuxt/fonts`.

## Access control

Both `cmd/api` and the frontend are gated behind HTTP Basic Auth on a single shared `APP_PASSWORD` (any username, only the password is checked — not a multi-user account system). Empty/unset disables the check on both, so local dev is ungated by default. `/api/v1/health` is always exempt (k8s liveness/readiness probes). Since the frontend and API are different origins, the browser prompts separately for each on first visit. Production value lives in the `secondhand-app` k8s secret (`nofutur3/osiris-cluster`'s `secondhand/app-secret.yaml.example` documents the `kubectl create secret` command; not committed anywhere).

## Database

Two tables plus a join table, defined in `src/backend/migrations/001_initial_schema.up.sql`: `searches`, `products`, `search_products` (many-to-many, also tracks `is_new`/`found_at` per search). Migrations auto-run on startup of `cmd/search`, `cmd/cron`, and `cmd/api` (see the CWD requirement above). `002_good_offer_config.up.sql` adds nullable `searches.max_price`/`searches.avg_discount_pct` for the eBay watcher below. `003_product_status.up.sql` adds `search_products.is_hidden`/`is_active` (per-search visibility, exposed via `cmd/api`'s `PATCH /searches/{id}/products/{id}` and `GetProductsBySearchIDWithStatus`). `004_good_offer_flag.up.sql` adds `search_products.is_good_offer`, set one-way by `cmd/cron` (`repo.SetGoodOffer`) when a listing matches the good-offer thresholds, surfaced in the frontend as a badge. `005_ebay_shipping_bid.up.sql` adds `products.shipping_cost`/`bid_count` (eBay-only; nil/zero elsewhere).

## eBay watcher

On top of the generic scrape/diff pipeline, `internal/adapter/ebay.go` (`EbayAdapter`, Browse API OAuth2 client-credentials, shop name `ebay.com`) and `internal/output/telegram.go` (`TelegramNotifier`) implement a "good offer" alert: `cmd/cron`, after computing diffs as usual, separately evaluates each new/price-dropped `ebay.com` listing against its search's `max_price`/`avg_discount_pct` (`internal/service/goodoffer.go`'s `EvaluateGoodOffer`, either threshold sufficient, neither configured means silent) and sends a Telegram message on a match — this runs unconditionally, independent of `cmd/cron`'s `-output` flag, not as another output-format case. Matches are also persisted (`search_products.is_good_offer`, see Database above). Thresholds are set per saved search via `cmd/search -max-price=... -avg-discount-pct=...` (not through `config.json`, which only configures which shop *adapters* are active). Config: `EBAY_CLIENT_ID`/`EBAY_CLIENT_SECRET`/`EBAY_API_BASE`, `EBAY_SHIP_TO_COUNTRY`/`EBAY_SHIP_TO_POSTAL_CODE` (default `CZ`/`58601`), and `TELEGRAM_BOT_TOKEN`/`TELEGRAM_CHAT_ID`/`TELEGRAM_API_BASE` env vars (see `.env.example`). Deployed as its own Kubernetes `CronJob` (every 30 minutes, dedicated `docker/cron/Dockerfile` image) rather than folded into the `api` deployment; manifests and secret docs live in `nofutur3/osiris-cluster`'s `secondhand/` directory, not in this repo — this repo's CI only builds/pushes images and bumps the image tag there.

Every Browse API request carries buyer-context headers (`EbayAdapter.setBuyerContextHeaders`): `X-EBAY-C-ENDUSERCTX: contextualLocation=...` resolves price and shipping cost in CZK for the configured delivery address (`item_summary/search` already returns `shippingOptions`/`currentBidPrice`/`bidCount` for auctions — no per-item call needed), and `Accept-Language: en-US` keeps titles in English (contextualLocation alone makes eBay machine-translate them into the destination country's language). `domain.Product.ShippingCost`/`BidCount` carry this through to the API/frontend, which shows a price + shipping = total breakdown and `auction · N bids`.

Good-offer thresholds are judged against `internal/service/goodoffer.go`'s `PerUnitCost` — `TotalCost` (price + shipping) divided by a title-parsed lot size (`detectLotSize`: "Lot of 6", "5 set", "Sada 10", "10-pack", "Bundle of 3", default 1) — not the raw listing price, so a €300 "Lot of 6" doesn't wrongly pass a €50 ceiling. `detectLotSize` is a best-effort text heuristic (titles are free text), not a structured eBay field.

Full HTML descriptions aren't included in search results (`shortDescription` only); `cmd/api`'s `GET /ebay-description?url=...` fetches one on demand via `EbayAdapter.GetDescription` (`get_item_by_legacy_id`, ID regex-extracted from the URL) and strips HTML server-side before returning plain text — never rendered as HTML client-side, so there's no XSS surface from seller-authored markup. Deliberately not called during bulk cron/search runs (Browse API per-app rate limits).

`internal/service/search.go`'s upsert always calls `UpdateProduct` on every rescrape (not gated on price having changed) — title/shipping/condition/etc. can change independent of price, and gating on price alone left stale data (e.g. a not-yet-fixed bad translation) behind indefinitely.

@.claude/workflow/CLAUDE.md
