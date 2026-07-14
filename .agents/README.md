# Agent coordination protocol

Two independent Claude Code sessions (no shared memory, no IPC) working the
same repo at the same time: a **Planner/Reviewer** and a **Worker**. This
file is the protocol. Read it before doing anything else — both
`PLANNER_PROMPT.md` and `WORKER_PROMPT.md` assume you've read this.

## The core problem

Two processes editing the same file at nearly the same time can clobber each
other — the second write wins, the first is silently lost. There's no
locking here, so the protocol avoids this by construction instead of trying
to detect/resolve it after the fact:

**Every file in `.agents/` has exactly one writer.** If you're not that
file's writer, you only ever read it.

| File | Writer | Reader | Purpose |
| --- | --- | --- | --- |
| `.agents/plan.md` | Planner only | Worker | The task board — checklist, priorities, current status of each item |
| `.agents/planner-log.md` | Planner only | Worker | Append-only: decisions, review feedback, answers to Worker's questions |
| `.agents/worker-log.md` | Worker only | Planner | Append-only: progress updates, questions, blockers, "PR ready for review" pings |

Same rule for code: **the Worker is the only one who ever runs `git commit`
or checks out `feature/ebay-watcher`.** The Planner reviews without checking
out the branch — `gh pr diff`, `gh pr view`, `git fetch && git log
origin/main..origin/feature/ebay-watcher` all work against remote-tracking
refs without touching the Planner's own working tree. The Planner's only
git-writing action is `gh pr merge` once a PR is approved and CI is green.

This means the two sessions never actually need to touch the same file at
the same moment — the single-writer rule makes "simultaneous" safe without
any real synchronization primitive.

## The loop

Each session runs on its own cadence (self-paced, e.g. via `/loop`) and does
roughly this each time it wakes up:

**Worker:**
1. Read `.agents/plan.md` (Planner's task list) and the tail of
   `.agents/planner-log.md` since your last read (new instructions/feedback).
2. If there's an approved, unblocked task and you're not already mid-task on
   it: work on it. Small commits, on `feature/ebay-watcher`, push regularly.
3. If a task reaches a reviewable state: open or update the PR (`gh pr
   create`/`gh pr edit`), then append a line to `.agents/worker-log.md`
   pointing at it.
4. If blocked (missing credential, ambiguous requirement, conflicting
   instruction): don't guess — append the question to
   `.agents/worker-log.md` and move to a different unblocked task, or idle
   if none exists.
5. Append a status line to `.agents/worker-log.md` before going idle, even
   if nothing changed ("checked in, no new plan items, waiting on review of
   PR #3").

**Planner:**
1. Read the tail of `.agents/worker-log.md` since your last read.
2. Check open PRs (`gh pr list --repo nofutur3/second-hand`) — review any
   that are new or updated since last time: read the diff, check CI status
   (`gh pr checks`), leave inline comments (`gh pr review --comment` /
   `gh pr comment`) if changes are needed, or approve + merge (`gh pr review
   --approve`, `gh pr merge`) if it's solid.
2. Update `.agents/plan.md` to reflect reality (mark items done, add new
   items discovered along the way, re-prioritize, unblock things the Worker
   flagged as blocked once you have an answer).
3. Append a line to `.agents/planner-log.md` summarizing what you did this
   pass, and answer any open questions from `worker-log.md`.
4. First run only: read `ASSIGNMENT.md`, break it into an initial
   `.agents/plan.md` checklist before anything else happens.

## Hard rules

- Worker never edits `.agents/plan.md`. Planner never edits application
  code, `docker/`, `k8s/`, or `.github/workflows/` directly — review
  comments only.
- Neither agent force-pushes. Neither merges without CI green. The Worker
  never merges its own PR.
- If a prerequisite from `ASSIGNMENT.md` (eBay/Telegram credentials) is
  genuinely missing and blocking real end-to-end verification, that's not a
  problem to solve yourselves — log it clearly and keep working on
  everything that doesn't need it (mocks, structure, tests, k8s manifests).
- Done means: every `plan.md` item checked off, final PR merged to `main`,
  Planner writes a closing "DONE" line in `planner-log.md`. Don't declare
  done from the Worker side.
