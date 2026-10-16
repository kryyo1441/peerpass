# Running PeerPass on another device (M5 cross-network test)

This is the M5-specific procedure: running the app on a second, genuinely
separate device/network so the WebRTC connection actually has to cross a
real NAT, not just confirm two windows on the same LAN can talk. For
day-to-day setup/build commands, see `../README.md` — this file doesn't
repeat them, only adds what M5 needs on top.

## Prerequisites on the second device

Follow `../README.md`'s **Setup** and **Linux (Arch, ...)** sections exactly
— same Go/Node/Wails CLI versions, same `-tags webkit2_41` requirement if
it's also webkit2gtk-4.1. Other platforms (Windows/macOS) aren't verified
yet; if you're trying one of those for the first time, note what broke so
it can be written up (see the crossplay note in `plan.md`).

```sh
git clone https://github.com/kryyo1441/peerpass.git
cd peerpass
wails dev -tags webkit2_41      # or: wails build -tags webkit2_41
```

You do **not** need to run `cmd/signal` on this device — M5 points both
devices at the already-deployed signaling server instead of localhost.

## Pointing at the deployed signaling server

In the app's "Signaling server" field, on **both** devices, enter:

```
wss://peerpass-signal.onrender.com
```

This is the Render deployment recorded in `decisions.md` ADR-008. Note the
security caveat still applies (ADR-005): this server has no auth, so don't
treat the room code as a secret beyond this test.

## Running the test

1. Device A: click **Create Room**, note the room code shown.
2. Device B (the other network): enter that code, click **Join Room**.
3. Watch the "WebRTC connected peers" panel on both devices — it lists the
   other peer's id once the data channel opens, and shows the M4 test
   message once one arrives.
4. **Connected + test message shown** → STUN alone was sufficient for this
   network pairing. **Never connects** → likely a symmetric NAT/CGNAT on
   one side, meaning TURN is needed (see ADR-004's STUN/TURN note — TURN
   is deliberately not built until a real test shows it's required).

Repeat across as many real network pairings as practical (home Wi-Fi,
phone hotspot, another location's Wi-Fi, etc.) — one pairing succeeding
isn't proof the rest will, since NAT behavior varies per network/ISP.

## Recording the result

Whatever happens, write it up as a dated entry in `progress.md` (what
networks were paired, connected or not) and update ADR-004's "empirical
question" note in `decisions.md` with the actual verdict. That's what
closes M5 per `plan.md` — this file only covers running the test, not
closing out the milestone.
