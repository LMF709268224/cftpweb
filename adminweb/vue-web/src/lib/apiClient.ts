import { clearAuthSession } from "./authStorage"

type JsonObject = Record<string, unknown>

export class ApiError extends Error {
  status: number
  payload: unknown

  constructor(message: string, status: number, payload: unknown) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.payload = payload
  }
}

let refreshPromise: Promise<boolean> | null = null

function isJsonObject(value: unknown): value is JsonObject {
  return !!value && typeof value === "object" && !Array.isArray(value)
}

function parsePayload(text: string): unknown {
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

function payloadMessage(payload: unknown, fallback: string) {
  if (!isJsonObject(payload)) {
    return typeof payload === "string" && payload.trim() ? payload.trim().slice(0, 500) : fallback
  }
  const message = payload.message || payload.error || payload.error_code
  return typeof message === "string" && message.trim() ? message : fallback
}

function isAuthEndpoint(input: string) {
  return input.startsWith("/api/auth/")
}

function requestHeaders(init: RequestInit) {
  const headers = new Headers(init.headers)
  if (!headers.has("Content-Type") && typeof init.body === "string") {
    headers.set("Content-Type", "application/json")
  }
  return headers
}

function sendRequest(input: string, init: RequestInit) {
  return fetch(input, {
    ...init,
    credentials: init.credentials ?? "same-origin",
    headers: requestHeaders(init),
  })
}

async function refreshAccessToken() {
  if (!refreshPromise) {
    refreshPromise = fetch("/api/auth/refresh", {
      method: "POST",
      credentials: "same-origin",
      headers: { Accept: "application/json" },
    })
      .then(async (response) => {
        if (!response.ok) return false
        const payload = parsePayload(await response.text())
        if (!isJsonObject(payload) || !("code" in payload)) return false
        const code = Number(payload.code)
        return code === 200 || code === 201
      })
      .catch(() => false)
      .finally(() => {
        refreshPromise = null
      })
  }
  return refreshPromise
}

export async function apiClient<T = unknown>(input: string, init: RequestInit = {}): Promise<T> {
  let response = await sendRequest(input, init)
  if (response.status === 401 && !isAuthEndpoint(input) && await refreshAccessToken()) {
    response = await sendRequest(input, init)
  }

  const text = await response.text()
  const payload = parsePayload(text)

  if (response.status === 401) {
    const message = payloadMessage(payload, "Unauthorized")
    clearAuthSession()
    if (!isAuthEndpoint(input)) {
      window.location.href = "/login"
    }
    throw new ApiError(message, response.status, payload)
  }

  if (!response.ok) {
    const message = payloadMessage(payload, response.statusText || "Request failed")
    throw new ApiError(message, response.status, payload)
  }

  if (isJsonObject(payload) && "code" in payload) {
    const code = Number(payload.code)
    if (code !== 200 && code !== 201) {
      throw new ApiError(payloadMessage(payload, "Request failed"), code, payload)
    }
    return payload.data as T
  }

  return payload as T
}
