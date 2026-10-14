<script lang="ts">
  import { CreateRoom, JoinRoom, LeaveRoom } from '../wailsjs/go/main/App.js'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'
  import type { main } from '../wailsjs/go/models'

  let signalAddr = '127.0.0.1:8080'
  let joinCode = ''
  let room: main.RoomInfo | null = null
  let peers: string[] = []
  let status = 'Not connected.'

  // The server always sends the full current peer list (not a delta), so
  // this just replaces -- no appending/filtering that could drift out of
  // sync if a notification arrived out of order or was missed.
  EventsOn('signal:peers', (updated: string[]) => {
    peers = updated
    status = `Peers in room: ${updated.length}`
  })

  // room-closed/disconnected only notify -- they don't tear down state
  // themselves, since that happens on the Go read-loop goroutine. The
  // frontend calling LeaveRoom() is what actually closes things out.
  EventsOn('signal:room-closed', async () => {
    status = 'Room closed (host left).'
    await leaveRoom(false)
  })
  EventsOn('signal:disconnected', async () => {
    status = 'Lost connection to the signaling server.'
    await leaveRoom(false)
  })

  async function createRoom(): Promise<void> {
    try {
      room = await CreateRoom(signalAddr)
      peers = room.peers
      status = `Room created: ${room.room}`
    } catch (err) {
      status = `Create failed: ${err}`
    }
  }

  async function joinRoom(): Promise<void> {
    try {
      room = await JoinRoom(signalAddr, joinCode.toUpperCase())
      peers = room.peers
      status = `Joined room: ${room.room}`
    } catch (err) {
      status = `Join failed: ${err}`
    }
  }

  // announceStatus is false when called from the room-closed/disconnected
  // handlers above, which already set a more specific status message.
  async function leaveRoom(announceStatus = true): Promise<void> {
    await LeaveRoom()
    room = null
    peers = []
    if (announceStatus) status = 'Left room.'
  }
</script>

<main>
  <h1>PeerPass</h1>
  <p class="status">{status}</p>

  {#if !room}
    <div class="panel">
      <label>
        Signaling server
        <input bind:value={signalAddr} placeholder="127.0.0.1:8080" />
      </label>
      <button on:click={createRoom}>Create Room</button>
    </div>
    <div class="panel">
      <label>
        Room code
        <input bind:value={joinCode} placeholder="ABC123" />
      </label>
      <button on:click={joinRoom} disabled={!joinCode}>Join Room</button>
    </div>
  {:else}
    <div class="panel">
      <p>Room code: <strong>{room.room}</strong> {room.host ? '(you are host)' : ''}</p>
      <p>Your peer ID: <code>{room.peerId}</code></p>
      <p>Peers in room ({peers.length}):</p>
      <ul>
        {#each peers as peer (peer)}
          <li><code>{peer}</code>{peer === room.peerId ? ' (you)' : ''}</li>
        {/each}
      </ul>
      <button on:click={() => leaveRoom()}>Leave Room</button>
    </div>
  {/if}
</main>

<style>
  main {
    max-width: 480px;
    margin: 0 auto;
    padding: 2rem 1rem;
    font-family: sans-serif;
  }

  .status {
    color: #9aa5b1;
    font-size: 0.9rem;
  }

  .panel {
    border: 1px solid #2a3647;
    border-radius: 6px;
    padding: 1rem;
    margin-bottom: 1rem;
  }

  label {
    display: block;
    margin-bottom: 0.5rem;
  }

  input {
    display: block;
    width: 100%;
    margin-top: 0.25rem;
    padding: 0.4rem;
    box-sizing: border-box;
  }

  button {
    margin-top: 0.5rem;
    padding: 0.4rem 1rem;
    cursor: pointer;
  }

  ul {
    padding-left: 1.2rem;
  }
</style>
