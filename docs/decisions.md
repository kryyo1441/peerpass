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
even with webkit2gtk-4.1 2.54.1 installed — `wails doctor`'s dependency check
is hardcoded to look for the old `webkit2gtk` (4.0) package name and doesn't
recognize 4.1. This is a **doctor-command limitation, not an actual build
blocker**: `wails build -tags webkit2_41` from the repo root completed
successfully (`Built '.../build/bin/peerpass' in 16.6s`). So Wails v2 on Arch
with only webkit2gtk-4.1 works, but **every `wails build`/`wails dev` call in
this project must pass `-tags webkit2_41`** or the Go side won't find the
right webkit headers. This should be captured in a Makefile/README build
instructions, not left as something to remember by hand.

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
surfaced as a **blank window** when actually running the app (both the native
window and the `wails dev` browser view at `localhost:34115` were blank,
confirmed via screenshot and console errors). Fixed by switching to Svelte 5's
`mount(App, { target: ... })` API. Verified after the fix: typed a name into
the running `wails dev` session via browser automation, clicked Greet, and
confirmed `Hello PeerPass, It's show time!` rendered — i.e. the Go↔Svelte call
was actually exercised, not just compiled.

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
- **Status:** Accepted (design commitment, not yet implemented — implementation
  lands in Milestone 3)

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
