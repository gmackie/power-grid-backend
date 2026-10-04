import { afterEach, describe, expect, it, vi } from "vitest"
import { createLobbyId } from "../../src/domain/lobby-id.ts"

afterEach(() => vi.restoreAllMocks())

describe("lobby codes", () => {
  it("retries occupied codes before allocating a new one", () => {
    vi.spyOn(crypto, "getRandomValues")
      .mockImplementationOnce(array => { (array as Uint8Array).fill(0); return array })
      .mockImplementationOnce(array => { (array as Uint8Array).fill(31); return array })
    const occupied = vi.fn((id: string) => id === "AAAAAA")
    expect(createLobbyId(occupied)).toBe("999999")
    expect(occupied).toHaveBeenCalledTimes(2)
  })
})
