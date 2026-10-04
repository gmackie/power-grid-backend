/** Resource market: tokens per price slot, finite supply, standard refill. */
import { RESOURCE_TYPES, type ResourceType } from "../protocol/game.ts"
import { INITIAL_MARKET, RESOURCE_SUPPLY, marketArrayLength, marketSlotCapacity, refillAmount } from "./rules.ts"

/** `slots[type][price]` = tokens for sale at that price (index 0 unused). Wire-identical to Go. */
export interface ResourceMarket {
  readonly slots: Readonly<Record<ResourceType, ReadonlyArray<number>>>
  /** Tokens still in the bank (not in market, not held by players). */
  readonly bank: Readonly<Record<ResourceType, number>>
}

const emptySlots = (type: ResourceType): Array<number> => new Array<number>(marketArrayLength(type)).fill(0)

/** Fill `amount` tokens into the most expensive open slots first (standard rules; the
 * initial layout coal 1-8, oil 3-8, garbage 7-8, uranium 14+16 falls out of this). */
const fillExpensiveFirst = (type: ResourceType, slots: ReadonlyArray<number>, amount: number): [Array<number>, number] => {
  const next = [...slots]
  let remaining = amount
  for (let price = next.length - 1; price >= 1 && remaining > 0; price--) {
    const cap = marketSlotCapacity(type, price)
    const room = cap - (next[price] ?? 0)
    if (room <= 0) continue
    const add = Math.min(room, remaining)
    next[price] = (next[price] ?? 0) + add
    remaining -= add
  }
  return [next, amount - remaining]
}

export const initialMarket = (initial: Partial<Record<ResourceType, number>> = {}): ResourceMarket => {
  const slots = {} as Record<ResourceType, ReadonlyArray<number>>
  const bank = {} as Record<ResourceType, number>
  for (const type of RESOURCE_TYPES) {
    const want = Math.min(initial[type] ?? INITIAL_MARKET[type], RESOURCE_SUPPLY[type])
    const [s, placed] = fillExpensiveFirst(type, emptySlots(type), want)
    slots[type] = s
    bank[type] = RESOURCE_SUPPLY[type] - placed
  }
  return { slots, bank }
}

export const available = (m: ResourceMarket, type: ResourceType): number =>
  m.slots[type].reduce((a, b) => a + b, 0)

/** Price of buying `amount` tokens, cheapest first. Undefined if not enough available. */
export const purchaseCost = (m: ResourceMarket, type: ResourceType, amount: number): number | undefined => {
  if (amount <= 0) return 0
  let remaining = amount
  let cost = 0
  const slots = m.slots[type]
  for (let price = 1; price < slots.length && remaining > 0; price++) {
    const take = Math.min(slots[price] ?? 0, remaining)
    cost += take * price
    remaining -= take
  }
  return remaining > 0 ? undefined : cost
}

/** Remove `amount` tokens, cheapest first (the ones that were paid for). */
export const removeTokens = (m: ResourceMarket, type: ResourceType, amount: number): ResourceMarket => {
  const next = [...m.slots[type]]
  let remaining = amount
  for (let price = 1; price < next.length && remaining > 0; price++) {
    const take = Math.min(next[price] ?? 0, remaining)
    next[price] = (next[price] ?? 0) - take
    remaining -= take
  }
  return { ...m, slots: { ...m.slots, [type]: next } }
}

/** Resources consumed by power plants go back to the bank. */
export const returnToBank = (m: ResourceMarket, used: Partial<Record<ResourceType, number>>): ResourceMarket => {
  const bank = { ...m.bank }
  for (const type of RESOURCE_TYPES) bank[type] += used[type] ?? 0
  return { ...m, bank }
}

/** Bureaucracy refill: standard table, limited by tokens in the bank, most expensive slots first. */
export const refill = (m: ResourceMarket, step: number, players: number): ResourceMarket => {
  const slots = { ...m.slots } as Record<ResourceType, ReadonlyArray<number>>
  const bank = { ...m.bank }
  for (const type of RESOURCE_TYPES) {
    const want = Math.min(refillAmount(type, step, players), bank[type])
    const [s, placed] = fillExpensiveFirst(type, slots[type], want)
    slots[type] = s
    bank[type] -= placed
  }
  return { slots, bank }
}

export const toWire = (m: ResourceMarket): Record<string, Array<number>> => {
  const out: Record<string, Array<number>> = {}
  for (const type of RESOURCE_TYPES) out[type] = [...m.slots[type]]
  return out
}
