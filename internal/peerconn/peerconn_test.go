package peerconn

import "testing"

func TestNewHasSTUNConfigured(t *testing.T) {
	pc, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer pc.Close()

	servers := pc.GetConfiguration().ICEServers
	if len(servers) == 0 {
		t.Fatal("expected ICE servers to be configured at construction time, got none")
	}
}

func TestCreateDataChannel(t *testing.T) {
	pc, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer pc.Close()

	dc, err := CreateDataChannel(pc)
	if err != nil {
		t.Fatalf("CreateDataChannel: %v", err)
	}
	if dc.Label() != DataChannelLabel {
		t.Fatalf("got label %q, want %q", dc.Label(), DataChannelLabel)
	}
}
