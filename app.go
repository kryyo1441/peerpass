package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pion/webrtc/v4"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"peerpass/internal/signalclient"
	"peerpass/internal/signaling"
	"peerpass/internal/webrtcsignal"
)

// Wails event names the frontend can subscribe to via runtime.EventsOn.
// The frontend never opens its own signaling connection -- it only reacts
// to these events and calls the bound methods below.
const (
	EventPeers         = "signal:peers"          // payload: []string, the full current peer list
	EventRoomClosed    = "signal:room-closed"    // host left; frontend should call LeaveRoom()
	EventDisconnected  = "signal:disconnected"   // connection to the signaling server itself died; frontend should call LeaveRoom()
	EventPeerConnected = "webrtc:peer-connected" // payload: peer id string; that peer's data channel is open
	EventTestMessage   = "webrtc:test-message"   // payload: TestMessage; M4's data-channel test message was received
)

// TestMessage is the M4 "data channel test message" surfaced to the
// frontend -- proof the data channel actually carries application data,
// not just that it opened.
type TestMessage struct {
	Peer    string `json:"peer"`
	Message string `json:"message"`
}

// RoomInfo is the plain-JSON shape returned to the frontend from
// CreateRoom/JoinRoom -- deliberately not signaling.Welcome, so the
// frontend's generated TS types stay simple strings rather than our
// internal PeerID/RoomCode types.
type RoomInfo struct {
	Room   string   `json:"room"`
	PeerID string   `json:"peerId"`
	Peers  []string `json:"peers"`
	Host   bool     `json:"host"`
}

// App struct
type App struct {
	ctx context.Context

	// mu guards client/cancelConn: bound methods (CreateRoom/JoinRoom/
	// LeaveRoom, called on whatever goroutine Wails dispatches them on) and
	// shutdown() can run concurrently with each other, and with
	// handleDisconnect (called from the signalclient read-loop goroutine).
	// Go has no GIL -- unlike Python, concurrent unsynchronized access to
	// these fields would be a real data race, not just a slow interleaving.
	mu         sync.Mutex
	cancelConn context.CancelFunc
	client     *signalclient.Client
	manager    *webrtcsignal.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// shutdown is called when the app is closing. Make sure we don't leave a
// signaling connection open past the app's own lifetime.
func (a *App) shutdown(ctx context.Context) {
	a.LeaveRoom()
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// CreateRoom connects to the signaling server at addr (e.g. "127.0.0.1:8080")
// and creates a new room, becoming its host.
func (a *App) CreateRoom(addr string) (RoomInfo, error) {
	return a.connect(addr, "")
}

// JoinRoom connects to the signaling server at addr and joins an existing
// room by its code.
func (a *App) JoinRoom(addr string, room string) (RoomInfo, error) {
	return a.connect(addr, room)
}

func (a *App) connect(addr string, room string) (RoomInfo, error) {
	a.LeaveRoom() // drop any previous connection first

	connCtx, cancel := context.WithCancel(a.ctx)
	client, welcome, err := signalclient.Connect(connCtx, addr, signaling.RoomCode(room), a.handleEvent, a.handleDisconnect)
	if err != nil {
		cancel()
		return RoomInfo{}, err
	}

	manager := webrtcsignal.NewManager(client, a.handleDataChannelOpen)

	a.mu.Lock()
	a.cancelConn = cancel
	a.client = client
	a.manager = manager
	a.mu.Unlock()

	peers := make([]string, len(welcome.Peers))
	for i, p := range welcome.Peers {
		peers[i] = string(p)
	}
	return RoomInfo{
		Room:   string(welcome.Room),
		PeerID: string(welcome.PeerID),
		Peers:  peers,
		Host:   welcome.Host,
	}, nil
}

// LeaveRoom closes the current signaling connection, if any. Safe to call
// when there is no active connection.
func (a *App) LeaveRoom() {
	a.mu.Lock()
	client, cancel, manager := a.client, a.cancelConn, a.manager
	a.client, a.cancelConn, a.manager = nil, nil, nil
	a.mu.Unlock()

	// Close the peer connections before the signaling client -- Manager
	// doesn't need the relay once it's told to shut down, but closing in
	// this order means nothing races to use a relay that's already gone.
	if manager != nil {
		manager.Close()
	}
	if client != nil {
		client.Close()
	}
	if cancel != nil {
		cancel()
	}
}

// handleEvent translates signaling messages into Wails frontend events.
// Runs on the signalclient read-loop goroutine, not the UI goroutine --
// runtime.EventsEmit is safe to call from any goroutine.
func (a *App) handleEvent(env signaling.Envelope) {
	switch env.Type {
	case signaling.TypePeerJoined, signaling.TypePeerLeft:
		var payload signaling.PeerListPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return
		}
		peers := make([]string, len(payload.Peers))
		for i, p := range payload.Peers {
			peers[i] = string(p)
		}
		runtime.EventsEmit(a.ctx, EventPeers, peers)

		// For TypePeerJoined, From is the peer that just joined (the server
		// stamps it -- see Hub.JoinRoom/broadcastPeerListEvent). The host
		// initiates the WebRTC connection to every new peer; a non-host
		// peer just waits for an incoming offer via TypeRelay below.
		if env.Type == signaling.TypePeerJoined && env.From != "" {
			a.mu.Lock()
			client, manager := a.client, a.manager
			a.mu.Unlock()
			if client != nil && client.IsHost && manager != nil {
				go func() {
					if err := manager.Offer(a.ctx, env.From); err != nil {
						runtime.LogErrorf(a.ctx, "webrtc: offer to %s: %v", env.From, err)
					}
				}()
			}
		}
	case signaling.TypeRoomClosed:
		runtime.EventsEmit(a.ctx, EventRoomClosed)
	case signaling.TypeRelay:
		a.mu.Lock()
		manager := a.manager
		a.mu.Unlock()
		if manager != nil {
			go func() {
				if err := manager.HandleRelay(a.ctx, env); err != nil {
					runtime.LogErrorf(a.ctx, "webrtc: handle relay from %s: %v", env.From, err)
				}
			}()
		}
	}
}

// handleDataChannelOpen is called by Manager once per remote peer when its
// data channel opens. Runs on whatever goroutine Pion invokes the Manager's
// callback from, not the UI goroutine -- runtime.EventsEmit is safe from
// any goroutine. Sends M4's hardcoded test message and forwards whatever
// the other side sends back -- this is the actual product-visible proof
// the data channel works, not just that it opened.
func (a *App) handleDataChannelOpen(peer signaling.PeerID, dc *webrtc.DataChannel) {
	runtime.EventsEmit(a.ctx, EventPeerConnected, string(peer))

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		runtime.EventsEmit(a.ctx, EventTestMessage, TestMessage{Peer: string(peer), Message: string(msg.Data)})
	})

	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil { // room was left between connecting and this channel opening
		return
	}
	if err := dc.SendText(fmt.Sprintf("hello from %s", client.PeerID)); err != nil {
		runtime.LogErrorf(a.ctx, "webrtc: send test message to %s: %v", peer, err)
	}
}

// handleDisconnect is called by signalclient when the connection drops for
// a reason other than our own LeaveRoom()/Close(). It deliberately does not
// call LeaveRoom() itself (that would run Close() from the read-loop
// goroutine that just reported its own death) -- it only notifies the
// frontend, which calls LeaveRoom() in response.
func (a *App) handleDisconnect() {
	runtime.EventsEmit(a.ctx, EventDisconnected)
}
