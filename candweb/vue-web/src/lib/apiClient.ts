import { toast } from "vue-sonner"
import { clearAccessToken, getAccessToken } from "./authStorage"
import { getErrorMessage, localizeApiErrorMessage } from "./errorCodes"

type ApiClientOptions = RequestInit & {
  timeoutMs?: number
  suppressErrorToast?: boolean
}

const DEFAULT_API_TIMEOUT_MS = 60000
const isSilentResourceEndpoint = (endpoint: string) => /\/thumbnail-url(?:[/?#]|$)/.test(endpoint)

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
  const showErrorToast = (message: string) => {
    if (!suppressErrorToast) toast.error(message)
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
    const errorMsg = isAbort
      ? currentLang === "zh"
        ? "请求超时，请稍后重试"
        : "Request timed out. Please try again."
      : currentLang === "zh"
        ? "网络请求失败，请检查网络后重试"
        : "Network request failed. Please check your connection and try again."
    showErrorToast(errorMsg)
    throw new Error(errorMsg)
  } finally {
    window.clearTimeout(timeoutId)
  }

  if (res.status === 401) {
    clearAccessToken()
    localStorage.removeItem("is_authenticated")
    localStorage.removeItem("user_name")
    showErrorToast(getErrorMessage("UNAUTHORIZED", currentLang))
    setTimeout(() => {
      window.location.href = "/login"
    }, 1500)
    throw new Error("401 Unauthorized")
  }

  let data: any
  try {
    data = await res.json()
  } catch {
    if (!res.ok) {
      const errorMsg = getErrorMessage("UNKNOWN_ERROR", currentLang)
      showErrorToast(errorMsg)
      throw new Error(errorMsg)
    }
    return res
  }

  if (!res.ok || (data.code !== 200 && data.code !== 201)) {
    const errorMsg = data.message
      ? localizeApiErrorMessage(data.error_code, data.message, currentLang)
      : getErrorMessage(data.error_code, currentLang)
    showErrorToast(errorMsg)
    throw new Error(errorMsg)
  }

  return data.data
}
