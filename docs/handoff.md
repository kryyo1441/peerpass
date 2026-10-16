# PeerPass — Handoff

Read this first to resume work. It points to everything else; it doesn't
duplicate content.

## Current state

- **Milestone:** M1–M3 are done and merged to `main`. **M4 (WebRTC peer
  connection + data channel test message) is done (2026-10-15)**, on
  branch `feature/m4-webrtc`, not yet merged to `main`. See
  [`plan.md`](plan.md) for the full roadmap and [`progress.md`](progress.md)
  for the dated log of what's actually been done.
- **Branch:** `feature/m4-webrtc`, done but unmerged -- `main` still only
  has through M3. `main` includes a review pass that fixed several real
  bugs found after the initial M3 "done" (leave-cleanup ordering,
  stuck-peer eviction, disconnect detection, a missing mutex, peer-list
  snapshot events) — see `decisions.md` and `progress.md` for what was
  actually wrong and fixed. M4's own build caught and fixed one more:
  trickled ICE candidates could arrive before the SDP that logically
  preceded them (separate relay messages, no ordering guarantee) —
  `internal/webrtcsignal` now queues and flushes them.
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
- [`run-project.md`](run-project.md) — running the app on a second device
  for M5's real cross-network test (points at the deployed signaling
  server, not localhost).
- `../README.md` — setup/build/test instructions, security status.

## What needs to happen next

M4 is done — see `progress.md`'s 2026-10-15 entries for the full build log
across its 5 parts (Pion verification, `internal/peerconn`,
`internal/webrtcsignal`, Wails integration, the test message). Two
decisions for the project owner before moving on:

1. **Merge `feature/m4-webrtc` to `main`?** Not done automatically.
2. **Frontend display:** the Svelte UI doesn't show WebRTC connection
   state or the test message at all yet — M4 was read as Go-side scope
   (same precedent as M2). Worth doing before M5, or fine to leave for
   later?

After that, M5 is cross-network connectivity: deploy the signaling server
somewhere reachable (ADR-008 is still an unverified placeholder — no host
chosen), then test for real whether the STUN config from M4 is sufficient
across real NATs, or whether a symmetric NAT forces TURN. That's an
empirical question for real devices on real networks, not something to
simulate locally (see the Docker-vs-real-devices discussion from
2026-10-15, not written up elsewhere since it didn't change any decision).

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
