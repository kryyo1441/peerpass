# PeerPass

A peer-to-peer desktop app for shared, turn-based play of a single-copy game.
One player hosts, others watch, and the host can hand the controller to a
requesting player, whose input is then injected on the host's machine.

## Current capabilities

**Milestone 3 of the roadmap.** There is a working Wails v2 desktop shell
(Go backend + Svelte/TypeScript frontend) that can create or join a
signaling room and see a live peer list. A standalone signaling server
(`cmd/signal`) mediates room creation/joining and relays opaque messages
between peers (used from Milestone 4 on for WebRTC SDP/ICE exchange). There
is no WebRTC peer connection, no input sharing, and no controller handoff
yet — see `docs/plan.md` for what's next.

## Security status — read this before running anything networked

**PeerPass has no authentication and no session authorization until
Milestone 9.** Any peer that knows a room code can join a room and exchange
messages with the other peers in it. There is no way to verify who you're
actually connecting to.

- The signaling server (`cmd/signal`) binds to `127.0.0.1` by default — not
  reachable from the network unless you explicitly pass `-addr` with a
  non-loopback address.
- **Do not pass a non-loopback `-addr` or expose this to the open internet.**
  Treat it as usable only between machines you control on a trusted local
  network, for now.
- See `docs/decisions.md` ADR-005 for the full interim trust model and when
  it's expected to end (Milestone 9).

## Setup

Requires:
- Go (currently developed against go1.27.x)
- Node.js + npm (currently developed against Node 26)
- The Wails v2 CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Linux (Arch, and likely other distros that only ship webkit2gtk-4.1)

`wails doctor` will report `libwebkit: Not Found` even with webkit2gtk-4.1
installed — this is a stale check in the doctor command, not a real blocker.
**Every `wails` build/dev command in this project must pass `-tags
webkit2_41`** or it will fail to find the right webkit headers. See
`docs/decisions.md` ADR-001 for the full finding.

```sh
wails dev -tags webkit2_41      # live development
wails build -tags webkit2_41    # production build, output in build/bin/
```

### Other platforms

Not yet verified on Windows or macOS. Will be documented here once actually
tested — see the crossplay note in `docs/plan.md`.

### Running the signaling server

Required before creating or joining a room from the app. Defaults to
`127.0.0.1:8080` (see the security status above before changing `-addr`):

```sh
go run ./cmd/signal
```

The app's "Signaling server" field should then point at that address
(`127.0.0.1:8080` by default).

For a real deployment (M5, cross-network testing), the server binds
`0.0.0.0:$PORT` instead when `$PORT` is set in the environment -- this is
how free-tier hosts like Render assign a port, and it only takes effect if
you deliberately set it. Point the app's "Signaling server" field at the
deployed server's `wss://...` URL in that case (plain `host:port` still
means `ws://` for local dev).

## Testing

```sh
go test -tags webkit2_41 ./...
go test -race ./...            # mandatory for the concurrent signaling code
gofmt -l .
go vet -tags webkit2_41 ./...
cd frontend && npm run check   # TypeScript/Svelte type-check
```

## Documentation

- [`docs/handoff.md`](docs/handoff.md) — start here to resume work
- [`docs/plan.md`](docs/plan.md) — milestone roadmap and open questions
- [`docs/decisions.md`](docs/decisions.md) — engineering decision log
- [`docs/progress.md`](docs/progress.md) — dated log of what's actually been done
- [`docs/architecture.md`](docs/architecture.md) — current components
- [`docs/learning-log.md`](docs/learning-log.md) — concepts and lessons learned
