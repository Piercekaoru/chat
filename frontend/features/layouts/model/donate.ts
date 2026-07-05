/**
 * Donation config.
 *
 * Put your Solana wallet address here, or set it at build time via the
 * NEXT_PUBLIC_DONATE_SOLANA_ADDRESS environment variable (the env value wins).
 */
export const DONATE_SOLANA_ADDRESS =
  process.env.NEXT_PUBLIC_DONATE_SOLANA_ADDRESS?.trim() ||
  "BDyEDrsh2KNxVcxhkocgqgJ5Psxy6X7ffsmRPMfAw7G6"

/** Network label shown next to the address. */
export const DONATE_NETWORK_LABEL = "Solana · SOL / USDT / USDC"

/** True once a real address has been configured. */
export const DONATE_ENABLED = DONATE_SOLANA_ADDRESS !== "YOUR_SOLANA_ADDRESS_HERE"

/** Shorten an address for display, e.g. `7xKq…9fRa`. */
export function truncateAddress(address: string, head = 4, tail = 4): string {
  if (address.length <= head + tail + 1) {
    return address
  }
  return `${address.slice(0, head)}…${address.slice(-tail)}`
}
