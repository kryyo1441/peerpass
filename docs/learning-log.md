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
