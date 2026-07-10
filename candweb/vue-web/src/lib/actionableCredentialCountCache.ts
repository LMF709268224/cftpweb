import { apiClient } from "./apiClient"

const cacheKey = "candidate_actionable_credential_count_cache"
const cacheTTL = 30_000
const changedEvent = "candidate-actionable-credential-count-changed"
let fetchPromise: Promise<number> | null = null

function readCachedCount() {
  const cached = localStorage.getItem(cacheKey)
  if (!cached) return null
  try {
    const payload = JSON.parse(cached) as { value: number; expiresAt: number }
    if (payload.expiresAt > Date.now()) return payload.value
  } catch {
    localStorage.removeItem(cacheKey)
  }
  return null
}

function writeCachedCount(value: number) {
  localStorage.setItem(cacheKey, JSON.stringify({ value, expiresAt: Date.now() + cacheTTL }))
}

function dispatchCountChanged(value: number) {
  window.dispatchEvent(new CustomEvent(changedEvent, { detail: { value } }))
}

export async function getCachedActionableCredentialCount() {
  const cached = readCachedCount()
  if (cached !== null) return cached

  if (!fetchPromise) {
    fetchPromise = apiClient("/api/credentials/actionable-count")
      .then((payload) => {
        const value = Number(payload?.actionable_count || 0)
        writeCachedCount(value)
        return value
      })
      .finally(() => {
        fetchPromise = null
      })
  }

  return fetchPromise
}

export async function fetchActionableCredentialCount(suppressErrorToast = false) {
  const payload = await apiClient("/api/credentials/actionable-count", { suppressErrorToast })
  const value = Number(payload?.actionable_count || 0)
  setCachedActionableCredentialCount(value)
  return value
}

export function setCachedActionableCredentialCount(value: number) {
  const normalized = Math.max(0, Number.isFinite(value) ? Math.trunc(value) : 0)
  writeCachedCount(normalized)
  dispatchCountChanged(normalized)
}

export function onActionableCredentialCountChanged(handler: (value: number) => void) {
  const listener = (event: Event) => {
    const value = Number((event as CustomEvent<{ value?: number }>).detail?.value)
    handler(Number.isFinite(value) ? value : 0)
  }
  window.addEventListener(changedEvent, listener)
  return () => window.removeEventListener(changedEvent, listener)
}
