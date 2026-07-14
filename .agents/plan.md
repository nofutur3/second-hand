# Task board — eBay Nintendo-parts watcher

Owned by the Planner. Worker: read-only.

Last updated by Planner: 2026-07-14 09:47 (restored after an unexplained
overwrite — see "Planner integrity note" below and the 09:47 entry in
`planner-log.md`).

## Status legend

`[ ]` not started · `[~]` in progress · `[x]` done · `[!]` blocked (see note)

## Planner integrity note (read this first)

Between the 09:40 and 09:47 loop passes, this file was overwritten with a
different task list (product/schema restructuring around an `eBayID`
column, a `1.5 * avg_price * condition_factor` "good offer" formula, a
`-output=telegram` formatter, a "CollectUS API" reference not present
anywhere in `ASSIGNMENT.md`). I did not write that content.

I'm not adopting it, for concrete reasons, not just "I didn't write it":

- `worker-log.md` has no new entries — still just the template header. The
  protocol requires the Worker to log a status line on every pass,
  including idle ones. Whoever/whatever wrote the alternate plan left no
  trail through the channel that's supposed to explain Worker actions.
- The proposed heuristic (`price <= 1.5 * avg_price * condition_factor`)
  is backwards: it flags listings *up to 50% above* average as a good
  offer, not below. That's a logic bug, not a style difference.
- `-output=telegram` contradicts `ASSIGNMENT.md` verbatim: "used
  specifically for 'good offer' matches (not just wired to every diff the
  way `-output=email` is)". Wiring Telegram as another `-output` case is
  the exact thing the assignment says not to do.
- No open PR, no pushed `feature/ebay-watcher` branch — there's no
  implementation this alternate plan could even be describing in
  retrospect.

Keeping `.agents/plan.md` correct — including reverting content that's
wrong on the merits, regardless of source — is this file's whole purpose
per `.agents/README.md`. If this was Jakub deliberately redirecting the
plan, the fix is simple: say so in a way I'll see (a `worker-log.md` entry
is read-only to the Worker but I don't currently have a channel for
"message from Jakub directly" other than this repo — a comment on this
file's next real PR, or just telling me directly, both work). Absent that,
I'm treating this as noise, not signal, and continuing from the plan
below, which is unchanged from the 09:35 version other than this note and
one genuine improvement folded into D2 (eBay's actual condition-string
values, which the alternate content happened to gesture at usefully even
though the rest of it doesn't hold up).

## Prerequisites check (Planner, 2026-07-14)

Checked `.env` / `.env.example` for `EBAY_*` / `TELEGRAM_*` — **neither eBay
Developer credentials nor a Telegram bot token exist yet.** This is
genuinely missing, not an oversight — it's Jakub's to obtain
(developer.ebay.com app + @BotFather bot). Per `ASSIGNMENT.md`, this does
**not** block implementation: everything below is buildable and testable
against the `mock-*` adapter convention and a fake Telegram HTTP endpoint in
tests. Items that need live credentials for a real end-to-end run are
marked `[!] needs real creds (Jakub)` — those stay open after everything
else is merged, and that's expected, not a failure state.

---

## Design decisions (decided by Planner, not TBD — implement as specified)

### D1. "Good offer" heuristic

**Decision:** a listing is a "good offer" if **either** of two independently
optional, per-search-configured thresholds is met:

1. **Price ceiling** — `searches.max_price` (nullable numeric). Listing
   price `<= max_price`.
2. **Trailing-average discount** — `searches.avg_discount_pct` (nullable
   numeric, e.g. `20.00` = "20% below average"). Compute the average price
   of all *previously stored* products for that search (via
   `search_products` → `products`, excluding the candidate itself). If the
   search has fewer than **3** prior stored products, this check is
   skipped entirely for that evaluation (not enough signal — a single
   prior listing isn't an "average," and treating it as one produces noisy
   false positives on a search's first few runs). If `>= 3` prior
   products exist and `price <= average * (1 - avg_discount_pct/100)`,
   it's a good offer.

If **neither** field is set on a search, that search never triggers a
Telegram notification (it still gets scraped/stored/diffed as normal —
only the notification is gated). Silence-by-default is the right failure
mode here: a search someone forgot to configure should not spam Telegram,
and a missing config is trivially fixable, while a false-positive spam
storm is not trivially undoable.

**Why this shape, not something simpler:** a bare price ceiling
(`max_price` alone) was the simplest option and is sufficient by itself,
but secondhand part prices for the same keyword drift over time (e.g.
"Joy-Con pair" trending up or down over months) — a static ceiling either
goes stale or has to be babysat. The trailing-average option reuses data
this app *already collects* (products are stored per search on every diff
run) so it costs a schema field and a query, not new infrastructure. Both
are optional and independent so Jakub can use whichever fits a given
search (a ceiling for something with a known target price, a discount-%
for something where "market rate" drifts and a relative signal matters
more).

**Scope of evaluation — which diffs get checked:** only `ProductDiff`
entries with `DiffType` `new` or `price_down`, and only where
`Product.ShopSource` is the eBay adapter's name (`"ebay.com"`, see D2 for
adapter naming) — i.e., this is genuinely the *eBay* watcher, not a
generic "any shop can now ping Telegram" feature. Nothing stops Jakub from
setting `max_price` on a non-eBay search later, so don't hardcode
`ShopSource == "ebay.com"` deep inside the heuristic function itself —
filter at the call site in `cmd/cron` and keep `EvaluateGoodOffer(search,
product, priorPrices)` shop-agnostic and independently unit-testable.

### D2. eBay Browse API adapter

- `EbayAdapter` implements `domain.ShopAdapter`, `Name()` returns
  `"ebay.com"` (matches existing adapters' `Name()` = shop hostname
  convention, e.g. `bazos.cz`).
- OAuth2 client-credentials flow against
  `POST {EbayAPIBase}/identity/v1/oauth2/token`
  (`grant_type=client_credentials&scope=https://api.ebay.com/oauth/api_scope`,
  HTTP Basic auth with client ID/secret). Cache the resulting token +
  expiry **in-memory on the adapter struct**, mutex-protected, refresh
  lazily a short buffer (e.g. 60s) before expiry. This does *not* persist
  across process restarts — that's fine and intentional: `cmd/cron` is a
  new process per CronJob run (see D4), so persistence would need
  DB/Redis storage for a benefit that only matters *within* a single run
  across multiple saved eBay searches, which in-memory caching already
  covers (`GetDiffForAllSearches` loops all searches in one process
  lifetime — that's the case worth optimizing, not cross-run reuse).
- Search via `GET {EbayAPIBase}/buy/browse/v1/item_summary/search?q={keyword}`.
  Map `itemSummaries[]` → `domain.Product`: `price.value`/`price.currency`,
  `itemWebUrl` → `URL`, `image.imageUrl` → `ImageURL`, `itemLocation` →
  `Location`, `seller.username` → `SellerName`. `condition` → map to the
  existing `domain.Condition` enum on a best-effort basis; eBay's Browse
  API returns human-readable condition strings, notably `"New"`,
  `"New other"`, `"New with defects"`, `"Certified refurbished"`,
  `"Seller refurbished"`, `"Used"`, `"For parts or not working"` —
  map `"New"`/`"New with tags"` → `ConditionNew`, anything containing
  `"refurbished"` → `ConditionLikeNew` (closest existing enum value — this
  app doesn't have a distinct "refurbished" condition, don't invent one),
  `"Used"`/`"Pre-owned"` → `ConditionUsed`, `"For parts or not working"` →
  `ConditionDamaged`, anything unrecognized → `ConditionUnknown`.
  `buyingOptions` containing `"AUCTION"` → `AuctionType = auction` (+
  `itemEndDate` → `EndingTime`), otherwise `AuctionType = sale`.
- `EbayAPIBase` is a config field (default `https://api.ebay.com`) so
  tests can point it at an `httptest.Server` — same reasoning as
  Telegram's testable base URL in D3. This is what makes "tests must not
  require live eBay credentials" achievable for the *real* adapter, not
  just via the `mock-` bypass in `registry.go` (both should work: `mock-`
  URLs continue to route to `MockAdapter` and skip `EbayAdapter` entirely;
  `EbayAdapter` itself is *also* independently tested against a fake
  server, since it's new code that deserves direct coverage, not just
  coverage-by-bypass).

### D3. TelegramNotifier

- New file `internal/output/telegram.go`, pattern-matched on
  `internal/output/email.go`: `NewTelegramNotifier(cfg
  *config.TelegramConfig)`, method `SendGoodOffer(product domain.Product,
  search domain.Search) error`. "Not configured" guard (empty bot token or
  chat ID) returns a clear error, same as `EmailSender.SendHTML`.
- Plain `net/http` POST to `{TelegramAPIBase}/bot{token}/sendMessage`
  with a JSON body (`chat_id`, `text`, sensible `parse_mode`). **Decision:
  no Telegram SDK dependency** — this is a single trivial endpoint; adding
  a library for one POST doesn't fit this repo's existing dependency
  posture (colly/pgx/gomail are each justified by real complexity this
  isn't).
- `TelegramAPIBase` config field, default `https://api.telegram.org`, same
  testability rationale as D2's `EbayAPIBase` — point it at
  `httptest.Server` in tests to exercise the actual send call without a
  real bot/chat, satisfying the Definition of Done's "fake/test Telegram
  endpoint" requirement literally.
- **Not a `-output` formatter — this is explicit in `ASSIGNMENT.md`, not a
  style preference.** `EmailSender`/`CLIFormatter`/`HTMLFormatter` render
  *all* diffs when selected via `-output=X`. Telegram is different in
  kind: it fires only for good-offer matches (D1), independent of whatever
  `-output` flag `cmd/cron` was given. Wire it as an unconditional extra
  step in `cmd/cron/main.go` after diffs are computed, **not** as a new
  `case "telegram":` in the existing output-format switch. (An earlier
  version of this file briefly proposed exactly that `-output=telegram`
  shape — it was wrong; see the integrity note above. Do not implement
  that version.)

### D4. CronJob: new image, not reuse of the api image

**Decision:** build a new `docker/cron/Dockerfile` (mirrors
`docker/api/Dockerfile` structurally: same builder base, same
migrations/config.json copy pattern, builds `./src/backend/cmd/cron`
instead of `./src/backend/cmd/api`) → new image
`registry.nofutur3.com/secondhand-cron`.

**Why not reuse the api image with a different command:** the api image's
build step (`go build -o api ./src/backend/cmd/api`) only produces the
`api` binary — it doesn't contain `cron` at all. "Reuse with a different
command" would require changing `docker/api/Dockerfile` to build *both*
binaries into one image, which (a) means every api-image rebuild also
rebuilds/repackages the cron binary and vice versa, coupling two
independently-deployed things for no real benefit, and (b) breaks the
one-binary-per-image convention this repo already established with
separate api/frontend images. A dedicated small image is more consistent
and keeps the CronJob's image lifecycle independent of the api
Deployment's.

**Schedule decision:** `*/30 * * * *` (every 30 minutes). Not specified in
`ASSIGNMENT.md`; secondhand listings for specific parts don't turn over
fast enough to need sub-10-minute polling, and 30 minutes keeps eBay API
call volume trivially within free-tier limits even with several saved
searches. Jakub can tighten this later by editing the CronJob manifest —
no code change needed.

**CronJob specifics:** `concurrencyPolicy: Forbid` (don't overlap runs if
one takes longer than the interval), `restartPolicy: OnFailure` in the
pod template, same `imagePullSecrets`/`resources` shape as
`api-deployment.yaml`, env from existing `secondhand-postgres` secret
(DB_PASSWORD) plus a new `secondhand-ebay` secret (EBAY_CLIENT_ID,
EBAY_CLIENT_SECRET, TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID) — same
hand-applied, gitignored, CI-apply-excluded convention as
`k8s/postgres-secret.yaml.example`.

---

## Checklist

### Phase 1 — Config plumbing
- [x] Add `EbayConfig` (`ClientID`, `ClientSecret`, `APIBase` default
      `https://api.ebay.com`) and `TelegramConfig` (`BotToken`, `ChatID`,
      `APIBase` default `https://api.telegram.org`) to
      `internal/config/config.go`, env-loaded via `getEnv` (mirrors
      `SMTPConfig`). Add `EBAY_CLIENT_ID`, `EBAY_CLIENT_SECRET`,
      `EBAY_API_BASE`, `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`,
      `TELEGRAM_API_BASE` to `.env.example` (values blank/placeholder,
      same as existing `SMTP_*` placeholders).

### Phase 2 — EbayAdapter (D2)
- [x] Implement `internal/adapter/ebay.go` per D2.
- [x] Wire `ebay.com` branch into `registry.createAdapter` →
      `NewEbayAdapter(url, cfg.Ebay, delayMS, timeoutSec)` (registry
      already has the full `*config.Config`, just thread `cfg.Ebay`
      through — check whether `createAdapter`'s signature needs to grow or
      whether `Registry` should just close over `cfg` directly instead of
      passing individual fields; Worker's call, either is fine as long as
      the other adapters' call sites don't need to change).
- [x] Add an `ebay.com` shop entry to `config.json`
      (`https://www.ebay.com`) and a `mock-ebay` entry to
      `config.test.json`, matching the existing pattern.
- [x] Unit tests in `internal/adapter/ebay_test.go`: OAuth token fetch +
      caching (assert the token endpoint is hit once across two `Search`
      calls within the token's validity window) and Browse API response
      parsing (condition/auction-type/price mapping — including the
      specific eBay condition strings listed in D2), both against
      `httptest.Server`. No real credentials.

**PR #1 merged 2026-07-14** (`0fe4d01`) — see planner-log.md 10:30 entry.

### Phase 3 — Schema + repository (D1)
- [ ] New migration `002_good_offer_config.up.sql` /
      `002_good_offer_config.down.sql`: `ALTER TABLE searches ADD COLUMN
      max_price NUMERIC(10,2)`, `ALTER TABLE searches ADD COLUMN
      avg_discount_pct NUMERIC(5,2)` (both nullable, no default). Down
      migration drops both columns (not auto-applied by `migrate.go`
      today, but keep the pair for consistency with `001_*`).
- [ ] Add `MaxPrice *float64` / `AvgDiscountPct *float64` to
      `domain.Search`.
- [ ] Add `SetGoodOfferConfig(ctx, searchID int64, maxPrice
      *float64, avgDiscountPct *float64) error` to both `domain.Repository`
      and `database.Repository` interfaces + `PostgresRepository`
      implementation. **Do not change `CreateSearch`'s signature** —
      it's called for every ad-hoc search across all shops, not just
      eBay good-offer searches; a separate setter keeps that call site
      untouched.
- [ ] Update `GetSearchByID`/`GetSearchByKeyword`/`GetAllSearches` SQL to
      select the two new columns so `domain.Search` is fully populated
      wherever a search is read.
- [ ] Add `-max-price` and `-avg-discount-pct` optional flags to
      `cmd/search/main.go`; if either is set, call `SetGoodOfferConfig`
      after `CreateSearch`. This is the mechanism for seeding real
      searches (see Phase 5) — there's no separate "keyword list" in
      `config.json`; `config.json`'s `shops` array configures which
      *adapters* are active (already covered by adding the `ebay.com`
      entry in Phase 2), while individual saved *searches* (keywords +
      thresholds) are DB rows created via `cmd/search`, same as today.
      **Correction to `ASSIGNMENT.md` requirement #4's wording** — "using
      the existing config.json shop-config mechanism" conflates the two;
      logged here so the Worker isn't stuck trying to add a per-keyword
      price field to `config.json`, which has no natural slot for it.

### Phase 4 — Good-offer evaluation + Telegram (D1, D3)
- [ ] `EvaluateGoodOffer(search domain.Search, product domain.Product,
      priorPrices []float64) bool` — pure function per D1, in
      `internal/service` (new file, e.g. `goodoffer.go`). Unit tests
      covering: ceiling met/not met, discount met/not met (with >=3 and
      <3 prior prices), neither configured, both configured (either
      sufficient).
- [ ] `internal/output/telegram.go` per D3, with tests against
      `httptest.Server` (assert the request body/URL, assert "not
      configured" error path, no network access).
- [ ] Wire into `cmd/cron/main.go`: after `diffService.GetDiffForAllSearches`,
      for each diff entry where `DiffType` is `new` or `price_down` and
      `Product.ShopSource == "ebay.com"`, fetch prior prices for that
      search (new repo query or reuse `GetProductsBySearchID` +
      filter/map to `[]float64` — Worker's call which is less invasive),
      call `EvaluateGoodOffer`, and on `true` call
      `TelegramNotifier.SendGoodOffer`. Runs regardless of `-output` value
      (see D3 — not a formatter branch). Log (stdout, matching this repo's
      existing `fmt.Printf` style, not a new logging framework) each
      notification attempt and its result.
- [ ] Integration-style test (still no live creds) proving a mock "good
      offer" listing → `EvaluateGoodOffer` true → an actual HTTP call
      lands on a fake Telegram endpoint. This directly satisfies the
      Definition of Done's explicit requirement — don't skip it as
      "covered by the unit tests above," it needs to exercise the
      trigger-to-send path together, e.g. at the `cmd/cron` logic level
      or a small service-level test wiring a `MockAdapter` (with a price
      below a test `max_price`) through diff + evaluate + notify.

### Phase 5 — Real saved searches (Definition of Done requirement)
- [ ] Once Phase 3+4 are merged: create at least one real saved eBay
      search via `cmd/search -adapter=ebay.com -keyword="joy-con pair"
      -max-price=<value>` (exact keyword list — Joy-Con pairs, Game Boy
      Color shells, Switch docks, etc. — Jakub refines, but at least one
      concrete one must exist and be demonstrated, per Definition of
      Done). **This step needs real eBay credentials to actually run
      against the live API** — `[!] needs real creds (Jakub)`. Until
      credentials exist, this can be demonstrated against `mock-ebay`
      instead to prove the mechanism works, with a note that the same
      command works unmodified once `EBAY_CLIENT_ID`/`EBAY_CLIENT_SECRET`
      are set and `-adapter=ebay.com` points at the real shop entry.

### Phase 6 — k8s + CI/CD (D4)
- [ ] `docker/cron/Dockerfile` per D4.
- [ ] `k8s/ebay-secret.yaml.example` (EBAY_CLIENT_ID, EBAY_CLIENT_SECRET,
      TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID placeholders, same header
      comment style as `k8s/postgres-secret.yaml.example`). Add
      `/k8s/ebay-secret.yaml` to `.gitignore` (mirrors the existing
      `/k8s/postgres-secret.yaml` entry).
- [ ] `k8s/ebay-cronjob.yaml` per D4's CronJob specifics.
- [ ] Validate locally: `kubectl apply --dry-run=client -f
      k8s/ebay-cronjob.yaml -f k8s/ebay-secret.yaml.example` (or the
      renamed real-secret form) — no live cluster access needed, matches
      how the existing manifests were validated.
- [ ] Extend `.github/workflows/ci-cd.yml`: new "Build and push cron
      image" step under `build` job (mirrors the api/frontend steps,
      `file: docker/cron/Dockerfile`, tags
      `secondhand-cron:latest`/`:sha-${{ github.sha }}`); add
      `k8s/ebay-cronjob.yaml` to the `deploy` job's `kubectl apply` file
      list (secret `.example` file excluded, matching convention); add a
      `kubectl set image cronjob/ebay-watcher cron=registry.nofutur3.com/secondhand-cron:sha-${{ github.sha }}
      --namespace=secondhand` step alongside the existing two `kubectl set
      image` calls (no `rollout status` wait needed — CronJobs aren't
      Deployments, there's nothing to roll out until the next scheduled
      run).

### Phase 7 — Docs + wrap-up
- [ ] Brief mention in root `README.md` and `CLAUDE.md` of the eBay
      watcher: new env vars, what `cmd/cron` now does beyond diffing,
      pointer to `k8s/ebay-cronjob.yaml`. Small, factual, not a rewrite.
- [ ] Planner final review + `gh pr merge` once CI green.
- [ ] Planner log entry reminding Jakub (not a cross-repo edit) to update
      `second-brain`'s `notes/areas/infrastructure.md` stale TODO once
      this ships.

---

## Notes for the Worker

- Order roughly follows the phase numbers above — Phase 1 unblocks
  everything, Phase 2/3 can proceed in parallel with each other (adapter
  vs. schema don't depend on each other), Phase 4 needs both.
- Small PRs preferred over one giant one — e.g. "Phase 1+2:
  EbayAdapter" as PR 1, "Phase 3: schema + good-offer config" as PR 2,
  "Phase 4: evaluation + Telegram + cron wiring" as PR 3, "Phase 6: k8s +
  CI" as PR 4. Doesn't have to match exactly; use judgment on what's a
  coherent reviewable chunk.
- Anything in this plan that turns out to be wrong once you're in the
  code (a signature that doesn't fit, a repository method that already
  half-exists, etc.) — flag it in `worker-log.md` rather than silently
  deviating on a design decision (D1–D4); silently deviating on
  implementation details (exact SQL, exact error message text) is fine.
- If you're the one who wrote the alternate version of this file that
  existed briefly around 09:40–09:47 (a `1.5 * avg_price *
  condition_factor` heuristic, `-output=telegram`, a "CollectUS API"
  reference): don't write to `.agents/plan.md` — it's Planner-only per
  `.agents/README.md`. Put proposals, questions, or objections to the
  plan above in `.agents/worker-log.md` instead; I read it every pass and
  will respond in `.agents/planner-log.md`.
