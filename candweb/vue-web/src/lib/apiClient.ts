import { toast } from "vue-sonner"
import { clearAccessToken, getAccessToken } from "./authStorage"
import { getErrorMessage, localizeApiErrorMessage } from "./errorCodes"

type ApiClientOptions = RequestInit & {
  timeoutMs?: number
  suppressErrorToast?: boolean
}

const DEFAULT_API_TIMEOUT_MS = 60000
const UNAUTHORIZED_TOAST_ID = "candidate-session-expired"
const isSilentResourceEndpoint = (endpoint: string) => /\/thumbnail-url(?:[/?#]|$)/.test(endpoint)

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

export async function apiClient(endpoint: string, options: ApiClientOptions = {}) {
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
    const errorMsg = getErrorMessage(isAbort ? "REQUEST_TIMEOUT" : "NETWORK_ERROR", currentLang)
    showErrorToast(errorMsg)
    throw new ApiClientError(errorMsg, { errorCode: isAbort ? "REQUEST_TIMEOUT" : "NETWORK_ERROR" })
  } finally {
    window.clearTimeout(timeoutId)
  }

  if (res.status === 401) {
    clearAccessToken()
    localStorage.removeItem("is_authenticated")
    localStorage.removeItem("user_name")
    showErrorToast(getErrorMessage("UNAUTHORIZED", currentLang), UNAUTHORIZED_TOAST_ID)
    setTimeout(() => {
      window.location.href = "/login"
    }, 1500)
    throw new ApiClientError(getErrorMessage("UNAUTHORIZED", currentLang), { errorCode: "UNAUTHORIZED", status: res.status })
  }

  let data: any
  try {
    data = await res.json()
  } catch {
    if (!res.ok) {
      const errorMsg = getErrorMessage("UNKNOWN_ERROR", currentLang)
      showErrorToast(errorMsg)
      throw new ApiClientError(errorMsg, { errorCode: "UNKNOWN_ERROR", status: res.status })
    }
    return res
  }

  if (!res.ok || (data.code !== 200 && data.code !== 201)) {
    const errorMsg = data.message
      ? localizeApiErrorMessage(data.error_code, data.message, currentLang)
      : getErrorMessage(data.error_code, currentLang)
    showErrorToast(errorMsg)
    throw new ApiClientError(errorMsg, {
      errorCode: data.error_code,
      rawMessage: typeof data.message === "string" ? data.message : undefined,
      status: res.status,
    })
  }

  return data.data
}
