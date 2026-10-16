// Command signal runs the PeerPass signaling server standalone, for local
// development and manual testing. In Milestone 9+ this also needs auth;
// until then it binds to localhost by default so it isn't reachable from
// the network without deliberately opting in with -addr (see
// docs/decisions.md ADR-005).
package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"peerpass/internal/signaling"
	"peerpass/internal/signalserver"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "address to listen on")
	flag.Parse()

	// Hosting platforms (Render, etc.) assign a port via $PORT and expect a
	// bind on 0.0.0.0, not localhost -- when set, it overrides -addr. Local
	// dev leaves $PORT unset, so the ADR-005 localhost-only default is
	// unchanged unless this is deliberately deployed.
	listenAddr := *addr
	if port := os.Getenv("PORT"); port != "" {
		listenAddr = "0.0.0.0:" + port
	}

	hub := signaling.NewHub()
	srv := signalserver.New(hub)

	log.Printf("signaling server listening on %s (ws endpoint: /ws)", listenAddr)
	if err := http.ListenAndServe(listenAddr, srv); err != nil {
		log.Fatal(err)
	}
}
