type StripeFactory = (publishableKey: string) => any

declare global {
  interface Window {
    Stripe?: StripeFactory
  }
}

const stripeScriptURL = "https://js.stripe.com/v3/"
const stripeScriptID = "stripe-js"
const stripeLoadTimeoutMs = 20_000

let stripeFactoryPromise: Promise<StripeFactory> | null = null

export function loadStripeFactory() {
  if (window.Stripe) return Promise.resolve(window.Stripe)
  if (stripeFactoryPromise) return stripeFactoryPromise

  stripeFactoryPromise = new Promise<StripeFactory>((resolve, reject) => {
    let script = document.getElementById(stripeScriptID) as HTMLScriptElement | null
    const shouldAppendScript = !script
    if (!script) {
      script = document.createElement("script")
      script.id = stripeScriptID
      script.src = stripeScriptURL
      script.async = true
    }

    const finish = (error?: Error) => {
      window.clearTimeout(timeoutID)
      script?.removeEventListener("load", handleLoad)
      script?.removeEventListener("error", handleError)
      if (error) {
        script?.remove()
        stripeFactoryPromise = null
        reject(error)
        return
      }
      resolve(window.Stripe as StripeFactory)
    }
    const handleLoad = () => {
      if (window.Stripe) {
        finish()
        return
      }
      finish(new Error("Stripe.js loaded without exposing Stripe"))
    }
    const handleError = () => finish(new Error("Failed to load Stripe.js"))
    const timeoutID = window.setTimeout(
      () => finish(new Error("Timed out loading Stripe.js")),
      stripeLoadTimeoutMs,
    )

    script.addEventListener("load", handleLoad, { once: true })
    script.addEventListener("error", handleError, { once: true })
    if (shouldAppendScript) document.head.appendChild(script)
  })

  return stripeFactoryPromise
}
