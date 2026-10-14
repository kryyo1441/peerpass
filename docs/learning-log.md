# PeerPass — Learning Log

Concepts learned, mistakes made, debugging lessons, and unresolved questions,
in the order they came up. Not a tutorial — a record for the project owner
(learning Go/networking/concurrency/WebRTC) to look back on.

## Milestone 1

- No Go/Wails code written yet — nothing to log here.

## Milestone 2 — Wails shell

### What a webview actually is

Wails doesn't bundle a browser. It opens a native OS window and puts the
platform's own webview control inside it — on Linux that's webkit2gtk (the
same WebKit engine Safari uses, wrapped for GTK), on Windows it's WebView2
(Edge/Chromium), on macOS it's WKWebView (also WebKit). Your Svelte/TS
frontend runs inside that webview exactly like a web page — same HTML/CSS/JS
— but there is no browser chrome (no address bar, no tabs), and critically,
**Go code is running in the same process as the native window**, not on some
remote server. That's what makes bindings possible.

### How Go↔JS bindings work

You write a normal Go method on a struct (`func (a *App) Greet(name string)
string`). Wails generates a matching JS/TS function in `frontend/wailsjs/go/`.
When the frontend calls that generated function, Wails serializes the
arguments, sends them over an internal IPC channel to the Go side, runs your
real Go method, and sends the return value back as a resolved Promise. From
the frontend's perspective it just looks like calling an async function —
there's no HTTP request, no JSON API you wrote yourself.

Python comparison: this is conceptually similar to something like PyWebView's
`js_api`, or even more loosely like how a Jupyter kernel exposes Python
objects to the frontend — you're not writing a server, you're exposing native
functions across a process-internal bridge.

### `wails dev` has two usable targets

Running `wails dev` starts *two* things: the actual native app window, **and**
a browser-reachable dev server (defaults to `localhost:34115`, but the plan
is to always read the real port from the command's own output, since it can
change). The browser version is useful for `claude-in-chrome`-style automated
verification and for using devtools to call Go methods directly from the
console — but it's still going through the same Wails IPC bridge, not a
separate mock.

### Debugging lesson: a green build does not mean the UI works

`wails build`, `go vet`, and `svelte-check` all passed cleanly while the app
was rendering a **blank screen** both in the native window and in the browser
dev view. The actual bug (Svelte 5 removed the `new Component()` constructor
that the Wails `svelte-ts` template's `main.ts` still used) only showed up as
a runtime exception in the browser console (`component_api_invalid_new`) —
none of the static tools caught it, because it's a *runtime* API removal, not
a type error. Lesson: always actually run the app and look at it (or drive it
with browser automation) before calling a milestone done — see
`docs/decisions.md` ADR-001's "Takeaway recorded for future milestones."

### Unresolved / open

- Haven't verified Windows or macOS builds at all yet — only Arch Linux.
- `wails doctor`'s dependency check for webkit2gtk is stale (doesn't recognize
  4.1) — worth reporting upstream at some point, not blocking for us since we
  know the workaround (`-tags webkit2_41`).

## Milestone 3 — signaling

### Why peer-to-peer still needs a server

"Peer-to-peer" describes how *game data* will flow later (directly between
two players' machines, M4 onward). But before a direct connection can exist,
each side needs to learn things about the other (and later, connection
details like ICE candidates) that neither one has yet — like two strangers
who can't call each other until someone hands out phone numbers first. The
signaling server's only job is to be that always-on meeting point: create
rooms, hand out room codes, tell peers who else is in the room, and (from M4
on) relay WebRTC handshake messages it doesn't itself need to understand.

### Mutex vs. a single-owner goroutine (and the Python GIL)

`internal/signaling.Hub` holds a `map[RoomCode]*Room` that many goroutines
(one per connected peer, each running its own read loop in
`internal/signalserver`) can touch at once — one peer joining while another
is leaving while a third is being relayed a message. Go does **not** have a
GIL: without protection, concurrent map reads/writes from different
goroutines can corrupt memory or crash the program outright, which is a much
sharper failure than Python's GIL-induced "just slow interleaving."

Two ways to make shared mutable state safe in Go:

1. **A mutex** (what `Hub` actually uses): every method that touches
   `h.rooms` takes `h.mu.Lock()`/`defer h.mu.Unlock()` first. Simple,
   familiar if you've used locks in Python's `threading` module — the
   difference is Go will actually race-condition-corrupt data without one,
   where Python's GIL accidentally protects some (not all) of that by
   serializing bytecode execution.
2. **A single-owner goroutine**: instead of locking, one goroutine owns the
   map exclusively and everyone else talks to it only through a channel of
   requests. No lock needed because only one goroutine ever touches the
   data — this is the "don't communicate by sharing memory, share memory by
   communicating" idiom Go advertises. Would have worked here too, but for
   a map this small and operations this short, a mutex is simpler and the
   standard lazy choice — ponytail: a dedicated owner-goroutine + channel
   protocol is the upgrade path if `Hub` ever needs to do something slow
   while holding state (it currently doesn't; every method returns fast).

`go test -race ./...` is the tool that actually catches a missed lock —
passing tests alone don't prove the code is race-free, since a race can
exist without ever triggering with a given test's exact timing. This is
part of why `-race` is now a required check for this codebase going forward,
not just a nice-to-have.

### Non-blocking sends to avoid one slow peer stalling everyone

`Hub.send` uses `select { case peer.Send <- env: default: }` instead of a
plain blocking channel send. If it blocked, one peer whose write-pump
goroutine died or is just slow to drain its channel would stall the *entire
hub* — every other room's join/leave/relay calls would hang waiting for that
one lock to free up, since they all go through the same mutex-guarded `Hub`.

**Update:** the first version of this just silently dropped the message on
a full buffer, which was fine for presence notifications but would have
meant a dropped WebRTC SDP/ICE relay (from M4 on) vanishing with no signal
to anyone. A later review pass caught this and added eviction instead: a
full buffer now means the peer gets removed from its room (via `Peer.Done()`,
which the transport layer watches to close the actual connection) rather
than quietly losing messages forever. See `docs/decisions.md` ADR for the
`send`/`removePeer` change.

### Why a separate `internal/signaling` package with no network imports

`Hub`/`Room`/`Peer` import nothing from `net/http` or `github.com/coder/
websocket`. That's deliberate: it means the room/peer rules (who gets
notified, what happens when the host leaves, rejecting a relay to someone
in a different room) can be unit-tested with plain `go test`, no sockets, no
`httptest` server, no network flakiness. The actual networking
(`internal/signalserver`) is a thin layer on top that only deals with
reading/writing bytes and calling into the Hub — the kind of split that
makes the hard-to-get-right concurrency logic testable in isolation.
