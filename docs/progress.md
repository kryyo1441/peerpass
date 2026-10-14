# PeerPass — Progress Log

Newest entries at top. This is a factual log of what was actually done and
actually verified — not a plan, not a status aspiration.

---

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
- Commit: `63581c8 build: add .gitignore for Go/Wails/frontend artifacts`
  (docs commit follows separately).

## Next

Milestone 2 — install Wails v2 CLI, scaffold the `svelte-ts` template, add one
real Go-bound method, verify the call end-to-end via `wails dev` + browser
automation, run format/vet/test/build, commit on `feat/wails-shell`.
