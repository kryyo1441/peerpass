# PeerPass — Progress Log

Newest entries at top. This is a factual log of what was actually done and
actually verified — not a plan, not a status aspiration.

---

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

## Next

Milestone 4 — establish a WebRTC peer connection and exchange a test message
over a data channel, using the existing signaling relay (`Hub.Relay` /
`signalclient.Relay`) to carry SDP offers/answers and ICE candidates. See
`docs/plan.md` open questions before M8 (controller-handoff behavior needs
clarifying with the project owner first, but that's M8, not M4).
