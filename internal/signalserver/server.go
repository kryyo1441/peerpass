// Package signalserver is the networking layer that drives a
// signaling.Hub over WebSocket connections. It owns all the I/O; the
// room/peer rules themselves live in internal/signaling so they can be
// unit-tested without a socket.
package signalserver

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"peerpass/internal/signaling"
)

var errUnknownMessageType = errors.New("signalserver: unknown message type")

// maxMessageSize bounds one incoming frame. Must comfortably fit a WebRTC
// SDP offer with ICE candidates (several KB) once M4 starts relaying those
// through here -- 16 KiB leaves headroom without being unbounded.
const maxMessageSize = 16 * 1024

// Server is an http.Handler that upgrades requests to WebSocket connections
// and drives them against a signaling.Hub.
type Server struct {
	hub *signaling.Hub
}

func New(hub *signaling.Hub) *Server {
	return &Server{hub: hub}
}

// ServeHTTP implements the one endpoint this server exposes:
//
//	GET /ws              -> create a new room, caller becomes host
//	GET /ws?room=ABC123   -> join an existing room
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return // Accept already wrote the appropriate HTTP error response.
	}
	conn.SetReadLimit(maxMessageSize)

	ctx := r.Context()
	roomParam := r.URL.Query().Get("room")

	var (
		code   signaling.RoomCode
		peerID signaling.PeerID
		peer   *signaling.Peer
		peers  []signaling.PeerID
		isHost bool
	)

	if roomParam == "" {
		var p *signaling.Peer
		code, peerID, p = s.hub.CreateRoom()
		peer = p
		peers = []signaling.PeerID{peerID}
		isHost = true
	} else {
		code = signaling.RoomCode(roomParam)
		id, p, roomPeers, err := s.hub.JoinRoom(code)
		if err != nil {
			writeErrorAndClose(ctx, conn, err)
			return
		}
		peerID, peer, peers = id, p, roomPeers
	}

	welcome := signaling.Envelope{Type: signaling.TypeWelcome, To: peerID}
	welcomePayload, _ := json.Marshal(welcomePayload{Room: code, PeerID: peerID, Peers: peers, Host: isHost})
	welcome.Payload = welcomePayload
	if err := wsjson.Write(ctx, conn, welcome); err != nil {
		conn.CloseNow()
		return
	}

	defer func() {
		// ErrRoomNotFound/ErrPeerNotInRoom here just mean someone else's
		// cleanup (e.g. the host leaving) already removed this peer or the
		// whole room -- not a real error, so only log anything else.
		if err := s.hub.Leave(code, peerID); err != nil &&
			!errors.Is(err, signaling.ErrRoomNotFound) &&
			!errors.Is(err, signaling.ErrPeerNotInRoom) {
			log.Printf("signalserver: leave cleanup for %s in %s: %v", peerID, code, err)
		}
	}()

	s.writePump(ctx, conn, peer)
	// writePump returns once the read loop (launched below) detects the
	// connection died and cancels its own context -- see run().
	_ = s.run(ctx, conn, code, peerID, peer)
}

type welcomePayload struct {
	Room   signaling.RoomCode `json:"room"`
	PeerID signaling.PeerID   `json:"peerId"`
	Peers  []signaling.PeerID `json:"peers"`
	Host   bool               `json:"host"`
}

// run is the read loop: it blocks relaying client -> hub messages until the
// connection closes or ctx is cancelled (e.g. by Wails app shutdown via the
// request context in a real deployment, or test teardown).
func (s *Server) run(ctx context.Context, conn *websocket.Conn, code signaling.RoomCode, from signaling.PeerID, peer *signaling.Peer) error {
	readCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-readCtx.Done()
		conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		var env signaling.Envelope
		if err := wsjson.Read(readCtx, conn, &env); err != nil {
			return err
		}

		switch env.Type {
		case signaling.TypeRelay:
			if err := s.hub.Relay(code, from, env.To, env.Payload); err != nil {
				errEnv := signaling.Envelope{Type: signaling.TypeError, Payload: errorPayload(err)}
				_ = wsjson.Write(readCtx, conn, errEnv)
			}
		default:
			errEnv := signaling.Envelope{Type: signaling.TypeError, Payload: errorPayload(errUnknownMessageType)}
			_ = wsjson.Write(readCtx, conn, errEnv)
		}
	}
}

// writePump drains peer.Send and writes each envelope to the socket, in its
// own goroutine so a slow reader never blocks the Hub (see signaling.send).
func (s *Server) writePump(ctx context.Context, conn *websocket.Conn, peer *signaling.Peer) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case env, ok := <-peer.Send:
				if !ok {
					return
				}
				writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				err := wsjson.Write(writeCtx, conn, env)
				cancel()
				if err != nil {
					return
				}
			}
		}
	}()
}

func writeErrorAndClose(ctx context.Context, conn *websocket.Conn, err error) {
	writeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_ = wsjson.Write(writeCtx, conn, signaling.Envelope{Type: signaling.TypeError, Payload: errorPayload(err)})
	conn.Close(websocket.StatusPolicyViolation, err.Error())
}

func errorPayload(err error) json.RawMessage {
	b, _ := json.Marshal(struct {
		Message string `json:"message"`
	}{Message: err.Error()})
	return b
}
