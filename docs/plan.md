# PeerPass — Milestone Roadmap

Full milestone detail lives in the original project brief (not checked in here
verbatim — see `docs/handoff.md` for where to find it if needed). This file
tracks **current status and open questions**, not the full spec.

## Current milestone

**Milestone 2 — Wails desktop shell with a verified Go↔frontend call.**
Milestone 1 (foundation, decisions, docs scaffold) is done.

## Roadmap at a glance

- **Phase A — Foundation**
  - M1: Repo, decisions.md, docs scaffold. ✅ Done (2026-10-14)
  - M2: Wails shell + verified Go↔Svelte call. 🔄 In progress
- **Phase B — Connectivity**
  - M3: Minimal signaling, N-party room model (see ADR-002)
  - M4: WebRTC peer connection + data channel test message (see ADR-003, ADR-004)
  - M5: Cross-network connectivity test, ICE/STUN/TURN behavior, recorded empirically
- **Phase C — Core product (input sharing)**
  - M6: Input-event protocol (validation, serialization, ordering, recovery)
  - M7: Real controller/input integration, platform-specific adapters, tested
    against a logging target before a real game
  - M8: Control-arbitration state machine (request/grant/revoke/host-override/
    disconnect-while-holding-control)
  - M9: Authentication, session authorization, connection lifecycle, security.
    **Until this lands, the app is unsafe for untrusted networks — see ADR-005.**
- **Phase D — Video (deferred, not cancelled)**
  - M10: Capture + encode pipeline, verified locally first
  - M11: Deliver stream to one remote peer, measure real latency/quality
  - M12: Multi-viewer, mesh vs SFU decision recorded
- **Phase E — Quality**
  - M13: Latency/reliability/resource measurement, optimize after correctness

## Open questions

- **Controller handoff exact behavior (relevant to M8):** The brief says to
  clarify this with the project owner before implementing — not yet done.
  Needs answers to: does the host have to explicitly approve every request, or
  can peers queue? What happens on simultaneous requests? Revisit when M8
  planning starts.
- **Crossplay / cross-platform priority:** Project owner confirmed (2026-10-14)
  they want Windows/Linux/macOS crossplay eventually, on top of developing on
  Arch Linux now. This doesn't change M1/M2 scope — it's why platform-specific
  code (input injection in M7, capture in M10) must sit behind interfaces from
  the start rather than being written Linux-only and retrofitted.
- **Wails v2 + webkit2gtk-4.1 on Arch:** see `docs/decisions.md` ADR-001 — to be
  resolved empirically during M2, not assumed.
