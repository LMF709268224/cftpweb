import { apiClient } from "./apiClient"

const unreadCountCacheKey = "candidate_unread_count_cache"
const unreadCountCacheTTL = 30_000
const unreadCountChangedEvent = "candidate-unread-count-changed"
let unreadCountPromise: Promise<number> | null = null

function readCachedUnreadCount() {
  const cached = localStorage.getItem(unreadCountCacheKey)
  if (!cached) return null
  try {
    const payload = JSON.parse(cached) as { value: number; expiresAt: number }
    if (payload.expiresAt > Date.now()) return payload.value
  } catch {
    localStorage.removeItem(unreadCountCacheKey)
  }
  return null
}

function writeCachedUnreadCount(value: number) {
  localStorage.setItem(unreadCountCacheKey, JSON.stringify({ value, expiresAt: Date.now() + unreadCountCacheTTL }))
}

function dispatchUnreadCountChanged(value: number) {
  window.dispatchEvent(new CustomEvent(unreadCountChangedEvent, { detail: { value } }))
}

export async function getCachedUnreadCount() {
  const cached = readCachedUnreadCount()
  if (cached !== null) return cached

  if (!unreadCountPromise) {
    unreadCountPromise = apiClient("/api/messages/unread-count")
      .then((payload) => {
        const value = Number(payload?.unread_count || 0)
        writeCachedUnreadCount(value)
        return value
      })
      .finally(() => {
        unreadCountPromise = null
      })
  }

  return unreadCountPromise
}

export async function fetchUnreadCount(suppressErrorToast = false) {
  const payload = await apiClient("/api/messages/unread-count", { suppressErrorToast })
  const value = Number(payload?.unread_count || 0)
  setCachedUnreadCount(value)
  return value
}

export function setCachedUnreadCount(value: number) {
  const normalized = Math.max(0, Number.isFinite(value) ? Math.trunc(value) : 0)
  writeCachedUnreadCount(normalized)
  dispatchUnreadCountChanged(normalized)
}

export function onUnreadCountChanged(handler: (value: number) => void) {
  const listener = (event: Event) => {
    const value = Number((event as CustomEvent<{ value?: number }>).detail?.value)
    handler(Number.isFinite(value) ? value : 0)
  }
  window.addEventListener(unreadCountChangedEvent, listener)
  return () => window.removeEventListener(unreadCountChangedEvent, listener)
}
