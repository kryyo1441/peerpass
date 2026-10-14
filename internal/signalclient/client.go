// Package signalclient is what the Wails Go backend uses to talk to a
// signaling server. The Svelte frontend never opens this connection itself
// -- it only sees the results via Wails-bound methods and events, so the
// frontend can't be tricked into talking to an arbitrary signaling server
// or sending unvalidated messages directly.
package signalclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync/atomic"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"peerpass/internal/signaling"
)

// Welcome is the server's first message, decoded for callers.
type Welcome struct {
	Room   signaling.RoomCode `json:"room"`
	PeerID signaling.PeerID   `json:"peerId"`
	Peers  []signaling.PeerID `json:"peers"`
	Host   bool               `json:"host"`
}

// EventHandler receives every message the server sends after the initial
// welcome: peer-joined, peer-left, room-closed, relay, error.
type EventHandler func(signaling.Envelope)

// Client holds one connection to a signaling server for the lifetime of a
// single room membership.
type Client struct {
	conn   *websocket.Conn
	cancel context.CancelFunc
	closed atomic.Bool // set by Close(), so readLoop can tell "we closed this on purpose" from "it just died"
	PeerID signaling.PeerID
	Room   signaling.RoomCode
	IsHost bool
}

// Connect dials the signaling server at addr (host:port, no scheme) and
// either creates a new room (room == "") or joins an existing one. The
// returned Client's read loop runs until ctx is cancelled or the connection
// drops; every subsequent server message is passed to onEvent.
//
// onDisconnect is called exactly once if the connection drops for any
// reason other than the caller's own Close() -- e.g. the signaling server
// crashed or the network dropped. It is never called after an explicit
// Close(), so callers can safely react to it by tearing down UI state
// without worrying about a spurious call racing their own cleanup.
//
// ctx should be tied to something that gets cancelled on explicit
// leave/disconnect and on application shutdown -- this client does not
// outlive its caller's lifecycle management.
func Connect(ctx context.Context, addr string, room signaling.RoomCode, onEvent EventHandler, onDisconnect func()) (*Client, Welcome, error) {
	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws"}
	if room != "" {
		q := u.Query()
		q.Set("room", string(room))
		u.RawQuery = q.Encode()
	}

	conn, _, err := websocket.Dial(ctx, u.String(), nil)
	if err != nil {
		return nil, Welcome{}, fmt.Errorf("signalclient: dial: %w", err)
	}

	var welcomeEnv signaling.Envelope
	if err := wsjson.Read(ctx, conn, &welcomeEnv); err != nil {
		conn.CloseNow()
		return nil, Welcome{}, fmt.Errorf("signalclient: read welcome: %w", err)
	}
	if welcomeEnv.Type == signaling.TypeError {
		conn.CloseNow()
		return nil, Welcome{}, fmt.Errorf("signalclient: server rejected connection: %s", welcomeEnv.Payload)
	}

	var welcome Welcome
	if err := json.Unmarshal(welcomeEnv.Payload, &welcome); err != nil {
		conn.CloseNow()
		return nil, Welcome{}, fmt.Errorf("signalclient: decode welcome: %w", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	c := &Client{conn: conn, cancel: cancel, PeerID: welcome.PeerID, Room: welcome.Room, IsHost: welcome.Host}

	go c.readLoop(runCtx, onEvent, onDisconnect)

	return c, welcome, nil
}

func (c *Client) readLoop(ctx context.Context, onEvent EventHandler, onDisconnect func()) {
	for {
		var env signaling.Envelope
		if err := wsjson.Read(ctx, c.conn, &env); err != nil {
			if !c.closed.Load() && onDisconnect != nil {
				onDisconnect()
			}
			return
		}
		if onEvent != nil {
			onEvent(env)
		}
	}
}

// Relay sends an opaque payload to another peer in the same room (used from
// Milestone 4 on for SDP offers/answers and ICE candidates).
func (c *Client) Relay(ctx context.Context, to signaling.PeerID, payload []byte) error {
	env := signaling.Envelope{Type: signaling.TypeRelay, To: to, Payload: payload}
	return wsjson.Write(ctx, c.conn, env)
}

// Close ends the connection and stops the read loop. Marks closed first so
// readLoop's resulting read error is recognized as an intentional close,
// not reported via onDisconnect.
func (c *Client) Close() {
	c.closed.Store(true)
	c.cancel()
	c.conn.Close(websocket.StatusNormalClosure, "")
}
