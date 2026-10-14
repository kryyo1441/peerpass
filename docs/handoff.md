# PeerPass — Handoff

Read this first to resume work. It points to everything else; it doesn't
duplicate content.

## Current state

- **Milestone:** M1–M3 are done and merged to `main`. **M4 (WebRTC peer
  connection + data channel test message) is explicitly paused at the
  project owner's request (2026-10-14) — not started.** See
  [`plan.md`](plan.md) for the full roadmap and [`progress.md`](progress.md)
  for the dated log of what's actually been done.
- **Branch:** none in progress. `main` has everything through M3, including
  a review pass that fixed several real bugs found after the initial M3
  "done" (leave-cleanup ordering, stuck-peer eviction, disconnect
  detection, a missing mutex, peer-list snapshot events) — see
  `decisions.md` and `progress.md` for what was actually wrong and fixed.
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

When M4 resumes: establish a WebRTC peer connection and exchange a test
message over a data channel. The existing signaling relay
(`Hub.Relay`/`signalclient.Relay`, already carrying opaque
`json.RawMessage` payloads) is what will carry SDP offers/answers and ICE
candidates — this is designed in already (ADR-003), not something to
retrofit. Evaluate Pion WebRTC for real before committing (ADR-004 is
currently unverified placeholder only).

**Carry this decision into M4 when it starts (discussed 2026-10-14, not
yet implemented):** wire a public STUN server into the peer connection's
ICE config from the beginning, rather than adding it in M5. It's a config
value at connection-creation time, not something to retrofit. M4 does not
need to determine whether this network actually needs STUN to connect
(likely moot on localhost/LAN) — just that ICE candidate gathering is
wired up and exercised. Whether STUN is sufficient or a symmetric NAT
forces TURN is M5's empirical question, not M4's. See `plan.md`'s open
questions for the fuller note.

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
