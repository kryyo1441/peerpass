package signalclient

import "testing"

func TestSplitSchemeHost(t *testing.T) {
	cases := []struct {
		addr       string
		wantScheme string
		wantHost   string
	}{
		{"127.0.0.1:8080", "ws", "127.0.0.1:8080"},
		{"ws://127.0.0.1:8080", "ws", "127.0.0.1:8080"},
		{"wss://peerpass-signal.onrender.com", "wss", "peerpass-signal.onrender.com"},
	}
	for _, c := range cases {
		scheme, host := splitSchemeHost(c.addr)
		if scheme != c.wantScheme || host != c.wantHost {
			t.Errorf("splitSchemeHost(%q) = (%q, %q), want (%q, %q)", c.addr, scheme, host, c.wantScheme, c.wantHost)
		}
	}
}
