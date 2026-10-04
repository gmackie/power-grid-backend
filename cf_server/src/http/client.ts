/** Browser-safe generated client: the contract is the only source of endpoint types. */
import { Effect } from "effect"
import { FetchHttpClient, HttpClient, HttpClientRequest } from "effect/http"
import { HttpApiClient } from "effect/http-api"
import { PowerGridApi } from "./api.ts"

export const makeClient=(baseUrl: string, token?: string) => HttpApiClient.make(PowerGridApi, {
  baseUrl,
  transformClient: (client) => token
    ? client.pipe(HttpClient.mapRequest(HttpClientRequest.bearerToken(token)))
    :client
}).pipe(Effect.provide(FetchHttpClient.layer))
