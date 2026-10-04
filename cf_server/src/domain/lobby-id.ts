/** Six readable characters, sampled uniformly from a 32-character alphabet. */
const ALPHABET = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

export const createLobbyId = (isTaken: (id: string) => boolean): string => {
  for (;;) {
    const bytes = crypto.getRandomValues(new Uint8Array(6))
    const id = Array.from(bytes, byte => ALPHABET[byte & 31]).join("")
    if (!isTaken(id)) return id
  }
}
