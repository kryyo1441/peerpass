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

	"peerpass/internal/signaling"
	"peerpass/internal/signalserver"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "address to listen on")
	flag.Parse()

	hub := signaling.NewHub()
	srv := signalserver.New(hub)

	log.Printf("signaling server listening on %s (ws endpoint: /ws)", *addr)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		log.Fatal(err)
	}
}
