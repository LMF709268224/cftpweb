import { apiClient } from "@/lib/apiClient"

const dashboardCacheKey = "candidate_dashboard_cache"
const dashboardCacheTTL = 30_000

let dashboardPromise: Promise<any> | null = null

function readCachedDashboard() {
  if (typeof window === "undefined") return null
  const cached = localStorage.getItem(dashboardCacheKey)
  if (!cached) return null
  try {
    const payload = JSON.parse(cached) as { value: any; expiresAt: number }
    if (payload.expiresAt > Date.now()) {
      return payload.value
    }
  } catch {
    localStorage.removeItem(dashboardCacheKey)
  }
  return null
}

function writeCachedDashboard(value: any) {
  if (typeof window === "undefined") return
  localStorage.setItem(dashboardCacheKey, JSON.stringify({ value, expiresAt: Date.now() + dashboardCacheTTL }))
}

export async function getCachedDashboard() {
  const cached = readCachedDashboard()
  if (cached) return cached

  if (!dashboardPromise) {
    dashboardPromise = apiClient("/api/dashboard")
      .then((dashboard) => {
        writeCachedDashboard(dashboard)
        return dashboard
      })
      .finally(() => {
        dashboardPromise = null
      })
  }

  return dashboardPromise
}
