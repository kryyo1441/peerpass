// Package webrtcsignal is the glue between the signaling relay
// (internal/signalclient, internal/signaling) and a peer connection
// (internal/peerconn): it defines the offer/answer/ICE-candidate messages
// that travel as opaque relay payloads, and routes them to the right peer
// connection. One per remote peer, matching the room model's N-party shape
// (docs/decisions.md ADR-002) even though M4 itself only exercises two.
package webrtcsignal

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pion/webrtc/v4"

	"peerpass/internal/peerconn"
	"peerpass/internal/signalclient"
	"peerpass/internal/signaling"
)

// Message is the Envelope.Payload shape for TypeRelay messages carrying
// WebRTC signaling. Exactly one of SDP/Candidate is set, per Kind.
type Message struct {
	Kind      string                     `json:"kind"`
	SDP       *webrtc.SessionDescription `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
}

const (
	kindOffer     = "offer"
	kindAnswer    = "answer"
	kindCandidate = "candidate"
)

// OnDataChannel is called once per remote peer when its data channel opens
// -- for the offering side, that's Pion's own channel-open event; for the
// answering side, it's the channel handed to us via OnDataChannel.
type OnDataChannel func(peer signaling.PeerID, dc *webrtc.DataChannel)

// Manager owns one peer connection per remote peer and moves SDP/ICE
// messages between them and the signaling relay.
type Manager struct {
	client        *signalclient.Client
	onDataChannel OnDataChannel

	mu    sync.Mutex
	conns map[signaling.PeerID]*webrtc.PeerConnection
}

// NewManager creates a Manager that sends/receives signaling over client.
// onDataChannel is called whenever a remote peer's data channel opens.
func NewManager(client *signalclient.Client, onDataChannel OnDataChannel) *Manager {
	return &Manager{
		client:        client,
		onDataChannel: onDataChannel,
		conns:         make(map[signaling.PeerID]*webrtc.PeerConnection),
	}
}

// Offer starts a connection to peer: creates its peer connection and data
// channel, and sends an SDP offer over the relay. Called by whichever side
// initiates -- the room host, when a new peer joins (that policy lives in
// the caller, not here).
func (m *Manager) Offer(ctx context.Context, peer signaling.PeerID) error {
	pc, err := m.newConn(peer)
	if err != nil {
		return err
	}

	dc, err := peerconn.CreateDataChannel(pc)
	if err != nil {
		return fmt.Errorf("webrtcsignal: create data channel for %s: %w", peer, err)
	}
	dc.OnOpen(func() {
		if m.onDataChannel != nil {
			m.onDataChannel(peer, dc)
		}
	})

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("webrtcsignal: create offer for %s: %w", peer, err)
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("webrtcsignal: set local description for %s: %w", peer, err)
	}

	return m.sendRelay(ctx, peer, Message{Kind: kindOffer, SDP: &offer})
}

// HandleRelay applies an incoming relay envelope to the right peer
// connection, creating one first if this is the initial offer from peer.
func (m *Manager) HandleRelay(ctx context.Context, env signaling.Envelope) error {
	var msg Message
	if err := json.Unmarshal(env.Payload, &msg); err != nil {
		return fmt.Errorf("webrtcsignal: decode relay payload from %s: %w", env.From, err)
	}

	switch msg.Kind {
	case kindOffer:
		return m.handleOffer(ctx, env.From, msg)
	case kindAnswer:
		return m.handleAnswer(env.From, msg)
	case kindCandidate:
		return m.handleCandidate(env.From, msg)
	default:
		return fmt.Errorf("webrtcsignal: unknown relay message kind %q from %s", msg.Kind, env.From)
	}
}

func (m *Manager) handleOffer(ctx context.Context, peer signaling.PeerID, msg Message) error {
	if msg.SDP == nil {
		return fmt.Errorf("webrtcsignal: offer from %s missing sdp", peer)
	}

	pc, err := m.newConn(peer)
	if err != nil {
		return err
	}
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnOpen(func() {
			if m.onDataChannel != nil {
				m.onDataChannel(peer, dc)
			}
		})
	})

	if err := pc.SetRemoteDescription(*msg.SDP); err != nil {
		return fmt.Errorf("webrtcsignal: set remote description (offer) from %s: %w", peer, err)
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("webrtcsignal: create answer for %s: %w", peer, err)
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("webrtcsignal: set local description (answer) for %s: %w", peer, err)
	}

	return m.sendRelay(ctx, peer, Message{Kind: kindAnswer, SDP: &answer})
}

func (m *Manager) handleAnswer(peer signaling.PeerID, msg Message) error {
	if msg.SDP == nil {
		return fmt.Errorf("webrtcsignal: answer from %s missing sdp", peer)
	}
	pc, ok := m.get(peer)
	if !ok {
		return fmt.Errorf("webrtcsignal: answer from %s with no pending offer", peer)
	}
	if err := pc.SetRemoteDescription(*msg.SDP); err != nil {
		return fmt.Errorf("webrtcsignal: set remote description (answer) from %s: %w", peer, err)
	}
	return nil
}

func (m *Manager) handleCandidate(peer signaling.PeerID, msg Message) error {
	if msg.Candidate == nil {
		return fmt.Errorf("webrtcsignal: candidate message from %s missing candidate", peer)
	}
	pc, ok := m.get(peer)
	if !ok {
		return fmt.Errorf("webrtcsignal: candidate from %s with no connection", peer)
	}
	if err := pc.AddICECandidate(*msg.Candidate); err != nil {
		return fmt.Errorf("webrtcsignal: add ICE candidate from %s: %w", peer, err)
	}
	return nil
}

// newConn creates peer's connection if it doesn't already exist, wiring its
// outgoing ICE candidates to the relay.
func (m *Manager) newConn(peer signaling.PeerID) (*webrtc.PeerConnection, error) {
	if pc, ok := m.get(peer); ok {
		return pc, nil
	}

	pc, err := peerconn.New()
	if err != nil {
		return nil, fmt.Errorf("webrtcsignal: create peer connection for %s: %w", peer, err)
	}
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil { // nil marks end of gathering, nothing to send
			return
		}
		init := c.ToJSON()
		// Signaling-client lifetime context, not request-scoped -- a trickled
		// candidate found after the caller's own ctx has nothing tying it to.
		_ = m.sendRelay(context.Background(), peer, Message{Kind: kindCandidate, Candidate: &init})
	})

	m.mu.Lock()
	m.conns[peer] = pc
	m.mu.Unlock()
	return pc, nil
}

func (m *Manager) get(peer signaling.PeerID) (*webrtc.PeerConnection, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pc, ok := m.conns[peer]
	return pc, ok
}

func (m *Manager) sendRelay(ctx context.Context, peer signaling.PeerID, msg Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("webrtcsignal: encode relay message for %s: %w", peer, err)
	}
	return m.client.Relay(ctx, peer, payload)
}
