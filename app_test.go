package main

import "testing"

func TestGreet(t *testing.T) {
	app := NewApp()
	got := app.Greet("Peer")
	want := "Hello Peer, It's show time!"
	if got != want {
		t.Errorf("Greet(%q) = %q, want %q", "Peer", got, want)
	}
}
