package signalserver

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"peerpass/internal/signaling"
)

func newTestServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	hub := signaling.NewHub()
	ts := httptest.NewServer(New(hub))
	return ts, ts.Close
}

func dial(t *testing.T, ctx context.Context, ts *httptest.Server, room string) (*websocket.Conn, welcomeMsg) {
	t.Helper()
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	if room != "" {
		url += "?room=" + room
	}
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	var env signaling.Envelope
	if err := wsjson.Read(ctx, conn, &env); err != nil {
		t.Fatalf("read welcome: %v", err)
	}
	if env.Type != signaling.TypeWelcome {
		t.Fatalf("first message type = %q, want %q", env.Type, signaling.TypeWelcome)
	}
	var w welcomeMsg
	if err := json.Unmarshal(env.Payload, &w); err != nil {
		t.Fatalf("unmarshal welcome payload: %v", err)
	}
	return conn, w
}

type welcomeMsg struct {
	Room   string   `json:"room"`
	PeerID string   `json:"peerId"`
	Peers  []string `json:"peers"`
	Host   bool     `json:"host"`
}

// TestThreeParty proves the room isn't hardcoded to two peers: a host plus
// two joining peers, all three visible to each other.
func TestThreeParty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ts, closeTS := newTestServer(t)
	defer closeTS()

	hostConn, host := dial(t, ctx, ts, "")
	defer hostConn.CloseNow()
	if !host.Host {
		t.Fatal("creator should be marked host")
	}

	peerAConn, peerA := dial(t, ctx, ts, host.Room)
	defer peerAConn.CloseNow()
	if len(peerA.Peers) != 2 {
		t.Fatalf("peerA welcome peer list = %v, want 2 entries", peerA.Peers)
	}

	// Host should see peerA join.
	var hostNotice signaling.Envelope
	if err := wsjson.Read(ctx, hostConn, &hostNotice); err != nil {
		t.Fatalf("host read peer-joined: %v", err)
	}
	if hostNotice.Type != signaling.TypePeerJoined || string(hostNotice.From) != peerA.PeerID {
		t.Fatalf("host notice = %+v, want peer-joined from %s", hostNotice, peerA.PeerID)
	}

	peerBConn, peerB := dial(t, ctx, ts, host.Room)
	defer peerBConn.CloseNow()
	if len(peerB.Peers) != 3 {
		t.Fatalf("peerB welcome peer list = %v, want 3 entries", peerB.Peers)
	}

	// Both host and peerA should see peerB join.
	for name, conn := range map[string]*websocket.Conn{"host": hostConn, "peerA": peerAConn} {
		var notice signaling.Envelope
		if err := wsjson.Read(ctx, conn, &notice); err != nil {
			t.Fatalf("%s read peer-joined: %v", name, err)
		}
		if notice.Type != signaling.TypePeerJoined || string(notice.From) != peerB.PeerID {
			t.Fatalf("%s notice = %+v, want peer-joined from %s", name, notice, peerB.PeerID)
		}
	}
}

// TestRelay_ReachesOnlyRecipient proves a targeted relay doesn't leak to
// every peer in the room.
func TestRelay_ReachesOnlyRecipient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ts, closeTS := newTestServer(t)
	defer closeTS()

	hostConn, host := dial(t, ctx, ts, "")
	defer hostConn.CloseNow()

	peerAConn, peerA := dial(t, ctx, ts, host.Room)
	defer peerAConn.CloseNow()
	var hostNotice signaling.Envelope
	_ = wsjson.Read(ctx, hostConn, &hostNotice) // drain peerA's join notice

	peerBConn, peerB := dial(t, ctx, ts, host.Room)
	defer peerBConn.CloseNow()
	_ = wsjson.Read(ctx, hostConn, &hostNotice)  // drain peerB's join notice on host
	_ = wsjson.Read(ctx, peerAConn, &hostNotice) // drain peerB's join notice on peerA

	// Host relays a message to peerA only.
	msg := signaling.Envelope{Type: signaling.TypeRelay, To: signaling.PeerID(peerA.PeerID), Payload: []byte(`{"sdp":"offer"}`)}
	if err := wsjson.Write(ctx, hostConn, msg); err != nil {
		t.Fatalf("write relay: %v", err)
	}

	var received signaling.Envelope
	if err := wsjson.Read(ctx, peerAConn, &received); err != nil {
		t.Fatalf("peerA read relay: %v", err)
	}
	if received.Type != signaling.TypeRelay || string(received.From) != host.PeerID {
		t.Fatalf("peerA received = %+v, want relay from host %s", received, host.PeerID)
	}

	// peerB must NOT receive it. Give it a short deadline so the test
	// doesn't hang if something is wrong -- it should time out, proving
	// nothing arrived.
	shortCtx, cancelShort := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancelShort()
	if err := wsjson.Read(shortCtx, peerBConn, &received); err == nil {
		t.Fatalf("peerB should not have received anything, got %+v", received)
	}
	_ = peerB
}

// TestLeave_BroadcastsAndClosesHost proves a disconnect is noticed and
// broadcast, and that the host disconnecting closes the room for everyone
// else.
func TestLeave_BroadcastsAndClosesHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ts, closeTS := newTestServer(t)
	defer closeTS()

	hostConn, host := dial(t, ctx, ts, "")

	peerAConn, _ := dial(t, ctx, ts, host.Room)
	defer peerAConn.CloseNow()
	var notice signaling.Envelope
	_ = wsjson.Read(ctx, hostConn, &notice) // drain peerA's join notice on host

	// Host disconnects -- peerA should get room-closed.
	hostConn.CloseNow()

	if err := wsjson.Read(ctx, peerAConn, &notice); err != nil {
		t.Fatalf("peerA read room-closed: %v", err)
	}
	if notice.Type != signaling.TypeRoomClosed {
		t.Fatalf("notice type = %q, want %q", notice.Type, signaling.TypeRoomClosed)
	}
}
