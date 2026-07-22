import { onBeforeUnmount } from "vue"

const DEFAULT_POLL_INTERVAL_MS = 30_000

type PollingOptions = {
  intervalMs?: number
  immediate?: boolean
  shouldPoll?: () => boolean
}

export function usePolling(task: () => void | Promise<void>, options: PollingOptions = {}) {
  const intervalMs = options.intervalMs ?? DEFAULT_POLL_INTERVAL_MS
  let timerId: number | null = null
  let stopped = false
  let running = false

  function clearTimer() {
    if (timerId !== null) {
      window.clearTimeout(timerId)
      timerId = null
    }
  }

  function schedule() {
    clearTimer()
    if (stopped) return
    timerId = window.setTimeout(runInBackground, intervalMs)
  }

  async function run() {
    if (stopped) return
    if (running) {
      schedule()
      return
    }
    if (document.visibilityState !== "visible" || options.shouldPoll?.() === false) {
      schedule()
      return
    }
    running = true
    try {
      await task()
    } finally {
      running = false
      schedule()
    }
  }

  function runInBackground() {
    void run().catch(() => {
      // Polling is best-effort; callers that need the error can await run() directly.
    })
  }

  function start() {
    stopped = false
    if (options.immediate) runInBackground()
    else schedule()
  }

  function stop() {
    stopped = true
    clearTimer()
  }

  onBeforeUnmount(stop)

  return { start, stop, run }
}
