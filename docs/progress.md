# PeerPass — Progress Log

Newest entries at top. This is a factual log of what was actually done and
actually verified — not a plan, not a status aspiration.

---

## 2026-10-14 — feat/signaling merged to main; M4 paused

- Merged `feat/signaling` (Milestone 3 + its review-pass fixes) into `main`,
  deleted the branch. `main` now has M1 through M3.
- Project owner asked to pause before Milestone 4 (WebRTC) and discuss STUN.
- Decision recorded for when M4 resumes: wire a public STUN server into the
  peer connection's ICE config from the start, not added later in M5 —
  it's a config value at connection-creation time, not a retrofit. What M4
  itself doesn't need to determine is whether this machine's network
  actually needs STUN to connect (likely moot on localhost/LAN); that
  empirical question, and whether STUN alone suffices or a symmetric NAT
  forces TURN, stays M5's job. See `plan.md` and `handoff.md`.

## 2026-10-14 — Milestone 3 review pass: real bugs found and fixed

A review of the just-completed M3 work (still on `feat/signaling`, not
merged) found several real issues the original implementation and its
tests missed, since `go test -race` only catches races a test actually
exercises, and nothing exercised `app.go`'s concurrent access or a genuine
server crash:

- **Leave-cleanup registered after the welcome write** in `signalserver`:
  a failed write would leave a phantom peer in the Hub forever. Moved the
  `defer` earlier.
- **Stuck peers were silently dropped, not evicted**: `Hub.send`'s comment
  claimed the transport layer handled disconnecting a stalled peer, but
  nothing did. Added `Peer.Done()` + eviction, wired the transport layer to
  actually close the connection when it fires. Added a test
  (`TestSend_EvictsStuckPeer`).
- **No disconnect detection**: `signalclient.readLoop` returned silently on
  any error, so a crashed signaling server left the UI saying "in room"
  forever. Added an `onDisconnect` callback, fired exactly once, distinct
  from the client's own `Close()`.
- **`App.client`/`App.cancelConn` had no lock**: `OnShutdown` runs on a
  different goroutine than Wails-bound calls. Added a mutex.
- **peer-joined/peer-left only sent a delta** (who changed), which could
  get clobbered by a join's own stale snapshot if a notification raced the
  call that triggered it. Changed to a full-list snapshot
  (`PeerListPayload`), consumed by one `signal:peers` frontend event instead
  of two delta events.
- Uppercased the typed room code before joining (codes are meant to be read
  aloud; people will type lowercase).
- Verified the gorilla/websocket "no concurrent writes" claim in ADR-007
  with an actual WebFetch of its docs (previously asserted without
  checking this session). Corrected ADR-002's room-cap note: it bounds
  peers per room, not how many rooms a client can create.
- Corrected `plan.md`/`handoff.md`, which had wrongly stated M3 was merged
  to `main` — it is not; still on `feat/signaling`.
- **Re-ran the full two-process manual check from scratch** against a
  binary built fresh from the post-fix code (the first re-check
  accidentally reused a stale signaling-server process left over from
  earlier in the session, so it wasn't trustworthy evidence for the new
  code — caught this, killed it, rebuilt, and redid the test properly).
  Confirmed, against the known-fresh server: host creates a room, a
  lowercase-typed room code still joins correctly (case-insensitivity fix),
  host leaving auto-returns the peer to the start screen via
  `signal:room-closed`, and killing the signaling server correctly shows
  "Lost connection to the signaling server" and auto-resets via the new
  `signal:disconnected` path -- both previously-untested code paths.
- `go test -race ./...`, `gofmt -l .`, `go vet`, `svelte-check`, and
  `wails build -tags webkit2_41` all re-run clean after every fix.

## 2026-10-14 — Milestone 3 complete: minimal signaling, N-party room model

- Merged `feat/wails-shell` to `main` (Milestone 2), deleted the branch.
- Checked `coder/websocket` vs `gorilla/websocket` before picking a
  dependency: `go list -m -versions` + WebFetch of pkg.go.dev showed
  coder/websocket actively released (v1.8.15, 2026-06-15) with native
  concurrent-write support; gorilla tops out at v1.5.3 with an open
  "no maintainer" issue. A WebSearch result claiming coder/websocket was
  dormant since 2021 was checked against the primary-source version list
  and found to be stale/wrong rather than trusted. Picked coder/websocket.
  See `docs/decisions.md` ADR-007.
- Built `internal/signaling` (Hub/Room/Peer model, no network imports,
  mutex-guarded, server-assigned peer IDs and room codes via `crypto/rand`,
  host-leave closes the room, `MaxPeersPerRoom = 8`). 10 unit tests, all
  passing under `go test -race`.
- Built `internal/signalserver` (WebSocket transport over the Hub, one
  `/ws` endpoint, 16 KiB read limit for future WebRTC SDP/ICE payloads).
  3-client integration test over real `httptest.NewServer` + real
  `coder/websocket` clients — proves the design isn't hardcoded to two
  peers, proves a targeted relay reaches only its recipient, proves a host
  disconnect broadcasts room-closed. All passing under `go test -race`.
- Found and fixed a real bug during testing: a departing peer's own
  deferred cleanup was logging `ErrRoomNotFound`/`ErrPeerNotInRoom` as
  errors when the host's leave had already removed the room first —
  expected race, not a real failure; now only logs genuine errors.
- Built `internal/signalclient` (what the Wails Go backend uses to connect
  — the frontend never opens a socket itself) and `cmd/signal` (thin
  standalone binary, binds to `127.0.0.1:8080` by default).
- Wired `app.go`: `CreateRoom`/`JoinRoom`/`LeaveRoom` bound methods, no
  generic "send anything" binding exposed. Signaling events translate to
  Wails frontend events (`signal:peer-joined`/`peer-left`/`room-closed`)
  via `runtime.EventsEmit`. `OnShutdown` now closes the connection cleanly.
- Replaced the Greet demo UI with a minimal room create/join UI in
  `App.svelte` showing a live peer list.
- **Verified end-to-end with two real, separate OS processes**, not just
  unit tests: ran the standalone signaling server, then two independent
  `wails dev` instances (one from a `git worktree`, to avoid both writing
  to the same build output directory) on different dev-server/Vite ports,
  driven through two browser tabs via `claude-in-chrome`. Host created room
  `3LZV84`; peer joined it; both sides' peer lists updated live via the
  `signal:peer-joined` event; peer left; host's list correctly dropped back
  to one entry via `signal:peer-left`. Screenshotted at each step.
- Updated `README.md`'s security section from "nothing to expose yet" to
  the actual interim trust model (ADR-005), added signaling-server setup
  instructions and `-race` to the testing section.
- Updated `docs/architecture.md`, `docs/learning-log.md` (signaling
  rationale, mutex-vs-goroutine with a Python GIL comparison, why
  non-blocking sends matter, why the room logic has no network imports),
  and `docs/decisions.md` (ADR-007, ADR-008, ADR-002/ADR-005 updates).

## 2026-10-14 — Milestone 2 complete: Wails shell with verified Go↔frontend call

- Installed Wails v2.16.0 CLI (`go install .../wails/v2/cmd/wails@latest`).
- `wails doctor` reported `libwebkit: Not Found` on this Arch machine despite
  webkit2gtk-4.1 being installed — confirmed this is a stale check, not a real
  blocker: `wails build -tags webkit2_41` builds successfully. Documented in
  `docs/decisions.md` ADR-001.
- Scaffolded the `svelte-ts` template on branch `feat/wails-shell` (had to
  scaffold into a scratch dir and copy in, since `wails init` refuses non-empty
  directories).
- Found and fixed two real upstream version mismatches in the current
  template, neither of which were our mistakes:
  - `vite@^8.0.13` pinned in `frontend/package.json` conflicted with
    `@sveltejs/vite-plugin-svelte@^6.0.0`'s peer range (`^6 || ^7`) — pinned
    vite to `^7.0.0`.
  - `frontend/src/main.ts` used the Svelte 4 `new App({...})` constructor,
    which Svelte 5 (also pinned by the template) removed — caused a **blank
    window** at runtime despite all static checks passing. Fixed with
    Svelte 5's `mount()` API.
- Added `Greet(name string) string` method verification: wrote `app_test.go`,
  ran `go test`/`gofmt`/`go vet` (all clean), ran frontend `svelte-check`
  (0 errors after the `main.ts` fix).
- **Verified end-to-end**, not just compiled: drove the browser-reachable dev
  server (`http://localhost:34115`, read from the command's own log output,
  started via `wails dev -tags webkit2_41`) with `claude-in-chrome` — typed
  "PeerPass", clicked Greet, confirmed "Hello PeerPass, It's show time!"
  rendered. Separately ran the packaged native binary directly and captured
  a real screenshot of the live Hyprland display confirming the native
  window renders the UI (not blank) — could not click Greet *inside* the
  native window itself, since no input-simulation tool (`ydotool`/`wtype`)
  was available in this environment; that click was only exercised through
  the browser dev view, which shares the identical frontend bundle and Go
  backend.
- Confirmed the `-tags webkit2_41` requirement both ways, not just assumed:
  `wails build` without the tag fails with a real `webkit2gtk-4.0 not found`
  pkg-config error; with the tag it succeeds.
- Ran `wails build -tags webkit2_41` for a production build; it succeeds.
- Wrote `README.md` with setup instructions (including the required
  `-tags webkit2_41` flag), current capabilities, and the security/interim
  trust-model status.
- Updated `docs/architecture.md` with the actual (small) component diagram,
  `docs/learning-log.md` with Wails/webview/binding concepts and the
  "green build ≠ working UI" lesson, and `docs/decisions.md` ADR-001 with the
  real findings.

## 2026-10-14 — Milestone 1 complete

- Initialized git repo (`main` branch), added `.gitignore`.
- Created `docs/` scaffold: `decisions.md`, `plan.md`, `progress.md` (this
  file), `handoff.md`, `architecture.md` (stub), `learning-log.md` (stub).
- Recorded ADR-001 through ADR-005 in `decisions.md`, including the honest
  gaps in Wails doc verification (403s on official sites) and the unverified
  Pion WebRTC placeholder.
- System check on dev machine: Arch Linux, gcc 16.2.1, gtk+-3.0 3.24.52,
  webkit2gtk-4.1 2.54.1 present, webkit2gtk-4.0 absent, no `wails` CLI
  installed yet.
- Project owner confirmed: Wails v2 (over v3 beta), and wants eventual
  Windows/macOS/Linux crossplay.
- Commit: `63581c8 build: add .gitignore for Go/Wails/frontend artifacts`.

## 2026-10-15 — Milestone 4 resumed, part 1 (Pion verification) done

- Project owner resumed M4 after the 2026-10-14 pause/STUN discussion, and
  split it into 5 parts: (1) verify Pion, (2) peer connection wrapper,
  (3) signaling glue for SDP/ICE over the existing relay, (4) Wails
  integration, (5) data channel test message + self-test.
- Part 1 done: checked `github.com/pion/webrtc/v4` at latest tag `v4.2.23`
  (2026-10-07) against `go.mod` (needs Go 1.24+; project is on 1.25, fine).
  Read the actual API — `Configuration.ICEServers`, `NewPeerConnection`,
  `CreateDataChannel`/`OnDataChannel`, `CreateOffer`/`CreateAnswer`/
  `SetLocalDescription`/`SetRemoteDescription`, `OnICECandidate`/
  `AddICECandidate` — all present, shaped like the browser API, no gaps
  against M4's scope. ADR-004 moved from provisional to accepted. Dependency
  not yet added to `go.mod` — that's part 2.

## 2026-10-15 — Milestone 4 part 2 (peer connection wrapper) done

- Added `github.com/pion/webrtc/v4 v4.2.23` to `go.mod` (the version
  verified in part 1).
- New `internal/peerconn` package: `New()` creates a Pion `PeerConnection`
  with the public STUN server wired into `ICEServers` at construction time
  (per the 2026-10-14 decision, not retrofitted), and
  `CreateDataChannel(pc)` creates the one data channel M4 needs, under a
  shared `DataChannelLabel` constant both sides of the signaling glue
  (part 3) will use. Deliberately thin — the offer/answer/ICE-candidate
  routing logic is signaling glue's job, not this package's.
  `peerconn_test.go` checks STUN is actually present on the built
  connection's config and that the data channel gets the right label.
- `go build ./...` and `go test ./internal/peerconn/...` both pass.

## 2026-10-15 — Milestone 4 part 3 (signaling glue) done

- Branched from part 2's work (still on `feature/m4-webrtc`).
- New `internal/webrtcsignal` package: `Manager` owns one
  `peerconn`-built peer connection per remote peer (matching the N-party
  room model, ADR-002, even though M4 only exercises two) and moves
  offer/answer/ICE-candidate messages between them and the signaling
  relay. `Message` is the JSON shape carried as the relay's opaque
  payload. `Offer(ctx, peer)` creates a connection + data channel and
  sends the SDP offer; `HandleRelay(ctx, env)` dispatches an incoming
  envelope to offer/answer/candidate handling, creating the connection on
  first offer. Trickled ICE candidates are relayed automatically via
  `OnICECandidate`.
- Test (`webrtcsignal_test.go`) is a real end-to-end check, not a mock:
  spins up an actual `signalserver`, connects two real
  `signalclient.Client`s to it, has one `Offer` the other, and asserts
  the resulting data channels actually open and a message sent on one
  side arrives on the other. This is the same-machine, two-process test
  discussed earlier for M4, now automated -- and it substantially covers
  what part 5's self-test was meant to prove, so part 5 should mostly be
  App-level wiring + surfacing the test message to the frontend, not
  re-proving the handshake.
- `go build ./...`, `go vet ./...`, and `go test ./... -race` all pass.

## 2026-10-15 — Milestone 4 part 4 (Wails integration) done, plus a real bug fix

- `App` now creates a `webrtcsignal.Manager` alongside the signaling
  client on `CreateRoom`/`JoinRoom`, and closes it in `LeaveRoom` (before
  the signaling client, so nothing races to use an already-gone relay).
- `handleEvent`: `TypePeerJoined`'s `From` field is the joining peer's own
  id (the server stamps it in `Hub.JoinRoom`/`broadcastPeerListEvent`) --
  used directly, no diffing needed. If this client is host, it calls
  `Manager.Offer` for that peer. `TypeRelay` envelopes now route to
  `Manager.HandleRelay` (previously silently ignored). Both run in a
  goroutine off the signalclient read loop and log via
  `runtime.LogErrorf` on failure rather than blocking it.
- New `webrtc:peer-connected` Wails event (payload: peer id string),
  emitted once a given peer's data channel opens.
- **Bug found and fixed while wiring this up:** `go test ./... -race`
  started failing `webrtcsignal`'s end-to-end test intermittently (~1 in
  6 runs) once relay handling ran through goroutines the way `app.go`
  now drives it -- `add ICE candidate: remote description is not set`.
  Root cause: trickled ICE candidates and the offer/answer SDP travel as
  separate relay messages with no ordering guarantee, so a candidate can
  legitimately arrive before the SDP that was logically sent first.
  Fixed by queueing per-peer candidates in `Manager` until
  `SetRemoteDescription` succeeds, then flushing. Confirmed with 30
  consecutive `-race` runs, no failures. This was a real correctness gap
  in part 3's glue, not an app.go issue -- caught by the race detector
  because app.go's goroutines interleave the read loop differently than
  the single-goroutine test harness did.
- `go build ./...`, `go vet ./...`, `go test ./... -race` all pass.

## 2026-10-15 — Milestone 4 part 5 done; Milestone 4 complete

- `handleDataChannelOpen` (`app.go`) now sends a hardcoded
  `"hello from <peerId>"` text message once a peer's data channel opens,
  and forwards whatever message arrives on that channel as a new
  `webrtc:test-message` Wails event (`TestMessage{Peer, Message}`).
  Guards against the room having been left between the channel opening
  and the send (`a.client` nil check).
- This is the actual product-visible goal of M4: not just that a data
  channel opens, but that application data crosses it and reaches the
  other side. The handshake correctness itself was already proven by
  part 3's end-to-end test (two real signaling clients, real
  offer/answer, message round-trip) -- this step is the thin, already-
  tested-underneath layer that makes it visible to the app.
- `go build ./...`, `go vet ./...`, `go test ./... -race` all pass.
- **Milestone 4 is complete.** All 5 parts done: Pion verified (ADR-004),
  `internal/peerconn`, `internal/webrtcsignal`, Wails integration, and
  the test message. Everything is on `feature/m4-webrtc`, not yet merged
  to `main` -- merge is a separate decision for the project owner.
- **Known gap, deliberately out of scope:** the Svelte frontend doesn't
  subscribe to or display `webrtc:peer-connected`/`webrtc:test-message`
  at all. M4's brief was "establish a WebRTC peer connection and exchange
  a test message over a data channel" -- read as Go-side scope, same as
  M2 only needing one verified Go↔frontend call. Surfacing this in the UI
  is a product decision, not assumed here.

## Next

Project owner to decide: merge `feature/m4-webrtc` to `main`, and whether
the frontend should display WebRTC connection state / the test message
before moving on, or whether M5 (cross-network connectivity test,
ICE/STUN/TURN behavior, recorded empirically) starts first. See
`docs/plan.md` open questions before M8 (controller-handoff behavior needs
clarifying with the project owner first, but that's M8, not M4).
