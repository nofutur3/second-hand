# Assignment: eBay Nintendo-parts watcher

## Goal

Add a watcher to this app that searches eBay for Nintendo parts/consoles on a
schedule and notifies Jakub via Telegram when a listing looks like a good
offer. This is a real feature addition to the existing second-hand scraper,
not a new project — it should reuse the app's existing architecture
(adapters, services, database, output formatters) wherever that architecture
already fits, and extend it cleanly where it doesn't.

There's a stale TODO about this in the second-brain repo
(`notes/areas/infrastructure.md`: `eBay watcher (Go/Telegram) — deployed?
where?`) — apparently discussed before, never actually built (or built and
lost). Treat this as starting fresh; don't go looking for a prior
implementation.

## Prerequisites (Jakub has to do these by hand — flag if missing, don't block on them)

- An eBay Developer Program account + application, for a Client ID/Secret
  (developer.ebay.com). **Use eBay's official Browse API, not scraping** —
  unlike bazos/sbazar/avizo/inzeruj/aukro, eBay actively blocks scrapers and
  this app's own README already flags scraping as unreliable/legally risky.
  The Browse API's `item_summary/search` endpoint only needs an OAuth2
  client-credentials app token (no user login flow) and is free at low
  volume.
- A Telegram bot token (via @BotFather) and the target chat ID.

Both agents can and should build/test against mocks in the meantime (see
`internal/adapter/mock.go` for the existing pattern) — don't stall on
credentials that only Jakub can obtain.

## Architecture context (read the real files, this is just a map)

- `src/backend/internal/domain/models.go` — `ShopAdapter` interface
  (`Name()`, `Search(ctx, keyword) ([]Product, error)`, `SupportsSearch()`).
  `Product` already has `AuctionType` (`sale`/`auction`) and `EndingTime`
  fields — this model was clearly built with eBay-style auctions in mind,
  even though no adapter uses them yet.
- `src/backend/internal/adapter/registry.go` — `createAdapter()` dispatches
  by substring-matching the shop URL (`bazos.cz`, `sbazar.cz`, ...). Add an
  `ebay.com` branch and a `NewEbayAdapter`.
- `src/backend/internal/config/config.go` — `Config.Shops` (JSON,
  `config.json`) for adapter URLs; `Database`/`SMTP`/`Scraping` are
  env-loaded via `getEnv`/`getEnvInt`. Add equivalent `Ebay`/`Telegram`
  config structs the same way.
- `src/backend/internal/output/email.go` — `EmailSender` pattern
  (`SendHTML`, credentials from config, clear "not configured" error). Mirror
  this for a new `TelegramNotifier`.
- `src/backend/cmd/cron/main.go` — the existing scheduled entry point: loads
  saved searches, runs `DiffService.GetDiffForAllSearches`, formats/outputs
  by `-output` flag (`cli`/`html`/`email`). This is the natural place to
  wire in eBay + Telegram, but "notify on every diff" isn't the same as
  "notify on a *good offer*" — see below.
- `src/backend/migrations/001_initial_schema.up.sql` + `internal/database/migrate.go`
  — hand-rolled migration runner, applies `*.up.sql` in filename order,
  tracked in `schema_migrations`. A schema change means a new
  `002_*.up.sql`/`.down.sql` pair, not an edit to 001.
- `k8s/` — already has `namespace.yaml`, `postgres-*`, `api-deployment.yaml`,
  `frontend-deployment.yaml`, `ingress.yaml`, all in the `secondhand`
  namespace on the `osiris` cluster. Deploys via
  `.github/workflows/ci-cd.yml` (build → push to `registry.nofutur3.com` →
  `kubectl apply` + `kubectl set image` + rollout wait). Secrets are applied
  by hand once (see `k8s/postgres-secret.yaml.example`) and deliberately
  excluded from the CI apply list — follow that same convention for any new
  secret.

## Requirements

1. **`EbayAdapter`** implementing `domain.ShopAdapter`, backed by eBay's
   Browse API (OAuth2 client-credentials), not scraping. Needs its own
   config (client ID/secret, token caching — app tokens are long-lived,
   don't re-auth every request).
2. **A concrete, configurable "good offer" definition.** This app doesn't
   have one today — `Search` only has a `Keyword`. You need to design this:
   a per-search price ceiling, a "% below recent average for this keyword"
   comparison, or something else defensible. It must be config-driven per
   saved search, not a hardcoded constant. This is a real design decision —
   make it deliberately and write down why.
3. **`TelegramNotifier`** in `internal/output`, used specifically for "good
   offer" matches (not just wired to every diff the way `-output=email` is).
4. **Saved eBay searches for actual Nintendo parts/consoles** (e.g. Joy-Con
   pairs, Game Boy Color shells, Switch docks — Jakub can refine the exact
   list) using the existing `config.json` shop-config mechanism.
5. **Scheduled execution** — extend `k8s/` with whatever's appropriate
   (likely a `CronJob`, not a long-running `Deployment`, given this runs
   `cmd/cron` on an interval rather than serving requests). Figure out
   whether that means a new Docker image (`docker/cron/Dockerfile`) or
   reusing the existing api image with a different entrypoint/command, and
   extend `.github/workflows/ci-cd.yml` accordingly.
6. **Tests must not require live eBay/Telegram credentials.** Follow the
   existing `mock-*` URL convention (`registry.go`, `mock.go`) so
   `go test ./src/...` stays credential-free in CI.

## Out of scope

- Don't touch the bazos/sbazar/avizo/inzeruj/aukro adapters.
- Don't change the frontend beyond what's trivially needed (this is a
  backend/notification feature, not a UI feature).
- Don't touch `secondhand-api`/`secondhand-frontend` k8s manifests except to
  add new resources alongside them.
- Don't modify anything in the `second-brain` repo — if the stale
  infrastructure.md TODO should be updated once this actually ships, leave a
  note for Jakub to do it himself, don't cross repos.

## Definition of done

- CI green (`go vet ./src/...`, `go test ./src/...`) on the merged result.
- At least one real Nintendo-part saved search configured.
- A mock "good offer" listing demonstrably triggers a Telegram send (dry-run
  acceptable if live credentials aren't available yet — verify the trigger
  logic and the send call are both exercised, e.g. via a fake/test Telegram
  endpoint).
- k8s manifests added, following the existing conventions, validated the
  same way the existing ones were (`kubectl kustomize` / `kubectl apply
  --dry-run=client` locally — no live cluster access needed for this).
- Secrets documented in a `.example` file, gitignored for real, never in the
  CI apply list.
- Final PR merged to `main` after Planner review.

## How this gets built

Two Claude Code agents work this: a **Planner** (plans, reviews, decides —
doesn't write code) and a **Worker** (implements, tests, opens PRs — doesn't
plan or merge). They coordinate entirely through files in `.agents/` and
GitHub PRs, since they're two independent sessions with no shared memory.
See `.agents/PLANNER_PROMPT.md` and `.agents/WORKER_PROMPT.md` for their
respective instructions, and `.agents/plan.md` for the live task board once
the Planner has populated it.
