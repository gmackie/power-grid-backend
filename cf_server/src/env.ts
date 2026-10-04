/** Worker bindings (mirrors wrangler.jsonc). */
export interface Env {
  readonly ASSETS?: Fetcher
  readonly DB: D1Database
  readonly LOBBY_HUB: DurableObjectNamespace
  readonly GAME: DurableObjectNamespace
  readonly SERVER_NAME: string
  readonly SERVER_VERSION: string
  readonly SESSION_IDLE_MINUTES: string
  readonly ADMIN_TOKEN?: string
  /** Comma-separated browser origins. Unset or "*" allows any origin (Go behaviour). */
  readonly ALLOWED_ORIGINS?: string
}
