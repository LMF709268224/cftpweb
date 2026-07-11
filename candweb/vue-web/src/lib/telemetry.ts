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

    const events = [...this.queue]
    this.queue = []

    try {
      // If token is missing, we could fallback to public endpoint, but let's try authenticated first
      // Assuming api client handles auth headers automatically if we use fetch
      const token = localStorage.getItem("token")
      const targetEndpoint = token ? this.endpoint : this.publicEndpoint

      await fetch(targetEndpoint, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { "Authorization": `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ events }),
      })
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
    const events = [...this.queue]
    this.queue = []
    const token = localStorage.getItem("token")
    const targetEndpoint = token ? this.endpoint : this.publicEndpoint
    const blob = new Blob([JSON.stringify({ events })], { type: "application/json" })
    
    // Attempt sendBeacon, fallback to keepalive fetch
    if (navigator.sendBeacon) {
      // sendBeacon doesn't support custom headers easily, so if it needs auth, we might lose token unless stored in cookie
      // But we try our best. fetch keepalive is preferred if supported.
      try {
        fetch(targetEndpoint, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            ...(token ? { "Authorization": `Bearer ${token}` } : {}),
          },
          body: JSON.stringify({ events }),
          keepalive: true,
        }).catch(() => {
          // ignore
        })
      } catch {
        navigator.sendBeacon(targetEndpoint, blob)
      }
    }
  }
}

export const telemetry = new TelemetryClient()
