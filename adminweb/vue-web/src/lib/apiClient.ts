import { clearAuthSession } from "./authStorage"

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

export async function apiClient<T = unknown>(input: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)

  if (!headers.has("Content-Type") && init.body) {
    headers.set("Content-Type", "application/json")
  }
  const response = await fetch(input, {
    ...init,
    credentials: init.credentials ?? "same-origin",
    headers,
  })

  const text = await response.text()
  const payload = text ? JSON.parse(text) : null

  if (response.status === 401) {
    const message = payload?.message || payload?.error || "Unauthorized"
    clearAuthSession()
    if (!input.includes("/api/auth/login")) {
      window.location.href = "/login"
    }
    throw new ApiError(message, response.status, payload)
  }

  if (!response.ok) {
    const message = payload?.message || payload?.error || response.statusText || "Request failed"
    throw new ApiError(message, response.status, payload)
  }

  if (payload && typeof payload === "object" && "code" in payload) {
    const code = Number(payload.code)
    if (code !== 200 && code !== 201) {
      throw new ApiError(payload.message || payload.error_code || "Request failed", code, payload)
    }
    return payload.data as T
  }

  return payload as T
}
