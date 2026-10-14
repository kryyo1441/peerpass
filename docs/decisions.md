# PeerPass — Engineering Decision Log

This file lives in `docs/` (not root) per project owner's instruction. `docs/handoff.md`
and `docs/plan.md` link here rather than duplicating it.

---

## ADR-001: Desktop shell — Wails v2 over Wails v3 beta

- **Date:** 2026-10-14
- **Status:** Accepted

### Context and problem

PeerPass must ship as a native desktop app, not a website. We need a Go-backed
desktop shell framework. Wails has two active lines: v2 (long-term stable) and
v3 (beta).

### Verification performed

- WebSearch + WebFetch of `github.com/wailsapp/wails/releases`: confirmed v3 is at
  `v3.0.0-beta.29` as of 2026-10-13, explicitly pre-release.
- Attempted WebFetch of `wails.io` and `v3.wails.io` official docs — both returned
  HTTP 403 (bot-blocked). **Official-docs platform/build-requirement details for
  either version are not independently verified**; this is a real gap, not an
  oversight, and is flagged here rather than filled in from training-data memory.
- System check on the actual dev machine (Arch Linux): `gcc 16.2.1`, `gtk+-3.0
  3.24.52` present, `webkit2gtk-4.1 2.54.1` present, **`webkit2gtk-4.0` absent**.
  Arch's AUR `wails` package is at `2.11.0-1`, and Arch does not ship
  webkit2gtk-4.0 at all — so *some* mechanism in the current v2 toolchain must
  already handle 4.1-only systems. WebSearch found GitHub issue history describing
  this as historically requiring a `-tags webkit2_41` build flag, but this was
  **not confirmed from an official source** — to be confirmed empirically in
  Milestone 2 by actually running `wails doctor` / `wails build` on this machine.

### Alternatives considered

| Option | Pros | Cons |
|---|---|---|
| Wails v2 | Stable, production-proven, AUR-packaged on Arch, large existing user base/examples | In maintenance mode, not actively gaining new features; possible future migration cost if we ever want v3-only capabilities |
| Wails v3 beta | Newer API, actively released (beta.29 as recently as yesterday relative to this decision) | Explicitly pre-release; breaking changes possible between betas; riskier for a project being used to learn Go |

### Decision

**Wails v2**, chosen directly by the project owner after being presented this
trade-off via AskUserQuestion.

### Consequences

- Must confirm webkit2gtk-4.1 compatibility empirically in Milestone 2 (see open
  item above). If `wails doctor`/`wails build` fails on this machine, the fix
  (build tag, package symlink, or otherwise) and its actual outcome will be
  recorded as an update to this ADR, not silently patched.
- A future v2→v3 migration is accepted as a deferred cost, not ruled out.

### Update — 2026-10-14, webkit2gtk-4.1 finding confirmed

Ran it for real: `wails doctor` on this machine reports `libwebkit: Not Found`
even with webkit2gtk-4.1 2.54.1 installed. (Inference, not directly confirmed
from Wails source: the doctor check likely looks for the old `webkit2gtk`
4.0 package name specifically and doesn't recognize 4.1 — plausible but not
verified by reading the doctor command's code.) This is a **doctor-command
limitation, not an actual build blocker** — confirmed both ways empirically:
- `wails build` **without** the tag fails with a real compile error:
  `pkg-config --cflags ... webkit2gtk-4.0` → `Package 'webkit2gtk-4.0' was
  not found`, `exit status 1`.
- `wails build -tags webkit2_41` completes successfully
  (`Built '.../build/bin/peerpass' in 16.6s`, reproduced again after fixes).

So Wails v2 on Arch with only webkit2gtk-4.1 works, but **every `wails
build`/`wails dev` call in this project must pass `-tags webkit2_41`**, and
this has now been verified both ways (fails without it, works with it), not
just assumed.

Separately, the `svelte-ts` template's `frontend/package.json` pins
`"vite": "^8.0.13"`, but `@sveltejs/vite-plugin-svelte@^6.0.0` only supports
vite `^6 || ^7` — `npm install` failed with an ERESOLVE conflict out of the
box. Fixed by pinning `vite` to `^7.0.0` in `frontend/package.json`. This is a
real upstream version mismatch in the current Wails v2.16.0 template as of
2026-10-14, not a mistake on our end.

A third mismatch: the template's `frontend/src/main.ts` instantiates the root
component with `new App({ target: ... })`, which is the Svelte 4 API. The same
template pins `"svelte": "^5.55.7"`, and Svelte 5 removed the `new Component()`
constructor entirely — it throws `component_api_invalid_new` at runtime. This
was invisible in `wails build`/`svelte-check` (both succeeded) and only
surfaced as a **blank window** at runtime: the project owner reported the
native window opened blank mid-session, and the `wails dev` browser view at
`localhost:34115` was independently confirmed blank via screenshot, with the
`component_api_invalid_new` exception visible in the browser console. Fixed by
switching to Svelte 5's `mount(App, { target: ... })` API.

Verified after the fix, in two separate ways:
- Browser dev view (`localhost:34115`): typed a name via browser automation,
  clicked Greet, confirmed `Hello PeerPass, It's show time!` rendered — the
  Go↔Svelte IPC call was actually exercised, not just compiled.
- Native window: ran the packaged `build/bin/peerpass` binary directly and
  captured a real screenshot of the live Hyprland/Wayland display (via
  `grim`, cropped to the window's reported geometry from `hyprctl clients
  -j`) — confirms the native webkit2gtk window now renders the UI instead of
  being blank. No input-simulation tool (`ydotool`/`wtype`) was available in
  this environment to also click Greet *inside* the native window, so that
  specific click was only exercised through the browser dev view, which runs
  the identical frontend bundle and Go backend over the same IPC bridge —
  not a separate mock.

**Takeaway recorded for future milestones:** a successful `go vet`/`wails
build`/`svelte-check` does not prove the UI renders. Milestone 2's
"verify by actually running it" step caught a bug none of the static checks
did — this is why the plan requires driving `wails dev` with browser
automation rather than stopping at a green build.

### Revisit when

Wails v3 reaches a stable (non-beta) release, or v2 stops receiving security
fixes.

---

## ADR-002: Signaling/room model designed for N-party from day one

- **Date:** 2026-10-14
- **Status:** Accepted; implemented in Milestone 3 (`internal/signaling`)

### Context

MVP only needs 1:1 input sharing and handoff. Video streaming and multi-peer
rooms are deferred, not cancelled, per the project brief.

### Decision

Room state, peer identity, join/leave, and the control-arbitration state machine
will be modeled as N-party (room code, peer list, host identity, per-peer role)
from the first implementation in Milestone 3, even though only two peers will
actually connect for a while. Retrofitting N-party semantics onto a hardcoded
1:1 design later is significantly harder than designing for it upfront.

### Consequences

- Milestone 3's room/session types carry a peer list, not a single `peerID`
  field, from the start.
- No premature building of video fan-out or SFU/mesh logic — only the data
  model and arbitration state machine need to tolerate N peers now.

### Update — 2026-10-14, Milestone 3: host-leave semantics decided

Implementing `Hub.Leave` required deciding what happens when a peer
disconnects. Two cases:

- **Non-host leaves:** removed from the room, remaining peers get a
  `peer-left` notification, room stays open. Uncontroversial.
- **Host leaves:** the room closes entirely and every remaining peer gets
  `room-closed`, rather than e.g. promoting another peer to host. Chosen
  because the host *is* the game machine in this product — if it's gone,
  there's nothing left for anyone else to connect to or control. A
  host-promotion model would be actively misleading (the new "host" still
  wouldn't be running the game). Revisit only if a future milestone changes
  what "host" means (e.g. a dedicated server architecture, which is not
  currently planned).

Verified in `internal/signaling/room_test.go` (`TestLeave_HostClosesRoom`,
`TestLeave_NonHost`) and in the 3-client integration test
(`TestLeave_BroadcastsAndClosesHost`).

### Update — 2026-10-14, Milestone 3: room cap set to 8

`MaxPeersPerRoom = 8` in `internal/signaling/room.go`. Arbitrary, not derived
from any load testing — just a reasonable bound for "host + a handful of
viewers" per room. **This caps peers within a room, not how many rooms a
client can create** — `CreateRoom` itself is currently unbounded, so a
malicious client hammering room creation is not actually stopped by this
cap. That's a real gap, tracked under ADR-005 (no auth/abuse protection
until Milestone 9) rather than patched piecemeal now. Revisit the peer cap
itself once Phase D (multi-viewer video) picks a real mesh/SFU topology and
room-size limit, which will likely replace this number with something
load-tested.

---

## ADR-003: Peer connection layer designed to add media tracks later

- **Date:** 2026-10-14
- **Status:** Accepted (design commitment, implementation lands in Milestone 4)

### Decision

The WebRTC peer-connection abstraction will not assume a data channel is the
only transport that will ever exist, so that a video media track (Phase D) can
be added without restructuring the connection layer. This does not mean
building video support now — only not architecting data-channel-only.

---

## ADR-004 (provisional, unverified): Pion WebRTC as the Go WebRTC implementation

- **Date:** 2026-10-14
- **Status:** Proposed, unverified — this is a placeholder for Milestone 4

### Context

The brief proposes Pion WebRTC as the established Go WebRTC implementation to
use, rather than hand-rolling ICE/SDP/DTLS/SRTP.

### Status of verification

**Not yet evaluated.** No compatibility check, no version check, no API read
has been performed as of this entry. This ADR exists only to record the
proposed direction and to make clear it is unverified until Milestone 4, when
it will be properly evaluated against the brief's requirement to check current
documentation and compatibility before committing.

### Update — 2026-10-14, M4 paused; STUN design decision recorded ahead of it

Project owner asked to pause before implementing M4 and discuss STUN
integration first. Decision for when M4 resumes: a public STUN server
(e.g. `stun:stun.l.google.com:19302`, or self-hosted later) goes into the
ICE server configuration at the moment the peer connection is created —
whichever Pion type corresponds to `RTCConfiguration.iceServers`. This is
not something to add after the fact; the connection-establishment code
needs to exist with ICE servers configured from its first version, since
retrofitting it later means touching code that didn't expect ICE
candidates to exist at all.

This is a design/config decision, not an empirical one: M4 itself does
not need to determine whether this machine's actual network requires STUN
to connect (plausibly moot entirely on localhost/LAN, where M4 is tested).
Whether STUN is sufficient, or a symmetric/carrier-grade NAT forces a TURN
fallback, is explicitly Milestone 5's empirical question per the brief —
tested for real against actual network conditions, not assumed here. TURN
itself stays fully out of scope until M5.

---

## ADR-006: Svelte + TypeScript for the frontend

- **Date:** 2026-10-14
- **Status:** Accepted, per the brief's direct instruction

### Context

The brief specifies Svelte with TypeScript "unless there is a compelling
documented reason to choose another frontend."

### Verification performed

None beyond using it. No benchmark or comparison against React/Vue/vanilla
was run — there was no documented reason surfaced to deviate from the brief's
instruction, so Svelte+TS was used as specified rather than independently
re-evaluated. The one real friction point found was unrelated to Svelte
itself: the Wails v2.16.0 `svelte-ts` template pins Svelte 5.55.7 but its own
`main.ts` still used the Svelte 4 instantiation API (see the Milestone 2
update under ADR-001) — a template/version-skew bug, not a reason to
reconsider the framework choice.

### Decision

Svelte + TypeScript, as specified. Not revisited unless a concrete problem
surfaces.

---

## ADR-005: Interim trust model (no auth until Milestone 9)

- **Date:** 2026-10-14
- **Status:** Accepted (policy, applies starting whenever the first networked
  build exists — Milestone 3 onward)

### Decision

Until Milestone 9 (authentication/session authorization) is complete, PeerPass
has no authentication and no session authorization. Any peer that can reach the
signaling server and knows a room code can join and potentially send input.
This must be stated plainly in `README.md` from the first working networked
build, and the application must never be described as safe to expose to the
open internet before Milestone 9 lands.

### Update — 2026-10-14, Milestone 3: first networked build lands

This ADR's trigger condition is now live: `internal/signalserver` is a real
networked server. `README.md`'s security section has been updated from "no
milestone involving networking has landed yet" to the actual interim trust
model. The signaling binary (`cmd/signal`) binds to `127.0.0.1` by default
and requires an explicit `-addr` flag to listen on anything else, so the
no-auth gap doesn't accidentally become internet-reachable by default.

---

## ADR-007: Signaling transport — coder/websocket over gorilla/websocket

- **Date:** 2026-10-14
- **Status:** Accepted

### Context and problem

The signaling server needs a persistent, bidirectional, low-latency channel
so peers can be pushed room/peer events (joins, leaves, relayed SDP/ICE from
M4) without polling. The standard library has no WebSocket server, so this is
a real dependency decision (brief §12: explain what it does, why it's needed,
and the alternatives before adding one).

### Verification performed

- `go list -m -versions` on both candidates: `github.com/gorilla/websocket`
  tops out at `v1.5.3`; `github.com/coder/websocket` is at `v1.8.15`.
- WebFetch of `pkg.go.dev/github.com/coder/websocket`: v1.8.15 published
  2026-06-15 (recent, actively released), explicitly documents **native
  concurrent writes** ("All methods may be called concurrently except for
  Reader and Read"), zero dependencies, passes the autobahn test suite.
- WebSearch on gorilla/websocket vs. nhooyr/coder maintenance status:
  gorilla has an open "no maintainer" issue with sparse recent commits. The
  search also surfaced a claim that nhooyr/coder was dormant since 2021 --
  **this claim is contradicted by the pkg.go.dev version/date check above**
  and was treated as stale/wrong rather than trusted, since it conflicts
  with primary-source evidence (the actual release list).
- WebFetch of `pkg.go.dev/github.com/gorilla/websocket` (checked during a
  later review pass, not assumed from memory): confirms gorilla does **not**
  support concurrent writes -- "Connections support one concurrent reader
  and one concurrent writer. Applications are responsible for ensuring that
  no more than one goroutine calls the write methods ... concurrently,"
  with Close/WriteControl as the only exception.

### Alternatives considered

| Option | Pros | Cons |
|---|---|---|
| coder/websocket | Actively released, zero deps, native concurrent writes (no manual write-mutex needed per connection), context-based API fits Go idioms | Newer/less ubiquitous name recognition than gorilla |
| gorilla/websocket | Extremely widely used historically, mature protocol implementation | Maintenance in question (open "no maintainer" issue); does **not** support concurrent writes on one connection without external synchronization, which this project's broadcast-while-relaying pattern needs |
| stdlib-only (SSE + POST) | No dependency at all | Awkward: have to correlate each POST with the right SSE stream per peer; no real benefit over a WebSocket library for this use case |

### Decision

`github.com/coder/websocket` v1.8.15. The deciding factor was concurrent
write support: the Hub's broadcast pattern (notify existing peers while
possibly also relaying a targeted message) means multiple goroutines may
need to write to the same connection around the same time, and coder/
websocket handles that internally instead of requiring a per-connection
write mutex we'd have to get right ourselves.

### Consequences

- One external dependency added, justified above.
- If coder/websocket's maintenance status changes for the worse, gorilla/
  websocket plus a manual write-mutex per connection is the fallback,
  not a full redesign.

---

## ADR-008 (proposed, unverified): Signaling server hosting

- **Date:** 2026-10-14
- **Status:** Proposed, not yet implemented or verified — local-only for now

### Context

The brief requires near-zero hosting cost and prefers a free-tier host over
requiring the game-hosting player to configure port forwarding (Fly.io,
Render, Oracle Cloud free tier, or similar).

### Status of verification

**Not yet evaluated.** Milestone 3 only runs `cmd/signal` locally
(`127.0.0.1:8080`) for development and the two-instance manual test. No
free-tier host has been signed up for, no deployment has happened, and no
current pricing/limits have been checked. This ADR is a placeholder for
Milestone 5 (cross-network connectivity testing), which is also where a real
deployment would happen -- and only with the project owner's explicit
permission, since deploying to a third-party service is a remote, externally
visible action, not a local one.
