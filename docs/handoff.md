# PeerPass — Handoff

Read this first to resume work. It points to everything else; it doesn't
duplicate content.

## Current state

- **Milestone:** M2 (Wails shell + verified Go↔frontend call) is done. M3
  (signaling, N-party room model) is next — not started. See
  [`plan.md`](plan.md) for the full roadmap and [`progress.md`](progress.md)
  for the dated log of what's actually been done.
- **Branch:** `feat/wails-shell` has the working shell, not yet merged to
  `main` — needs review/approval before merging.
- **Stack decision:** Wails v2 (not v3 beta), Go backend, Svelte+TS frontend.
  Full reasoning in [`decisions.md`](decisions.md) ADR-001, including two real
  upstream template bugs found and fixed (vite version pin, Svelte 5 `mount()`
  API).
- **Dev machine:** Arch Linux, webkit2gtk-4.1 — confirmed working, but
  **every `wails` command needs `-tags webkit2_41`**. See `README.md`.

## Where things live

- [`decisions.md`](decisions.md) — engineering decision log (ADRs).
- [`plan.md`](plan.md) — milestone roadmap, current milestone, open questions.
- [`progress.md`](progress.md) — dated log of what was actually done.
- [`architecture.md`](architecture.md) — current components.
- [`learning-log.md`](learning-log.md) — concepts/mistakes/debugging lessons.
- `../README.md` — setup/build/test instructions, security status.

## What needs to happen next

1. Review `feat/wails-shell` (commits, diff, docs) and decide whether to
   merge to `main`.
2. Start Milestone 3: minimal signaling flow. Model room state, peer list,
   host identity, and join/leave as N-party from the start (ADR-002), even
   though only two peers connect initially. Explain to the project owner why
   signaling is needed for a P2P app before writing it.

## Open questions (don't lose these)

- **Controller handoff exact behavior (relevant to M8):** not yet discussed
  with the project owner. Needs: does the host approve every request
  explicitly, or can peers queue? What happens on simultaneous requests?
  Revisit when M8 planning starts, not before.
- **Crossplay (Windows/macOS/Linux):** confirmed by the project owner
  (2026-10-14) as a real future requirement, on top of developing on Arch
  Linux now. Doesn't change current scope — it's why platform-specific code
  (input injection in M7, capture in M10) must sit behind interfaces from the
  start. No Windows/macOS build has been attempted yet.
