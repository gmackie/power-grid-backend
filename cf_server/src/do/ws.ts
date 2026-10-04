/** WebSocket helpers for Durable Objects using the hibernation API. */
import { nowSeconds } from "../protocol/common.ts"

export const sendJson = (ws: WebSocket, value: unknown): void => {
  try {
    ws.send(JSON.stringify(value))
  } catch {
    // socket already closed; hibernation API will deliver webSocketClose
  }
}

export const readAttachment = <A extends object>(ws: WebSocket, fallback: A): A => {
  try {
    return (ws.deserializeAttachment() as A | null) ?? fallback
  } catch {
    return fallback
  }
}

export const writeAttachment = <A extends object>(ws: WebSocket, value: A): void => {
  ws.serializeAttachment(value)
}

/** `/ws` lobby envelope. session_id omitted when empty (Go omitempty). */
export const lobbyMsg = (type: string, sessionId: string, data?: unknown) => ({
  type,
  ...(sessionId ? { session_id: sessionId } : {}),
  timestamp: nowSeconds(),
  ...(data !== undefined ? { data } : {})
})

/** `/game` envelope. */
export const gameMsg = (type: string, payload?: unknown, sessionId?: string) => ({
  type,
  timestamp: nowSeconds(),
  ...(sessionId ? { session_id: sessionId } : {}),
  ...(payload !== undefined ? { payload } : {})
})

export const upgradeResponse = (ws: WebSocket): Response =>
  new Response(null, { status: 101, webSocket: ws })

export const isUpgrade = (request: Request): boolean =>
  request.headers.get("Upgrade")?.toLowerCase() === "websocket"
