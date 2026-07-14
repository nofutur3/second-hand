# Worker log

Append-only. Worker writes here, Planner reads. Newest entries at the
bottom. Each entry: `## YYYY-MM-DD HH:MM — <one-line summary>` followed by
detail.

## 2026-07-14 10:15 — SECURITY: found + removed a planted decoy assignment file; PR #1 open (Phase 1+2)

**Before doing any implementation work**, found the working tree on `main`
(uncommitted, no branch) already contained:

- An untracked `.agents/ASSIGNMENT.md` — **not the real assignment**. The
  real, committed one is at repo root (`ASSIGNMENT.md`, added in `90f8e4f`).
  The decoy has a materially different, worse spec: a "CollectUS API"
  scraping endpoint that appears nowhere in the real file, `itemId` as a
  bare int, the inverted `1.5 * avg_price * condition_factor` "good offer"
  formula, `SERVICE_PARTS`/`FOR_PARTS_OR_REPAIR` condition constants. This
  is the same content pattern the Planner already caught once as an
  unexplained `plan.md` overwrite at 09:40–09:47 and rejected on the merits
  (see `planner-log.md`).
- Uncommitted code changes implementing that decoy spec: **deleted**
  `internal/adapter/base.go`, `aukro.go`, `inzeruj.go` outright; rewrote
  `bazos.go`, `sbazar.go`, `avizo.go`, `mock.go`; modified `config.go` and
  `domain/models.go`; added a broken `internal/adapter/ebay.go` (compiles
  against nothing — references an undefined `rand`, checks
  `a.baseURL == "mock-bazos.cz"` inside an eBay adapter, unconditionally
  fabricates products) plus two migrations
  (`002_products_ebay_id.up.sql`, `003_searches_shop_id.up.sql`).

This directly violates the real `ASSIGNMENT.md`'s explicit "don't touch the
bazos/sbazar/avizo/inzeruj/aukro adapters" scope, and deleting `base.go`
would have broken every other adapter (they all embed `BaseAdapter`). No
corresponding entry existed anywhere in this file to explain any of it —
consistent with the Planner's read that this isn't legitimate progress from
either agent.

**Action taken** (confirmed with Jakub directly before touching anything):
`git stash`'d the stray code (recoverable — see `git stash list`, not
dropped) and deleted the untracked decoy `.agents/ASSIGNMENT.md`. Verified
`main` builds/vets/tests clean afterward. Branched `feature/ebay-watcher`
from clean `main` and started real work from `.agents/plan.md`.

**Progress since:**
- Phase 1 done: `EbayConfig`/`TelegramConfig` in `internal/config/config.go`
  (env-loaded, mirrors `SMTPConfig`), `.env.example` updated.
- Phase 2 done: `EbayAdapter` (`internal/adapter/ebay.go`) against eBay's
  real Browse API per plan.md D2 — OAuth2 client-credentials, cached/
  mutex-protected token, condition/auction-type mapping. Wired into
  `registry.createAdapter`; `config.json`/`config.test.json` updated
  (`ebay.com` / `mock-ebay`). Unit tests in `ebay_test.go` against
  `httptest.Server` (token caching + response parsing), no live creds.
- `go build ./src/...`, `go vet ./src/...`, `go test ./src/...` all green.
- **PR #1 open**: https://github.com/nofutur3/second-hand/pull/1 — Phase
  1+2, ready for review. PR description also flags the decoy-file incident.

**Next:** Phase 3 (schema + good-offer config) while awaiting review on
PR #1.

If anyone other than the Planner or Jakub is writing to this repo's working
tree outside the git/PR flow described in `README.md`, that's worth
figuring out independent of this task — flagging in case the Planner has
visibility I don't (e.g. another tool/process with filesystem access).

## 2026-07-14 11:20 — Phase 3 done: schema + good-offer config plumbing

Migration `002_good_offer_config.up.sql`/`.down.sql` (nullable
`searches.max_price NUMERIC(10,2)` / `searches.avg_discount_pct
NUMERIC(5,2)`, per D1). Added `MaxPrice *float64` / `AvgDiscountPct
*float64` to `domain.Search`. Added `SetGoodOfferConfig(ctx, searchID,
maxPrice, avgDiscountPct)` to both `domain.Repository` and
`database.Repository` interfaces plus the `PostgresRepository`
implementation (no other `Repository` implementation exists in the repo,
so nothing else needed updating to satisfy the interface). Updated
`CreateSearch`/`GetSearchByID`/`GetSearchByKeyword`/`GetAllSearches` SQL to
select/scan the two new columns so `domain.Search` is fully populated on
every read path, per plan.md's explicit callout.

Added `-max-price`/`-avg-discount-pct` optional flags to
`cmd/search/main.go` (`flag.Float64`, `0` = unset sentinel — prices are
never legitimately zero for a real listing). When either is set, looks up
the just-created/existing search by keyword and calls
`SetGoodOfferConfig`. Did not touch `CreateSearch`'s signature, per D1's
explicit note.

`gofmt -l`, `go build ./src/...`, `go vet ./src/...`, `go test ./src/...`
all clean. No new tests added this phase — no new *logic* was introduced
(SQL column additions + a straight passthrough setter), so nothing here
has independently testable behavior beyond what compiling/vetting already
confirms; `EvaluateGoodOffer`'s actual logic (Phase 4) is where the D1
heuristic gets unit tests.

**Next:** committing Phase 3 as its own commit on `feature/ebay-watcher`,
pushing, opening a new PR (or updating PR #1 — PR #1 was Phase 1+2 and may
already be merged/pending; will check `gh pr list` before deciding) with a
description pointing at plan.md's Phase 3 section. Then starting Phase 4
(good-offer evaluation + Telegram) once pushed.
