# PeerPass — Handoff

Read this first to resume work. It points to everything else; it doesn't
duplicate content.

## Current state

- **Milestone:** M3 (minimal signaling, N-party room model) is done and
  merged to `main`. M4 (WebRTC peer connection + data channel test message)
  is next — not started. See [`plan.md`](plan.md) for the full roadmap and
  [`progress.md`](progress.md) for the dated log of what's actually been
  done.
- **Branch:** `feat/signaling` has the M3 work, not yet merged to `main` —
  needs review/approval before merging.
- **Stack decision:** Wails v2, Go backend, Svelte+TS frontend,
  `coder/websocket` for signaling transport (ADR-007). Full reasoning in
  [`decisions.md`](decisions.md).
- **Dev machine:** Arch Linux, webkit2gtk-4.1 — confirmed working, but
  **every `wails` command needs `-tags webkit2_41`**. See `README.md`.
- **Security:** no auth until M9 — see `README.md`'s security section and
  ADR-005. Signaling server binds to `127.0.0.1` by default.

## Where things live

- [`decisions.md`](decisions.md) — engineering decision log (ADRs).
- [`plan.md`](plan.md) — milestone roadmap, current milestone, open questions.
- [`progress.md`](progress.md) — dated log of what was actually done.
- [`architecture.md`](architecture.md) — current components.
- [`learning-log.md`](learning-log.md) — concepts/mistakes/debugging lessons.
- `../README.md` — setup/build/test instructions, security status.

## What needs to happen next

1. Review `feat/signaling` (commits, diff, docs) and decide whether to merge
   to `main`.
2. Start Milestone 4: establish a WebRTC peer connection and exchange a test
   message over a data channel. The existing signaling relay
   (`Hub.Relay`/`signalclient.Relay`, already carrying opaque
   `json.RawMessage` payloads) is what will carry SDP offers/answers and ICE
   candidates — this is designed in already (ADR-003), not something to
   retrofit. Evaluate Pion WebRTC for real before committing (ADR-004 is
   currently unverified placeholder only).

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
- **Signaling hosting (relevant to M5):** ADR-008 is a proposed/unverified
  placeholder. No free-tier host has been chosen or deployed. Only runs
  locally so far.
