# PeerPass — Architecture

Describes real, built components only. Nothing here is aspirational — see
`docs/plan.md` for what's planned but not yet built.

## Current state (as of Milestone 3)

Two processes: the Wails desktop app (per player), and a signaling server
mediating room creation/joining between them. No WebRTC yet — that's M4.

```
┌───────────────────────────┐              ┌───────────────────────────┐
│  Wails app (host player)   │              │  Wails app (peer player)   │
│  ┌───────────────────────┐ │              │  ┌───────────────────────┐ │
│  │ Svelte frontend        │ │              │  │ Svelte frontend        │ │
│  │ App.svelte              │ │              │  │ App.svelte              │ │
│  │ CreateRoom/JoinRoom/    │ │              │  │ CreateRoom/JoinRoom/    │ │
│  │ LeaveRoom bindings      │ │              │  │ LeaveRoom bindings      │ │
│  │ signal:* events in      │ │              │  │ signal:* events in      │ │
│  └───────────┬─────────────┘ │              │  └───────────┬─────────────┘ │
│              │ Wails IPC      │              │              │ Wails IPC      │
│  ┌───────────▼─────────────┐ │              │  ┌───────────▼─────────────┐ │
│  │ Go backend (app.go)      │ │              │  │ Go backend (app.go)      │ │
│  │ owns a signalclient.     │ │              │  │ owns a signalclient.     │ │
│  │ Client, never the        │ │              │  │ Client, never the        │ │
│  │ frontend                │ │              │  │ frontend                │ │
│  └───────────┬─────────────┘ │              │  └───────────┬─────────────┘ │
└──────────────┼───────────────┘              └──────────────┼───────────────┘
               │ WebSocket (ws://.../ws)                      │ WebSocket
               └──────────────────────┬────────────────────────┘
                                       ▼
                     ┌─────────────────────────────────────┐
                     │  cmd/signal (signaling server)        │
                     │  ┌─────────────────────────────────┐ │
                     │  │ internal/signalserver             │ │
                     │  │ WebSocket transport, read/write   │ │
                     │  │ pumps per connection               │ │
                     │  └────────────────┬────────────────┘ │
                     │  ┌────────────────▼────────────────┐ │
                     │  │ internal/signaling                 │ │
                     │  │ Hub: rooms, peers, relay logic     │ │
                     │  │ (pure, no network code, unit-      │ │
                     │  │ tested without a socket)           │ │
                     │  └─────────────────────────────────┘ │
                     └─────────────────────────────────────┘
```

- **`internal/signaling`** — the room/peer model. `Hub` owns every active
  `Room`, assigns peer IDs and room codes (`crypto/rand`), and decides who
  gets notified of what (join/leave/host-leave-closes-room/relay). No
  networking code at all — mutex-guarded, testable with plain Go tests.
  N-party by design (ADR-002): a room is a peer list, not a hardcoded pair.
- **`internal/signalserver`** — the WebSocket transport. One HTTP handler,
  `GET /ws` (create) or `GET /ws?room=CODE` (join). Drives the Hub: reads
  client messages and calls `Hub.Relay`, writes whatever the Hub queues onto
  each peer's `Send` channel. Read limit capped at 16 KiB to leave headroom
  for WebRTC SDP/ICE payloads from M4.
- **`internal/signalclient`** — what the Wails Go backend uses to connect.
  The frontend never opens a socket itself; only `app.go` does, through this
  package, and only exposes `CreateRoom`/`JoinRoom`/`LeaveRoom` to the UI.
- **`cmd/signal`** — thin binary: flag parsing (`-addr`, default
  `127.0.0.1:8080`) and `http.ListenAndServe`. All real logic lives in the
  two internal packages above.
- **`app.go`** — the Wails-bound methods (`CreateRoom`, `JoinRoom`,
  `LeaveRoom`, plus the original `Greet`). Translates signaling events into
  Wails frontend events (`signal:peer-joined`, `signal:peer-left`,
  `signal:room-closed`) via `runtime.EventsEmit`.
- **`frontend/src/App.svelte`** — lets a player type a signaling address,
  create or join a room, and see the live peer list update as others join
  or leave.
