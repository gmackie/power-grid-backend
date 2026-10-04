import { SELF } from "cloudflare:test"

export interface Msg {
  type: string
  session_id?: string
  timestamp?: number
  data?: Record<string, unknown>
  payload?: unknown
}

/** A tiny WebSocket client over SELF.fetch with a message queue. */
export class WsClient {
  private queue: Array<Msg> = []
  private waiters: Array<(m: Msg) => void> = []

  private constructor(readonly ws: WebSocket) {
    ws.addEventListener("message", (ev) => {
      const msg = JSON.parse(String(ev.data)) as Msg
      const w = this.waiters.shift()
      if (w) w(msg)
      else this.queue.push(msg)
    })
  }

  static async open(path: string): Promise<WsClient> {
    const res = await SELF.fetch(`http://example.com${path}`, { headers: { Upgrade: "websocket" } })
    if (res.status !== 101 || !res.webSocket) throw new Error(`upgrade failed: ${res.status}`)
    res.webSocket.accept()
    return new WsClient(res.webSocket)
  }

  send(msg: Msg): void {
    this.ws.send(JSON.stringify(msg))
  }

  next(timeoutMs = 3000): Promise<Msg> {
    const queued = this.queue.shift()
    if (queued) return Promise.resolve(queued)
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error("timed out waiting for message")), timeoutMs)
      this.waiters.push((m) => {
        clearTimeout(timer)
        resolve(m)
      })
    })
  }

  /** Read messages until one of the given types arrives. */
  async until(...types: Array<string>): Promise<Msg> {
    for (let i = 0; i < 50; i++) {
      const m = await this.next()
      if (types.includes(m.type)) return m
    }
    throw new Error(`never received ${types.join("|")}`)
  }

  close(): void {
    this.ws.close(1000, "done")
  }
}

export const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
