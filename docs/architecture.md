# PeerPass — Architecture

Describes real, built components only. Nothing here is aspirational — see
`docs/plan.md` for what's planned but not yet built.

## Current state (as of Milestone 2)

A single Wails v2 desktop application, no networking yet.

```
┌─────────────────────────────────────────────┐
│              Native desktop window           │
│  ┌─────────────────────────────────────────┐ │
│  │   Webview (webkit2gtk on Linux)          │ │
│  │   Svelte + TypeScript frontend           │ │
│  │   frontend/src/App.svelte, main.ts       │ │
│  └───────────────────┬───────────────────────┘ │
│                       │ Wails-generated JS      │
│                       │ bindings (wailsjs/)     │
│  ┌────────────────────▼──────────────────────┐ │
│  │   Go backend (main.go, app.go)             │ │
│  │   App struct — exposes bound methods       │ │
│  │   e.g. Greet(name string) string           │ │
│  └─────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
```

- **`main.go` / `app.go`** — the Go side. `App` is the struct whose exported
  methods Wails auto-generates JS bindings for (`frontend/wailsjs/go/`). Only
  `Greet` exists right now, carried over from the template as the first
  verified Go↔frontend call.
- **`frontend/src/`** — the Svelte/TypeScript UI. `main.ts` mounts `App.svelte`
  into `#app` using Svelte 5's `mount()` API. `App.svelte` calls the generated
  `Greet` binding and renders the result.
- **No session/room/signaling/WebRTC/input layers exist yet** — those land in
  Milestones 3+ per `docs/plan.md`, and will be documented here as they're
  actually built, respecting the N-party-room (ADR-002) and
  deferred-media-track (ADR-003) design commitments.
