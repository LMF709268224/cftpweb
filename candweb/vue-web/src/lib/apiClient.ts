import { toast } from "vue-sonner"
import { clearAccessToken, getAccessToken, rememberPostLoginRedirect, setAccessToken } from "./authStorage"
import { getErrorMessage, localizeApiErrorMessage } from "./errorCodes"
import { telemetry } from "./telemetry"

type ApiClientOptions = RequestInit & {
  timeoutMs?: number
  suppressErrorToast?: boolean
}

const DEFAULT_API_TIMEOUT_MS = 60000
const REFRESH_TIMEOUT_MS = 15000
const REFRESH_ENDPOINT = "/api/auth/refresh"
const LOGOUT_ENDPOINT = "/api/auth/logout"
const UNAUTHORIZED_TOAST_ID = "candidate-session-expired"
const isSilentResourceEndpoint = (endpoint: string) => /\/thumbnail-url(?:[/?#]|$)/.test(endpoint)
let refreshPromise: Promise<boolean> | null = null
let loginRedirectScheduled = false

type ApiClientErrorMeta = {
  errorCode?: string
  rawMessage?: string
  status?: number
}

export class ApiClientError extends Error {
  errorCode?: string
  rawMessage?: string
  status?: number

  constructor(message: string, meta: ApiClientErrorMeta = {}) {
    super(message)
    this.name = "ApiClientError"
    this.errorCode = meta.errorCode
    this.rawMessage = meta.rawMessage
    this.status = meta.status
  }
}

function shouldAttemptRefresh(endpoint: string) {
  try {
    const url = new URL(endpoint, window.location.origin)
    return url.origin === window.location.origin
      && url.pathname.startsWith("/api/")
      && !url.pathname.startsWith("/api/auth/")
      && !url.pathname.startsWith("/api/public/")
  } catch {
    return false
  }
}

async function performSessionRefresh() {
  const controller = new AbortController()
  const timeoutId = window.setTimeout(() => controller.abort(), REFRESH_TIMEOUT_MS)
  try {
    const response = await fetch(REFRESH_ENDPOINT, {
      method: "POST",
      credentials: "include",
      signal: controller.signal,
    })
    if (!response.ok) return false

    const payload = await response.json().catch(() => null)
    if (!payload || (payload.code !== 200 && payload.code !== 201)) return false

    const token = String(payload.data?.token || "").trim()
    if (token) setAccessToken(token)
    localStorage.setItem("is_authenticated", "true")
    return true
  } catch {
    return false
  } finally {
    window.clearTimeout(timeoutId)
  }
}

function refreshSession() {
  if (!refreshPromise) {
    refreshPromise = performSessionRefresh().finally(() => {
      refreshPromise = null
    })
  }
  return refreshPromise
}

function clearSessionAndRedirect(showErrorToast: (message: string, id?: string) => void, currentLang: "zh" | "en") {
  if (loginRedirectScheduled) return
  loginRedirectScheduled = true

  rememberPostLoginRedirect(window.location.pathname + window.location.search + window.location.hash)
  clearAccessToken()
  localStorage.removeItem("is_authenticated")
  localStorage.removeItem("user_name")
  void fetch(LOGOUT_ENDPOINT, { method: "POST", credentials: "include", keepalive: true }).catch(() => undefined)
  showErrorToast(getErrorMessage("UNAUTHORIZED", currentLang), UNAUTHORIZED_TOAST_ID)
  window.setTimeout(() => {
    window.location.href = "/login"
  }, 1500)
}

export async function apiClient(endpoint: string, options: ApiClientOptions = {}) {
  return requestApi(endpoint, options, true)
}

async function requestApi(endpoint: string, options: ApiClientOptions, allowRefresh: boolean): Promise<any> {
  const {
    timeoutMs = DEFAULT_API_TIMEOUT_MS,
    suppressErrorToast = isSilentResourceEndpoint(endpoint),
    signal,
    ...fetchOptions
  } = options
  const currentLang = (localStorage.getItem("app_lang") || "zh") as "zh" | "en"
  const headers = new Headers(options.headers)
  const token = getAccessToken()
  const controller = new AbortController()
  const timeoutId = window.setTimeout(() => controller.abort(), timeoutMs)
  const showErrorToast = (message: string, id?: string) => {
    if (!suppressErrorToast) toast.error(message, id ? { id } : undefined)
  }

  if (signal) {
    if (signal.aborted) {
      controller.abort()
    } else {
      signal.addEventListener("abort", () => controller.abort(), { once: true })
    }
  }

  if (token) headers.set("Authorization", `Bearer ${token}`)
  if (options.body && !(options.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }

  let res: Response
  try {
    res = await fetch(endpoint, {
      credentials: "include",
      ...fetchOptions,
      headers,
      signal: controller.signal,
    })
  } catch (err) {
    const isAbort = err instanceof DOMException && err.name === "AbortError"
    const errorCode = isAbort ? "REQUEST_TIMEOUT" : "NETWORK_ERROR"
    const errorMsg = getErrorMessage(errorCode, currentLang)
    showErrorToast(errorMsg)
    telemetry.track("api_error", { url: endpoint, error_code: errorCode, message: errorMsg })
    throw new ApiClientError(errorMsg, { errorCode })
  } finally {
    window.clearTimeout(timeoutId)
  }

  if (res.status === 401) {
    if (allowRefresh && shouldAttemptRefresh(endpoint) && await refreshSession()) {
      return requestApi(endpoint, options, false)
    }

    clearSessionAndRedirect(showErrorToast, currentLang)
    telemetry.track("api_error", { url: endpoint, error_code: "UNAUTHORIZED", status: res.status })
    throw new ApiClientError(getErrorMessage("UNAUTHORIZED", currentLang), { errorCode: "UNAUTHORIZED", status: res.status })
  }

  let data: any
  try {
    data = await res.json()
  } catch {
    if (!res.ok) {
      const errorMsg = getErrorMessage("UNKNOWN_ERROR", currentLang)
      showErrorToast(errorMsg)
      telemetry.track("api_error", { url: endpoint, error_code: "UNKNOWN_ERROR", status: res.status })
      throw new ApiClientError(errorMsg, { errorCode: "UNKNOWN_ERROR", status: res.status })
    }
    return res
  }

  if (!res.ok || (data.code !== 200 && data.code !== 201)) {
    const errorMsg = data.message
      ? localizeApiErrorMessage(data.error_code, data.message, currentLang)
      : getErrorMessage(data.error_code, currentLang)
    showErrorToast(errorMsg)
    telemetry.track("api_error", {
      url: endpoint,
      error_code: data.error_code,
      message: typeof data.message === "string" ? data.message : undefined,
      status: res.status,
    })
    throw new ApiClientError(errorMsg, {
      errorCode: data.error_code,
      rawMessage: typeof data.message === "string" ? data.message : undefined,
      status: res.status,
    })
  }

  return data.data
}
