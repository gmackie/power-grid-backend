/** Small deterministic PRNG (mulberry32) so engine state is replayable from its seed. */
export interface Rng {
  readonly state: number
}

export const seedRng = (seed: number): Rng => ({ state: seed >>> 0 })

export const nextFloat = (r: Rng): [number, Rng] => {
  let t = (r.state + 0x6d2b79f5) >>> 0
  const state = t
  t = Math.imul(t ^ (t >>> 15), t | 1)
  t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
  const value = ((t ^ (t >>> 14)) >>> 0) / 4294967296
  return [value, { state }]
}

export const shuffle = <A>(r: Rng, items: ReadonlyArray<A>): [Array<A>, Rng] => {
  const out = [...items]
  let rng = r
  for (let i = out.length - 1; i > 0; i--) {
    const [f, next] = nextFloat(rng)
    rng = next
    const j = Math.floor(f * (i + 1))
    const tmp = out[i]!
    out[i] = out[j]!
    out[j] = tmp
  }
  return [out, rng]
}
