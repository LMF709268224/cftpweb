export type TelemetryEvent = {
  event_name: string
  payload?: Record<string, unknown>
  timestamp?: string
  url?: string
}

class TelemetryClient {
  private queue: TelemetryEvent[] = []
  private endpoint = "/api/telemetry"
  private publicEndpoint = "/api/public/telemetry"
  private flushTimer: number | null = null
  private flushInterval = 5000 // 5 seconds
  private maxBatchSize = 100
  private isFlushing = false

  constructor() {
    if (typeof window !== "undefined") {
      window.addEventListener("beforeunload", () => this.flushSync())
      window.addEventListener("visibilitychange", () => {
        if (document.visibilityState === "hidden") {
          this.flushSync()
        }
      })
    }
  }

  public track(eventName: string, payload?: Record<string, unknown>) {
    this.queue.push({
      event_name: eventName,
      payload,
      timestamp: new Date().toISOString(),
      url: typeof window !== "undefined" ? window.location.href : "",
    })

    if (!this.flushTimer) {
      this.flushTimer = window.setTimeout(() => {
        void this.flush()
      }, this.flushInterval)
    }
  }

  private async flush() {
    if (this.isFlushing || this.queue.length === 0) return
    this.isFlushing = true
    if (this.flushTimer) {
      clearTimeout(this.flushTimer)
      this.flushTimer = null
    }

    const events = this.queue.splice(0, this.maxBatchSize)

    try {
      // If token is missing, we could fallback to public endpoint, but let's try authenticated first
      // Assuming api client handles auth headers automatically if we use fetch
      const token = localStorage.getItem("token")
      const targetEndpoint = token ? this.endpoint : this.publicEndpoint

      const response = await fetch(targetEndpoint, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { "Authorization": `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ events }),
      })
      if (!response.ok) {
        if (response.status === 400 || response.status === 413) {
          console.warn(`Telemetry batch was rejected with HTTP ${response.status}`)
          return
        }
        throw new Error(`Telemetry endpoint returned HTTP ${response.status}`)
      }
    } catch (err) {
      console.error("Telemetry flush failed", err)
      // Put events back at the start of the queue if failed
      this.queue = [...events, ...this.queue]
    } finally {
      this.isFlushing = false
      if (this.queue.length > 0 && !this.flushTimer) {
        this.flushTimer = window.setTimeout(() => {
          void this.flush()
        }, this.flushInterval)
      }
    }
  }

  private flushSync() {
    if (this.queue.length === 0) return
    const token = localStorage.getItem("token")
    const targetEndpoint = token ? this.endpoint : this.publicEndpoint

    while (this.queue.length > 0) {
      const events = this.queue.splice(0, this.maxBatchSize)
      const body = JSON.stringify({ events })
      const blob = new Blob([body], { type: "application/json" })

      try {
        fetch(targetEndpoint, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            ...(token ? { "Authorization": `Bearer ${token}` } : {}),
          },
          body,
          keepalive: true,
        }).catch(() => {
          // The page is unloading, so there is no reliable retry path.
        })
      } catch {
        navigator.sendBeacon(targetEndpoint, blob)
      }
    }
  }
}

export const telemetry = new TelemetryClient()
