package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"peerpass/internal/signalclient"
	"peerpass/internal/signaling"
)

// Wails event names the frontend can subscribe to via runtime.EventsOn.
// The frontend never opens its own signaling connection -- it only reacts
// to these events and calls the bound methods below.
const (
	EventPeerJoined = "signal:peer-joined"
	EventPeerLeft   = "signal:peer-left"
	EventRoomClosed = "signal:room-closed"
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
	ctx        context.Context
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
	client, welcome, err := signalclient.Connect(connCtx, addr, signaling.RoomCode(room), a.handleEvent)
	if err != nil {
		cancel()
		return RoomInfo{}, err
	}

	a.cancelConn = cancel
	a.client = client

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
	if a.client != nil {
		a.client.Close()
		a.client = nil
	}
	if a.cancelConn != nil {
		a.cancelConn()
		a.cancelConn = nil
	}
}

// handleEvent translates signaling messages into Wails frontend events.
// Runs on the signalclient read-loop goroutine, not the UI goroutine --
// runtime.EventsEmit is safe to call from any goroutine.
func (a *App) handleEvent(env signaling.Envelope) {
	switch env.Type {
	case signaling.TypePeerJoined:
		runtime.EventsEmit(a.ctx, EventPeerJoined, string(env.From))
	case signaling.TypePeerLeft:
		runtime.EventsEmit(a.ctx, EventPeerLeft, string(env.From))
	case signaling.TypeRoomClosed:
		runtime.EventsEmit(a.ctx, EventRoomClosed)
	}
}
