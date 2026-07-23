export type PendingPaymentSession = {
  paymentKey?: string
  orderId?: string
  bizType?: string
  bizRefUlid?: string
  source?: string
  returnPath?: string
}

export const PENDING_PAYMENT_SESSION_KEY = "pending_payment_session"

const PAYMENT_RETURN_VALIDATION_ORIGIN = "https://payment-return.invalid"

function internalPaymentReturnPath(value: unknown) {
  if (typeof value !== "string") return ""
  const candidate = value.trim()
  if (!candidate.startsWith("/") || candidate.startsWith("//")) return ""

  try {
    const parsed = new URL(candidate, PAYMENT_RETURN_VALIDATION_ORIGIN)
    if (parsed.origin !== PAYMENT_RETURN_VALIDATION_ORIGIN) return ""
    return `${parsed.pathname}${parsed.search}${parsed.hash}`
  } catch {
    return ""
  }
}

export function sanitizePaymentReturnPath(value: unknown, fallback = "") {
  return internalPaymentReturnPath(value) || internalPaymentReturnPath(fallback)
}

export function readPendingPaymentSession(): PendingPaymentSession | null {
  if (typeof window === "undefined") return null
  const raw = window.localStorage.getItem(PENDING_PAYMENT_SESSION_KEY)
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as PendingPaymentSession
    if (!parsed) return null
    if (parsed.paymentKey !== undefined && typeof parsed.paymentKey !== "string") return null
    if (parsed.returnPath !== undefined && typeof parsed.returnPath !== "string") return null
    return {
      ...parsed,
      returnPath: sanitizePaymentReturnPath(parsed.returnPath),
    }
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
