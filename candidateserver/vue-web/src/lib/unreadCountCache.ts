import { apiClient } from "./apiClient"

const unreadCountCacheKey = "candidate_unread_count_cache"
const unreadCountCacheTTL = 30_000
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
