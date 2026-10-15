// Package peerconn creates Pion WebRTC peer connections configured the way
// PeerPass needs them: STUN wired in from the start (not retrofitted later,
// per docs/decisions.md ADR-004's STUN note) and a single data channel for
// the M4 test message. SDP/ICE exchange over the signaling relay, and the
// offer/answer/candidate routing logic, live in the signaling-glue layer
// that calls this package -- this package only builds the connection.
package peerconn

import "github.com/pion/webrtc/v4"

// DataChannelLabel is the one data channel M4 creates. A future media track
// (ADR-003) is a separate concern added alongside this, not a reason to
// restructure how the connection itself is built.
const DataChannelLabel = "peerpass-control"

// defaultICEServers is deliberately a public STUN server for now -- good
// enough to exercise ICE gathering (M4's job). Whether it's sufficient over
// a real NAT, or TURN is needed, is M5's empirical question.
var defaultICEServers = []webrtc.ICEServer{
	{URLs: []string{"stun:stun.l.google.com:19302"}},
}

// New creates a Pion peer connection with STUN configured.
func New() (*webrtc.PeerConnection, error) {
	return webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: defaultICEServers,
	})
}

// CreateDataChannel creates the one data channel this peer connection will
// use, with the shared label both sides expect. Called by whichever side is
// the offerer; the answerer gets the channel via OnDataChannel instead.
func CreateDataChannel(pc *webrtc.PeerConnection) (*webrtc.DataChannel, error) {
	return pc.CreateDataChannel(DataChannelLabel, nil)
}
