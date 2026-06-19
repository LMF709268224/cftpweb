export type PendingPaymentSession = {
  paymentKey?: string
  orderId?: string
  bizType?: string
  bizRefUlid?: string
  source?: string
  returnPath?: string
}

export const PENDING_PAYMENT_SESSION_KEY = "pending_payment_session"

export function storePendingPaymentSession(session: PendingPaymentSession) {
  if (typeof window === "undefined") return
  window.localStorage.setItem(PENDING_PAYMENT_SESSION_KEY, JSON.stringify(session))
}

export function readPendingPaymentSession(): PendingPaymentSession | null {
  if (typeof window === "undefined") return null
  const raw = window.localStorage.getItem(PENDING_PAYMENT_SESSION_KEY)
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as PendingPaymentSession
    if (!parsed) return null
    if (parsed.paymentKey !== undefined && typeof parsed.paymentKey !== "string") return null
    return parsed
  } catch {
    return null
  }
}

export function clearPendingPaymentSession() {
  if (typeof window === "undefined") return
  window.localStorage.removeItem(PENDING_PAYMENT_SESSION_KEY)
}

export function stripeCheckoutUrl(paymentKey: unknown) {
  if (typeof paymentKey !== "string") return ""
  const value = paymentKey.trim()
  if (!value) return ""
  if (/^https:\/\/checkout\.stripe\.com\//i.test(value)) return value
  if (value.startsWith("/c/pay/")) return `https://checkout.stripe.com${value}`
  return ""
}

export function stripeEmbeddedClientSecret(paymentKey: unknown) {
  if (typeof paymentKey !== "string") return ""
  const value = paymentKey.trim()
  return value.startsWith("cs_") ? value : ""
}

export function openPaymentBridge(session: PendingPaymentSession) {
  if (typeof window === "undefined") return false
  storePendingPaymentSession(session)
  const bridgeUrl = "/payment-bridge"
  const popup = window.open(bridgeUrl, "_blank", "noopener,noreferrer")
  if (!popup) {
    window.location.assign(bridgeUrl)
    return false
  }
  return true
}
