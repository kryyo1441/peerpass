package signaling

import (
	"errors"
	"testing"
)

func TestCreateRoom(t *testing.T) {
	h := NewHub()
	code, hostID, peer := h.CreateRoom()

	if code == "" {
		t.Fatal("expected non-empty room code")
	}
	if hostID == "" {
		t.Fatal("expected non-empty host peer ID")
	}
	if peer.ID != hostID {
		t.Fatalf("peer.ID = %q, want %q", peer.ID, hostID)
	}
}

func TestJoinRoom(t *testing.T) {
	h := NewHub()
	code, hostID, hostPeer := h.CreateRoom()

	joinedID, _, peerList, err := h.JoinRoom(code)
	if err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}
	if joinedID == hostID {
		t.Fatal("joining peer got the same ID as the host")
	}
	if len(peerList) != 2 {
		t.Fatalf("peerList = %v, want 2 entries", peerList)
	}

	select {
	case env := <-hostPeer.Send:
		if env.Type != TypePeerJoined {
			t.Fatalf("host notification type = %q, want %q", env.Type, TypePeerJoined)
		}
		if env.From != joinedID {
			t.Fatalf("host notification From = %q, want %q", env.From, joinedID)
		}
	default:
		t.Fatal("expected host to receive a peer-joined notification")
	}
}

func TestJoinRoom_NotFound(t *testing.T) {
	h := NewHub()
	_, _, _, err := h.JoinRoom("NOSUCH")
	if !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("err = %v, want ErrRoomNotFound", err)
	}
}

func TestJoinRoom_Full(t *testing.T) {
	h := NewHub()
	code, _, _ := h.CreateRoom()

	for i := 1; i < MaxPeersPerRoom; i++ {
		if _, _, _, err := h.JoinRoom(code); err != nil {
			t.Fatalf("JoinRoom #%d: %v", i, err)
		}
	}

	if _, _, _, err := h.JoinRoom(code); !errors.Is(err, ErrRoomFull) {
		t.Fatalf("err = %v, want ErrRoomFull", err)
	}
}

func TestLeave_NonHost(t *testing.T) {
	h := NewHub()
	code, hostID, hostPeer := h.CreateRoom()
	joinedID, _, _, _ := h.JoinRoom(code)
	<-hostPeer.Send // drain the peer-joined notification

	if err := h.Leave(code, joinedID); err != nil {
		t.Fatalf("Leave: %v", err)
	}

	select {
	case env := <-hostPeer.Send:
		if env.Type != TypePeerLeft {
			t.Fatalf("notification type = %q, want %q", env.Type, TypePeerLeft)
		}
	default:
		t.Fatal("expected host to receive a peer-left notification")
	}

	// Room should still exist for the host.
	if _, _, _, err := h.JoinRoom(code); err != nil {
		t.Fatalf("room should still be open after non-host left: %v", err)
	}
	_ = hostID
}

func TestLeave_HostClosesRoom(t *testing.T) {
	h := NewHub()
	code, hostID, _ := h.CreateRoom()
	_, peer, _, _ := h.JoinRoom(code)

	if err := h.Leave(code, hostID); err != nil {
		t.Fatalf("Leave: %v", err)
	}

	select {
	case env := <-peer.Send:
		if env.Type != TypeRoomClosed {
			t.Fatalf("notification type = %q, want %q", env.Type, TypeRoomClosed)
		}
	default:
		t.Fatal("expected remaining peer to receive a room-closed notification")
	}

	if _, _, _, err := h.JoinRoom(code); !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("room should be gone after host left, got err = %v", err)
	}
}

func TestLeave_UnknownRoom(t *testing.T) {
	h := NewHub()
	if err := h.Leave("NOSUCH", "whoever"); !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("err = %v, want ErrRoomNotFound", err)
	}
}

func TestRelay_Success(t *testing.T) {
	h := NewHub()
	code, hostID, _ := h.CreateRoom()
	joinedID, joinedPeer, _, _ := h.JoinRoom(code)

	payload := []byte(`{"sdp":"fake-offer"}`)
	if err := h.Relay(code, hostID, joinedID, payload); err != nil {
		t.Fatalf("Relay: %v", err)
	}

	select {
	case env := <-joinedPeer.Send:
		if env.Type != TypeRelay {
			t.Fatalf("type = %q, want %q", env.Type, TypeRelay)
		}
		if env.From != hostID {
			t.Fatalf("From = %q, want %q (server must stamp it)", env.From, hostID)
		}
		if string(env.Payload) != string(payload) {
			t.Fatalf("payload = %s, want %s", env.Payload, payload)
		}
	default:
		t.Fatal("expected recipient to receive the relayed envelope")
	}
}

func TestRelay_UnknownRecipient(t *testing.T) {
	h := NewHub()
	code, hostID, _ := h.CreateRoom()

	err := h.Relay(code, hostID, "nobody", []byte(`{}`))
	if !errors.Is(err, ErrRecipientUnknown) {
		t.Fatalf("err = %v, want ErrRecipientUnknown", err)
	}
}

func TestSend_EvictsStuckPeer(t *testing.T) {
	h := NewHub()
	code, hostID, _ := h.CreateRoom()
	_, stuckPeer, _, _ := h.JoinRoom(code)

	// Fill the stuck peer's buffer without ever draining it -- simulates a
	// transport that stopped reading from Send (dead/stuck connection).
	for i := 0; i < cap(stuckPeer.Send); i++ {
		stuckPeer.Send <- Envelope{Type: TypeRelay}
	}

	select {
	case <-stuckPeer.Done():
		t.Fatal("peer should not be evicted yet -- buffer isn't full until one more send")
	default:
	}

	// This send pushes it over the edge: the buffer is now full, so the
	// Hub must evict rather than silently drop.
	if err := h.Relay(code, hostID, stuckPeer.ID, []byte(`{}`)); err != nil {
		t.Fatalf("Relay: %v", err)
	}

	select {
	case <-stuckPeer.Done():
	default:
		t.Fatal("expected stuck peer to be evicted (Done() closed)")
	}

	// And it must actually be gone from the room, not a phantom entry.
	if err := h.Relay(code, hostID, stuckPeer.ID, []byte(`{}`)); !errors.Is(err, ErrRecipientUnknown) {
		t.Fatalf("err = %v, want ErrRecipientUnknown (evicted peer should be removed from the room)", err)
	}
}

func TestRelay_CrossRoomRejected(t *testing.T) {
	h := NewHub()
	codeA, hostA, _ := h.CreateRoom()
	_, hostB, _ := h.CreateRoom()

	// hostA tries to relay to hostB's ID, but that peer is in a
	// different room than codeA -- must be rejected even though the ID
	// exists somewhere in the hub.
	err := h.Relay(codeA, hostA, hostB, []byte(`{}`))
	if !errors.Is(err, ErrRecipientUnknown) {
		t.Fatalf("err = %v, want ErrRecipientUnknown (cross-room relay must be rejected)", err)
	}
}
