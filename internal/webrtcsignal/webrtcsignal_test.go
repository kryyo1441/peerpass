package webrtcsignal

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"

	"peerpass/internal/signalclient"
	"peerpass/internal/signaling"
	"peerpass/internal/signalserver"
)

// handlerBox lets a Manager's HandleRelay be wired in after
// signalclient.Connect has already started its read-loop goroutine --
// Connect needs an EventHandler before the Manager (which needs the
// Client) exists. Guarded because set/call run on different goroutines.
type handlerBox struct {
	mu sync.Mutex
	f  func(signaling.Envelope)
}

func (b *handlerBox) set(f func(signaling.Envelope)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.f = f
}

func (b *handlerBox) call(env signaling.Envelope) {
	b.mu.Lock()
	f := b.f
	b.mu.Unlock()
	if f != nil {
		f(env)
	}
}

// TestOfferAnswerEstablishesDataChannel is the same-machine, two-process
// test discussed for M4: two Managers, talking only through a real
// signaling server (no shortcuts), end up with an open data channel on
// both sides. This is the thing M4 actually needs to prove works.
func TestOfferAnswerEstablishesDataChannel(t *testing.T) {
	hub := signaling.NewHub()
	ts := httptest.NewServer(signalserver.New(hub))
	defer ts.Close()
	addr := strings.TrimPrefix(ts.URL, "http://")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var hostHandler, peerHandler handlerBox

	hostOpened := make(chan *webrtc.DataChannel, 1)
	hostClient, hostWelcome, err := signalclient.Connect(ctx, addr, "", hostHandler.call, nil)
	if err != nil {
		t.Fatalf("host connect: %v", err)
	}
	defer hostClient.Close()
	hostMgr := NewManager(hostClient, func(_ signaling.PeerID, dc *webrtc.DataChannel) {
		hostOpened <- dc
	})
	hostHandler.set(func(env signaling.Envelope) {
		if env.Type == signaling.TypeRelay {
			if err := hostMgr.HandleRelay(ctx, env); err != nil {
				t.Errorf("host HandleRelay: %v", err)
			}
		}
	})

	peerOpened := make(chan *webrtc.DataChannel, 1)
	peerClient, _, err := signalclient.Connect(ctx, addr, hostWelcome.Room, peerHandler.call, nil)
	if err != nil {
		t.Fatalf("peer connect: %v", err)
	}
	defer peerClient.Close()
	peerMgr := NewManager(peerClient, func(_ signaling.PeerID, dc *webrtc.DataChannel) {
		peerOpened <- dc
	})
	peerHandler.set(func(env signaling.Envelope) {
		if env.Type == signaling.TypeRelay {
			if err := peerMgr.HandleRelay(ctx, env); err != nil {
				t.Errorf("peer HandleRelay: %v", err)
			}
		}
	})

	peerID := peerClient.PeerID
	if err := hostMgr.Offer(ctx, peerID); err != nil {
		t.Fatalf("host Offer: %v", err)
	}

	var hostDC, peerDC *webrtc.DataChannel
	select {
	case hostDC = <-hostOpened:
	case <-ctx.Done():
		t.Fatal("timed out waiting for host's data channel to open")
	}
	select {
	case peerDC = <-peerOpened:
	case <-ctx.Done():
		t.Fatal("timed out waiting for peer's data channel to open")
	}

	received := make(chan string, 1)
	peerDC.OnMessage(func(msg webrtc.DataChannelMessage) {
		received <- string(msg.Data)
	})
	if err := hostDC.SendText("hello from host"); err != nil {
		t.Fatalf("SendText: %v", err)
	}

	select {
	case got := <-received:
		if got != "hello from host" {
			t.Fatalf("got message %q, want %q", got, "hello from host")
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for data channel message")
	}
}
