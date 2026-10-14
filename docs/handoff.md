# PeerPass — Handoff

Read this first to resume work. It points to everything else; it doesn't
duplicate content.

## Current state

- **Milestone:** M2 (Wails shell + verified Go↔frontend call) — in progress.
  M1 (foundation) is done. See [`plan.md`](plan.md) for the full roadmap and
  [`progress.md`](progress.md) for the dated log of what's actually been done.
- **Repo:** `main` has one commit (`.gitignore`). `docs/` scaffold is being
  added. No application code exists yet.
- **Stack decision:** Wails v2 (not v3 beta), Go backend, Svelte+TS frontend.
  Full reasoning and verification gaps in [`decisions.md`](decisions.md)
  ADR-001.
- **Dev machine:** Arch Linux, webkit2gtk-4.1 present (not 4.0) — unresolved
  question of whether Wails v2 builds cleanly on this is being tested live in
  M2, not assumed. See ADR-001.

## Where things live

- [`decisions.md`](decisions.md) — engineering decision log (ADRs).
- [`plan.md`](plan.md) — milestone roadmap, current milestone, open questions.
- [`progress.md`](progress.md) — dated log of what was actually done.
- [`architecture.md`](architecture.md) — current components (stub until M2
  produces something to describe).
- [`learning-log.md`](learning-log.md) — concepts/mistakes/debugging lessons.

## What needs to happen next

1. Install Wails v2 CLI, run `wails doctor` on this Arch machine — resolves
   the webkit2gtk-4.1 open question in `decisions.md` ADR-001 for real.
2. Scaffold `svelte-ts` template on branch `feat/wails-shell`.
3. Add one real Go-bound method called from the frontend, prove it works via
   `wails dev` + browser automation (not just "it compiled").
4. Run `gofmt`/`go vet`/`go test`/frontend type-check/`wails build`, commit in
   logical groups, report status — do not merge to `main` without explicit
   approval.

## Open questions (don't lose these)

- Exact controller-handoff behavior (needed before M8) — not yet discussed
  with the project owner. See `plan.md` "Open questions".
- Crossplay (Windows/macOS/Linux) confirmed as a real future requirement —
  doesn't change current scope, but platform-specific code must stay behind
  interfaces. See `plan.md`.
