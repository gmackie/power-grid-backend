import { SELF } from "cloudflare:test"
import { describe, expect, it } from "vitest"
import { WsClient, type Msg } from "./helpers.ts"

const data = (m: Msg) => m.data as Record<string, any>
const payload = (m: Msg) => m.payload as Record<string, any>

describe("lobby -> game over WebSockets", () => {
  it("runs the lobby protocol and hands off to a game Durable Object", async () => {
    const alice = await WsClient.open("/ws")
    const bob = await WsClient.open("/ws")

    // CONNECT with a client-chosen session id (Go semantics)
    alice.send({ type: "CONNECT", session_id: "sess-alice", data: { player_name: "Alice" } })
    const aliceConnected = await alice.next()
    expect(aliceConnected.type).toBe("CONNECTED")
    expect(aliceConnected.session_id).toBe("sess-alice")
    expect(data(aliceConnected)).toMatchObject({ player_name: "Alice", session_id: "sess-alice", reconnected: false, message: "New session created successfully" })
    const aliceId = data(aliceConnected)["player_id"] as string

    bob.send({ type: "CONNECT", session_id: "sess-bob", data: { player_name: "Bob" } })
    const bobId = data(await bob.next())["player_id"] as string

    // Reconnect restores the same player id
    alice.send({ type: "CONNECT", session_id: "sess-alice", data: { player_name: "Ignored" } })
    const restored = await alice.next()
    expect(data(restored)).toMatchObject({ player_id: aliceId, player_name: "Alice", reconnected: true, message: "Session restored successfully" })

    // Errors use the exact Go strings
    bob.send({ type: "CREATE_LOBBY", data: { lobby_name: "x", map_id: "mars" } })
    expect(data(await bob.next())["message"]).toBe("Invalid map selected")
    bob.send({ type: "NOPE" })
    expect(data(await bob.next())["message"]).toBe("Unknown message type")

    // LIST_MAPS comes from D1
    alice.send({ type: "LIST_MAPS", session_id: "sess-alice" })
    const maps = await alice.next()
    expect(maps.type).toBe("MAPS_LISTED")
    expect((data(maps)["maps"] as Array<{ id: string }>).map((m) => m.id).sort()).toEqual(["germany", "usa"])

    // Create lobby: creator gets LOBBY_CREATED, everyone gets LOBBIES_LISTED
    alice.send({ type: "CREATE_LOBBY", session_id: "sess-alice", data: { lobby_name: "Room", max_players: 4, map_id: "usa" } })
    const created = await alice.until("LOBBY_CREATED")
    const lobby = data(created)["lobby"] as Record<string, any>
    expect(lobby["status"]).toBe("waiting")
    expect(lobby["players"][aliceId]).toMatchObject({ is_host: true, is_ready: true, name: "Alice" })
    expect(lobby["messages"][0]["content"]).toBe("Lobby created. Waiting for players...")
    const listed = await bob.until("LOBBIES_LISTED")
    expect(listed.session_id).toBeUndefined()
    expect(data(listed)["lobbies"]).toEqual([expect.objectContaining({ id: lobby["id"], player_count: 1, max_players: 4, has_password: false })])

    // Join + ready
    bob.send({ type: "JOIN_LOBBY", session_id: "sess-bob", data: { lobby_id: lobby["id"] } })
    const joined = await bob.until("LOBBY_JOINED")
    expect(Object.keys(data(joined)["lobby"]["players"])).toHaveLength(2)
    const updated = await alice.until("LOBBY_UPDATED")
    expect(data(updated)["lobby"]["messages"].at(-1)["content"]).toBe("Bob joined the lobby")

    bob.send({ type: "START_GAME", session_id: "sess-bob" })
    expect(data(await bob.until("ERROR"))["message"]).toBe("Only the host can start the game")
    alice.send({ type: "START_GAME", session_id: "sess-alice" })
    expect(data(await alice.until("ERROR"))["message"]).toBe("Cannot start the game")

    bob.send({ type: "SET_READY", session_id: "sess-bob", data: { ready: true } })
    expect(data(await bob.until("READY_UPDATED"))).toEqual({ player_id: bobId, ready: true })

    alice.send({ type: "START_GAME", session_id: "sess-alice" })
    const starting = await alice.until("GAME_STARTING")
    const gameId = data(starting)["game_id"] as string
    expect(gameId).toMatch(/[0-9a-f-]{36}/)
    expect(data(starting)["game_url"]).toBe(`/game?game_id=${gameId}`)
    await bob.until("GAME_STARTING")

    // --- game socket, direct route ---
    const ga = await WsClient.open(`/game?game_id=${gameId}`)
    ga.send({ type: "CONNECT", payload: { player_id: aliceId, player_name: "Alice", game_id: gameId } })
    const state0 = await ga.until("GAME_STATE")
    expect(payload(state0)).toMatchObject({ game_id: gameId, status: "LOBBY", current_phase: "PLAYER_ORDER", current_round: 1 })
    expect(Object.keys(payload(state0)["players"]).sort()).toEqual([aliceId, bobId].sort())
    expect(payload(state0)["map"]["name"]).toBe("United States")
    expect(payload(state0)["market"]["resources"]["Coal"]).toHaveLength(9)
    await ga.until("PLAYER_JOINED")

    // --- game socket through the bare /game relay (LÖVE client path) ---
    const gb = await WsClient.open("/game")
    gb.send({ type: "CONNECT", payload: { player_id: bobId, player_name: "Bob", game_id: gameId } })
    expect(payload(await gb.until("GAME_STATE"))["game_id"]).toBe(gameId)
    await ga.until("PLAYER_JOINED")

    // Start the game; both sockets see phase + state
    ga.send({ type: "START_GAME" })
    const phase = await ga.until("PHASE_CHANGE")
    expect(payload(phase)["phase"]).toBe("PLAYER_ORDER")
    const playing = await ga.until("GAME_STATE")
    expect(payload(playing)["status"]).toBe("PLAYING")
    expect(payload(playing)["current_phase"]).toBe("AUCTION")
    expect(payload(playing)["power_plants"]).toHaveLength(8)
    expect(payload(playing)["plant_market"]["current"].map((p: any) => p.id)).toEqual([3, 4, 5, 6])
    const viaRelay = await gb.until("GAME_STATE")
    expect(payload(viaRelay)["status"]).toBe("PLAYING")

    // Out-of-turn bid gets the Go error string; in-turn bid opens an auction
    const turnOrder = payload(playing)["turn_order"] as Array<string>
    const first = turnOrder[0]!
    const [firstSock, secondSock] = first === aliceId ? [ga, gb] : [gb, ga]
    secondSock.send({ type: "BID_PLANT", payload: { plant_id: 3, bid: 3 } })
    const err = await secondSock.until("ERROR")
    expect(payload(err)).toEqual({ code: "MESSAGE_ERROR", message: "not your turn" })

    firstSock.send({ type: "BID_PLANT", payload: { plant_id: 4, bid: 4 } })
    const auctionState = await firstSock.until("GAME_STATE")
    expect(payload(auctionState)["auction"]).toMatchObject({ plant_id: 4, current_bid: 4, nominated_by: first })
    secondSock.send({ type: "BID_PLANT", payload: { plant_id: 4, bid: 0 } }) // pass -> first wins plant 4
    const afterWin = await secondSock.until("PLANT_BOUGHT")
    expect(payload(afterWin)).toMatchObject({ player_id: first, plant_id: 4, price: 4 })
    const won = await secondSock.until("GAME_STATE")
    expect(payload(won)["players"][first]["power_plants"]).toEqual([{ id: 4, cost: 4, capacity: 1, resource_type: "Coal", resource_cost: 2 }])
    expect(payload(won)["players"][first]["money"]).toBe(46)

    // Records landed in D1 via the HTTP API
    const live = await SELF.fetch(`http://example.com/api/games/${gameId}/live`)
    expect(live.status).toBe(200)
    const rec = (await (await SELF.fetch(`http://example.com/api/games/${gameId}`)).json()) as Record<string, any>
    expect(rec["status"]).toBe("playing")
    expect(rec["participants"]).toHaveLength(2)
    expect(rec["events"].map((e: any) => e.event_type)).toContain("game_started")
    const players = (await (await SELF.fetch("http://example.com/api/players")).json()) as { players: Array<{ name: string }> }
    expect(players.players.map((p) => p.name).sort()).toEqual(["Alice", "Bob"])

    // Spectator feed needs the admin token
    const denied = await SELF.fetch(`http://example.com/ws/admin/game/${gameId}`, { headers: { Upgrade: "websocket" } })
    expect(denied.status).toBe(401)
    const spectator = await WsClient.open(`/ws/admin/game/${gameId}?token=test-token`)
    expect(payload(await spectator.until("GAME_STATE"))["game_id"]).toBe(gameId)

    for (const c of [alice, bob, ga, gb, spectator]) c.close()
  })

  it("answers PING on both sockets and rejects garbage", async () => {
    const g = await WsClient.open("/game?game_id=nonexistent")
    g.send({ type: "PING" })
    expect((await g.next()).type).toBe("PONG")
    g.ws.send("not json")
    expect(payload(await g.next())).toEqual({ code: "INVALID_MESSAGE", message: "Could not parse message" })
    g.send({ type: "CONNECT", payload: { player_id: "x", game_id: "nonexistent" } })
    expect(payload(await g.next())["message"]).toBe("game not found: nonexistent")
    g.close()

    const l = await WsClient.open("/ws")
    l.ws.send("{broken")
    expect(data(await l.next())["message"]).toBe("Invalid message format")
    l.send({ type: "CREATE_LOBBY", data: { lobby_name: "x" } })
    expect(data(await l.next())["message"]).toBe("No active session found. Please send CONNECT message first.")
    l.close()
  })
})
