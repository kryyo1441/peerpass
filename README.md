# PeerPass

A peer-to-peer desktop app for shared, turn-based play of a single-copy game.
One player hosts, others watch, and the host can hand the controller to a
requesting player, whose input is then injected on the host's machine.

## Current capabilities

**Milestone 2 of the roadmap.** There is a working Wails v2 desktop shell
(Go backend + Svelte/TypeScript frontend) with one verified Go-bound method
call (`Greet`). There is no networking, no signaling, no WebRTC, no input
sharing, and no controller handoff yet — see `docs/plan.md` for what's next.

## Security status — read this before running anything networked

**No milestone involving networking has landed yet**, so there is nothing to
expose. Once Milestone 3 (signaling) lands, this section will state plainly:
PeerPass has no authentication and no session authorization until Milestone 9.
Any peer that knows a room code will be able to join. Until then, treat any
networked build as usable only between machines you control on a trusted
network — never expose it to the open internet. See `docs/decisions.md`
ADR-005.

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

## Testing

```sh
go test -tags webkit2_41 ./...
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
