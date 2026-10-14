package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"peerpass/internal/signalclient"
	"peerpass/internal/signaling"
)

// Wails event names the frontend can subscribe to via runtime.EventsOn.
// The frontend never opens its own signaling connection -- it only reacts
// to these events and calls the bound methods below.
const (
	EventPeers        = "signal:peers"        // payload: []string, the full current peer list
	EventRoomClosed   = "signal:room-closed"  // host left; frontend should call LeaveRoom()
	EventDisconnected = "signal:disconnected" // connection to the signaling server itself died; frontend should call LeaveRoom()
)

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

	a.mu.Lock()
	a.cancelConn = cancel
	a.client = client
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
	client, cancel := a.client, a.cancelConn
	a.client, a.cancelConn = nil, nil
	a.mu.Unlock()

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
	case signaling.TypeRoomClosed:
		runtime.EventsEmit(a.ctx, EventRoomClosed)
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
