# PeerPass — Architecture

Describes real, built components only. Nothing here is aspirational — see
`docs/plan.md` for what's planned but not yet built.

## Current state (as of Milestone 4)

Two processes: the Wails desktop app (per player), and a signaling server
mediating room creation/joining between them. The apps now also hold a
direct WebRTC peer connection with each other, negotiated by exchanging
SDP/ICE as opaque payloads over the same signaling relay (not a separate
channel) — the signaling server never sees WebRTC content, only forwards it.

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
- **`internal/peerconn`** — builds a Pion `webrtc.PeerConnection` with a
  public STUN server wired into `ICEServers` at construction time (not
  retrofitted later — see ADR-004's STUN note), plus the one data channel
  M4 uses, under a shared label constant.
- **`internal/webrtcsignal`** — the glue between signaling and WebRTC.
  `Manager` owns one `peerconn`-built connection per remote peer and moves
  offer/answer/ICE-candidate messages between them and
  `signalclient.Relay`, as JSON carried in the relay's opaque payload.
  Queues trickled ICE candidates that arrive before the remote description
  is set (no ordering guarantee between separate relay messages) and
  flushes them once it is. `Offer` initiates (the host, to each new peer);
  `HandleRelay` handles an incoming envelope either way.
- **`app.go`** — the Wails-bound methods (`CreateRoom`, `JoinRoom`,
  `LeaveRoom`, plus the original `Greet`). Translates signaling events into
  Wails frontend events (`signal:peers`, `signal:room-closed`,
  `signal:disconnected`) via `runtime.EventsEmit`. Owns a
  `webrtcsignal.Manager` alongside the signaling client: offers to each
  peer as it joins (if host), routes relay envelopes to the Manager, and
  once a peer's data channel opens, sends M4's hardcoded test message and
  forwards whatever comes back as `webrtc:test-message`.
- **`frontend/src/App.svelte`** — lets a player type a signaling address,
  create or join a room, and see the live peer list update as others join
  or leave. Does not yet show WebRTC connection state or the test message
  — that's frontend work still to do, not part of M4's Go-side scope.
