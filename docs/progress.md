# PeerPass — Progress Log

Newest entries at top. This is a factual log of what was actually done and
actually verified — not a plan, not a status aspiration.

---

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
- **Verified end-to-end**, not just compiled: ran `wails dev -tags
  webkit2_41`, confirmed the native window opened on the live display, and
  separately drove the browser-reachable dev server
  (`http://localhost:34115`, read from the command's own log output) via
  `claude-in-chrome` — typed "PeerPass", clicked Greet, confirmed
  "Hello PeerPass, It's show time!" rendered.
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

Milestone 3 — minimal signaling flow, N-party room model (room code, peer
list, host identity, join/leave) even though only two peers will connect
initially. See `docs/plan.md` open questions before M8 (controller-handoff
behavior needs clarifying with the project owner first, but that's M8, not M3).
