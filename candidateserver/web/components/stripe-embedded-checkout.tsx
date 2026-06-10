"use client"

import { useEffect, useId, useState } from "react"
import { Loader2 } from "lucide-react"

type StripeEmbeddedCheckoutProps = {
  clientSecret: string
  publishableKey?: string
  loadingText: string
  missingKeyText: string
  failedText: string
}

type EmbeddedCheckoutInstance = {
  mount: (selector: string) => void
  destroy?: () => void
}

type StripeInstance = {
  initEmbeddedCheckout: (options: { fetchClientSecret: () => Promise<string> }) => Promise<EmbeddedCheckoutInstance>
}

type PublicConfigResponse = {
  code?: number
  data?: {
    stripe_publishable_key?: string
  }
}

declare global {
  interface Window {
    Stripe?: (publishableKey: string) => StripeInstance
  }
}

let stripeScriptPromise: Promise<void> | null = null
let publishableKeyPromise: Promise<string> | null = null

const loadStripeScript = () => {
  if (typeof window === "undefined") return Promise.reject(new Error("window is unavailable"))
  if (window.Stripe) return Promise.resolve()
  if (stripeScriptPromise) return stripeScriptPromise

  stripeScriptPromise = new Promise((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>('script[src="https://js.stripe.com/v3/"]')
    if (existing) {
      existing.addEventListener("load", () => resolve(), { once: true })
      existing.addEventListener("error", () => reject(new Error("Stripe.js failed to load")), { once: true })
      return
    }

    const script = document.createElement("script")
    script.src = "https://js.stripe.com/v3/"
    script.async = true
    script.onload = () => resolve()
    script.onerror = () => reject(new Error("Stripe.js failed to load"))
    document.head.appendChild(script)
  })

  return stripeScriptPromise
}

const loadPublishableKey = async () => {
  if (publishableKeyPromise) return publishableKeyPromise

  publishableKeyPromise = fetch("/api/public/config", { credentials: "include" })
    .then(async (res) => {
      if (!res.ok) return ""
      const payload = (await res.json()) as PublicConfigResponse
      return payload.data?.stripe_publishable_key?.trim() || ""
    })
    .catch(() => "")

  return publishableKeyPromise
}

export function StripeEmbeddedCheckout({
  clientSecret,
  publishableKey,
  loadingText,
  missingKeyText,
  failedText,
}: StripeEmbeddedCheckoutProps) {
  const checkoutId = useId().replace(/:/g, "")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!clientSecret) return

    let cancelled = false
    let checkout: EmbeddedCheckoutInstance | null = null

    const mountCheckout = async () => {
      setError("")
      setLoading(true)

      try {
        const runtimePublishableKey = publishableKey?.trim() || await loadPublishableKey()
        if (!runtimePublishableKey) {
          throw new Error(missingKeyText)
        }

        await loadStripeScript()
        if (cancelled) return

        const stripe = window.Stripe?.(runtimePublishableKey)
        if (!stripe) {
          throw new Error(failedText)
        }

        checkout = await stripe.initEmbeddedCheckout({
          fetchClientSecret: async () => clientSecret,
        })

        if (cancelled) {
          checkout.destroy?.()
          return
        }

        checkout.mount(`#${checkoutId}`)
      } catch (err) {
        console.error(err)
        if (!cancelled) setError(err instanceof Error ? err.message : failedText)
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void mountCheckout()

    return () => {
      cancelled = true
      checkout?.destroy?.()
    }
  }, [checkoutId, clientSecret, failedText, missingKeyText, publishableKey])

  return (
    <div className="rounded-xl border border-border bg-background p-3">
      {loading && (
        <div className="flex min-h-40 items-center justify-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          {loadingText}
        </div>
      )}
      {error && <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{error}</div>}
      <div id={checkoutId} className={loading || error ? "hidden" : "min-h-[480px]"} />
    </div>
  )
}
